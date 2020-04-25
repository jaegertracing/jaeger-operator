package upgrade

import (
	"context"
	"reflect"
	"sort"
	"strings"

	"github.com/Masterminds/semver"

	log "github.com/sirupsen/logrus"
	"github.com/spf13/viper"
	"go.opentelemetry.io/otel/global"
	"sigs.k8s.io/controller-runtime/pkg/client"

	v1 "github.com/jaegertracing/jaeger-operator/pkg/apis/jaegertracing/v1"
	"github.com/jaegertracing/jaeger-operator/pkg/tracing"
)

// ManagedInstances finds all the Jaeger instances for the current operator and upgrades them, if necessary
func ManagedInstances(ctx context.Context, c client.Client, reader client.Reader, latestVersion string) error {
	tracer := global.TraceProvider().GetTracer(v1.ReconciliationTracer)
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
			log.WithFields(log.Fields{
				"our-identity":   identity,
				"owner-identity": owner,
			}).Debug("skipping CR upgrade as we are not owners")
			continue
		}

		jaeger, err := ManagedInstance(ctx, c, j, latestVersion)
		if err != nil {
			// nothing to do at this level, just go to the next instance
			continue
		}

		if !reflect.DeepEqual(jaeger, j) {
			// the CR has changed, store it!
			if err := c.Update(ctx, &jaeger); err != nil {
				log.WithFields(log.Fields{
					"instance":  jaeger.Name,
					"namespace": jaeger.Namespace,
				}).WithError(err).Error("failed to store the upgraded instance")
				tracing.HandleError(err, span)
			}
		}
	}

	return nil
}

// ManagedInstance performs the necessary changes to bring the given Jaeger instance to the current version
func ManagedInstance(ctx context.Context, client client.Client, jaeger v1.Jaeger, latestVersion string) (v1.Jaeger, error) {
	tracer := global.TraceProvider().GetTracer(v1.ReconciliationTracer)
	ctx, span := tracer.Start(ctx, "ManagedInstance")
	defer span.End()
	currentSemVersion, _ := semver.NewVersion(jaeger.Status.Version)
	startSemVersion, _ := semver.NewVersion("1.11.0")
	latestSemVersion, _ := semver.NewVersion(latestVersion)

	if currentSemVersion.LessThan(startSemVersion) {
		// We don't know how to do an upgrade from versions lower than 1.11.0
		jaeger.Logger().WithFields(log.Fields{
			"instance":  jaeger.Name,
			"namespace": jaeger.Namespace,
			"to":        latestVersion,
		}).Warn("Cannot automatically upgrade from versions lower than 1.11.0")
		return jaeger, nil
	}

	versionLists := make([]*semver.Version, len(versions))
	versionIndex := 0
	for v := range versions {
		semv, err := semver.NewVersion(v)
		if err != nil {
			jaeger.Logger().WithFields(log.Fields{
				"instance":  jaeger.Name,
				"namespace": jaeger.Namespace,
				"to":        v,
			}).WithError(err).Error("Unable to parse version")
			return jaeger, err
		}
		versionLists[versionIndex] = semv
		versionIndex++
	}

	// apply the updates in order
	sort.Sort(semver.Collection(versionLists))

	for _, v := range versionLists {
		// we don't need to run the upgrade function for the version 'v', only the next ones
		if v.GreaterThan(currentSemVersion) && (v.LessThan(latestSemVersion) || v.Equal(latestSemVersion)) {
			upgraded, err := versions[v.String()].upgrade(ctx, client, jaeger)
			if err != nil {
				log.WithFields(log.Fields{
					"instance":  jaeger.Name,
					"namespace": jaeger.Namespace,
					"to":        v.String(),
				}).WithError(err).Warn("failed to upgrade managed instance")
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
