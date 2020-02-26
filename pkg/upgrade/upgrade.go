package upgrade

import (
	"context"
	"reflect"
	"strings"

	log "github.com/sirupsen/logrus"
	"github.com/spf13/viper"
	"go.opentelemetry.io/otel/global"
	"sigs.k8s.io/controller-runtime/pkg/client"

	v1 "github.com/jaegertracing/jaeger-operator/pkg/apis/jaegertracing/v1"
	"github.com/jaegertracing/jaeger-operator/pkg/tracing"
)

// ManagedInstances finds all the Jaeger instances for the current operator and upgrades them, if necessary
func ManagedInstances(ctx context.Context, c client.Client, reader client.Reader) error {
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

		jaeger, err := ManagedInstance(ctx, c, j)
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
func ManagedInstance(ctx context.Context, client client.Client, jaeger v1.Jaeger) (v1.Jaeger, error) {
	tracer := global.TraceProvider().GetTracer(v1.ReconciliationTracer)
	ctx, span := tracer.Start(ctx, "ManagedInstance")
	defer span.End()

	if v, ok := versions[jaeger.Status.Version]; ok {
		// we don't need to run the upgrade function for the version 'v', only the next ones
		for n := v.next; n != nil; n = n.next {
			// performs the upgrade to version 'n'
			upgraded, err := n.upgrade(ctx, client, jaeger)
			if err != nil {
				log.WithFields(log.Fields{
					"instance":  jaeger.Name,
					"namespace": jaeger.Namespace,
					"to":        n.v,
				}).WithError(err).Warn("failed to upgrade managed instance")
				return jaeger, tracing.HandleError(err, span)
			}

			upgraded.Status.Version = n.v
			jaeger = upgraded
		}
	}

	return jaeger, nil
}
