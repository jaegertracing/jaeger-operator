package upgrade

import (
	"context"
	"reflect"
	"strings"

	"github.com/Masterminds/semver"
	"github.com/spf13/viper"
	"go.opentelemetry.io/otel"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"

	v1 "github.com/jaegertracing/jaeger-operator/apis/v1"
	"github.com/jaegertracing/jaeger-operator/pkg/tracing"
)

// ManagedInstances finds all the Jaeger instances for the current operator and upgrades them, if necessary
func ManagedInstances(ctx context.Context, c client.Client, reader client.Reader, latestVersion string) error {
	tracer := otel.GetTracerProvider().Tracer(v1.ReconciliationTracer)
	ctx, span := tracer.Start(ctx, "ManagedInstances")
	defer span.End()

	list := &v1.JaegerList{}
	identity := viper.GetString(v1.ConfigIdentity)
	opts := []client.ListOption{
		client.MatchingLabels(map[string]string{
			v1.LabelOperatedBy: identity,
		}),
	}

	if watchNamespaces := viper.GetString(v1.ConfigWatchNamespace); watchNamespaces != v1.WatchAllNamespaces {
		for _, namespace := range strings.Split(watchNamespaces, ",") {
			nsOpts := append(opts, client.InNamespace(namespace))
			nsList := &v1.JaegerList{}
			if err := reader.List(ctx, nsList, nsOpts...); err != nil {
				return tracing.HandleError(err, span)
			}
			list.Items = append(list.Items, nsList.Items...)
		}
	} else {
		if err := reader.List(ctx, list, opts...); err != nil {
			return tracing.HandleError(err, span)
		}
	}

	for _, j := range list.Items {
		// this check shouldn't have been necessary, as I'd expect the list of items to come filtered out already
		// but apparently, at least the fake client used in the unit tests doesn't filter it out... so, let's double-check
		// that we indeed own the item
		owner := j.Labels[v1.LabelOperatedBy]
		if owner != identity {
			log.Log.V(-1).Info(
				"skipping CR upgrade as we are not owners",
				"our-identity", identity,
				"owner-identity", owner,
			)

			continue
		}
		patch := client.MergeFrom(j.DeepCopy())
		jaeger, err := ManagedInstance(ctx, c, j, latestVersion)
		if err != nil {
			// nothing to do at this level, just go to the next instance
			log.Log.Error(
				err,
				"Failed to upgrade",
				"jaeger", jaeger.Name,
				"namespace", jaeger.Namespace,
				"latest-jaeger-version", latestVersion,
			)

			continue
		}
		if !reflect.DeepEqual(jaeger, j) {
			version := jaeger.Status.Version
			// the CR has changed, store it!

			if err := c.Patch(ctx, &jaeger, patch); err == nil {
				patch := client.MergeFrom(jaeger.DeepCopy())
				jaeger.Status.Version = version
				if err := c.Status().Patch(ctx, &jaeger, patch); err != nil {
					log.Log.Error(
						err,
						"failed to update status the upgraded instance",
						"instance", jaeger.Name,
						"namespace", jaeger.Namespace,
						"version", version,
					)
					tracing.HandleError(err, span)
				}
			} else {
				log.Log.Error(
					err,
					"failed to store the upgraded instance",
					"instance", jaeger.Name,
					"namespace", jaeger.Namespace,
				)
				tracing.HandleError(err, span)
			}
		}
	}

	return nil
}

// ManagedInstance performs the necessary changes to bring the given Jaeger instance to the current version
func ManagedInstance(ctx context.Context, client client.Client, jaeger v1.Jaeger, latestVersion string) (v1.Jaeger, error) {
	tracer := otel.GetTracerProvider().Tracer(v1.ReconciliationTracer)
	ctx, span := tracer.Start(ctx, "ManagedInstance")
	defer span.End()

	currentSemVersion, err := semver.NewVersion(jaeger.Status.Version)
	if err != nil {
		jaeger.Logger().Error(
			err,
			"failed to parse current Jaeger instance version. Unable to perform upgrade",
			"instance", jaeger.Name,
			"namespace", jaeger.Namespace,
			"current", jaeger.Status.Version,
		)
		return jaeger, err
	}
	latestSemVersion := semver.MustParse(latestVersion)

	if currentSemVersion.LessThan(startUpdatesVersion) {
		// We don't know how to do an upgrade from versions lower than 1.11.0
		jaeger.Logger().Error(
			err,
			"Cannot automatically upgrade from versions lower than 1.11.0",
			"instance", jaeger.Name,
			"namespace", jaeger.Namespace,
			"version", latestVersion,
			"current", jaeger.Status.Version,
		)
		return jaeger, nil
	}

	if currentSemVersion.GreaterThan(latestSemVersion) {
		// This jaeger instance has a version greater than the latest version of the operator
		jaeger.Logger().V(1).Info(
			"Jaeger instance has a version greater that the latest version",
			"instance", jaeger.Name,
			"namespace", jaeger.Namespace,
			"current", jaeger.Status.Version,
			"latest", latestVersion,
		)
		return jaeger, nil
	}

	for _, v := range semanticVersions {
		// we don't need to run the upgrade function for the version 'v', only the next ones
		if v.GreaterThan(currentSemVersion) && (v.LessThan(latestSemVersion) || v.Equal(latestSemVersion)) {
			upgraded, err := upgrades[v.String()](ctx, client, jaeger)
			if err != nil {
				log.Log.Error(
					err,
					"failed to upgrade managed instance",
					"instance", jaeger.Name,
					"namespace", jaeger.Namespace,
					"to", v.String(),
				)
				return jaeger, tracing.HandleError(err, span)
			}

			upgraded.Status.Version = v.String()
			jaeger = upgraded
		}
	}

	// Set to latestVersion
	jaeger.Status.Version = latestSemVersion.String()

	return jaeger, nil
}
