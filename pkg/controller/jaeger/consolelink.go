package jaeger

import (
	"context"

	osconsolev1 "github.com/openshift/api/console/v1"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/viper"
	"go.opentelemetry.io/otel/global"
	"sigs.k8s.io/controller-runtime/pkg/client"

	v1 "github.com/jaegertracing/jaeger-operator/pkg/apis/jaegertracing/v1"
	"github.com/jaegertracing/jaeger-operator/pkg/inventory"
	"github.com/jaegertracing/jaeger-operator/pkg/tracing"
)

func (r *ReconcileJaeger) applyConsoleLinks(ctx context.Context, jaeger v1.Jaeger, desired []osconsolev1.ConsoleLink) error {
	tracer := global.TraceProvider().GetTracer(v1.ReconciliationTracer)
	ctx, span := tracer.Start(ctx, "applyConsoleLinks")
	defer span.End()

	if viper.GetString(v1.ConfigOperatorScope) != v1.OperatorScopeCluster {
		jaeger.Logger().Trace("console link skipped, operator isn't cluster-wide")
		return nil
	}

	opts := []client.ListOption{
		client.MatchingLabels(map[string]string{
			"app.kubernetes.io/instance":   jaeger.Name,
			"app.kubernetes.io/namespace":  jaeger.Namespace,
			"app.kubernetes.io/managed-by": "jaeger-operator",
		}),
	}
	list := &osconsolev1.ConsoleLinkList{}
	if err := r.rClient.List(ctx, list, opts...); err != nil {
		return tracing.HandleError(err, span)
	}

	inv := inventory.ForConsoleLinks(list.Items, desired)
	for _, d := range inv.Create {
		jaeger.Logger().WithFields(log.Fields{
			"consoleLink": d.Name,
			"namespace":   d.Namespace,
		}).Debug("creating console link")
		if err := r.client.Create(ctx, &d); err != nil {
			return tracing.HandleError(err, span)
		}
	}

	for _, d := range inv.Update {
		jaeger.Logger().WithFields(log.Fields{
			"consoleLink": d.Name,
			"namespace":   d.Namespace,
		}).Debug("updating console link")
		if err := r.client.Update(ctx, &d); err != nil {
			return tracing.HandleError(err, span)
		}
	}

	for _, d := range inv.Delete {
		jaeger.Logger().WithFields(log.Fields{
			"consoleLink": d.Name,
			"namespace":   d.Namespace,
		}).Debug("deleting console link")
		if err := r.client.Delete(ctx, &d); err != nil {
			return tracing.HandleError(err, span)
		}
	}

	return nil
}
