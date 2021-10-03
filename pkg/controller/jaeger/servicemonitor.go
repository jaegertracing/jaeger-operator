package jaeger

import (
	"context"

	monitoringv1 "github.com/prometheus-operator/prometheus-operator/pkg/apis/monitoring/v1"
	log "github.com/sirupsen/logrus"
	"go.opentelemetry.io/otel"
	"sigs.k8s.io/controller-runtime/pkg/client"

	v1 "github.com/jaegertracing/jaeger-operator/pkg/apis/jaegertracing/v1"
	"github.com/jaegertracing/jaeger-operator/pkg/inventory"
	"github.com/jaegertracing/jaeger-operator/pkg/tracing"
)

func (r *ReconcileJaeger) applyServiceMonitors(ctx context.Context, jaeger v1.Jaeger, desired []*monitoringv1.ServiceMonitor) error {
	tracer := otel.GetTracerProvider().Tracer(v1.ReconciliationTracer)
	ctx, span := tracer.Start(ctx, "applyServiceMonitors")
	defer span.End()

	opts := []client.ListOption{
		client.InNamespace(jaeger.Namespace),
		client.MatchingLabels(map[string]string{
			"app.kubernetes.io/instance":   jaeger.Name,
			"app.kubernetes.io/managed-by": "jaeger-operator",
		}),
	}
	list := &monitoringv1.ServiceMonitorList{}
	if err := r.rClient.List(ctx, list, opts...); err != nil {
		return tracing.HandleError(err, span)
	}

	inv := inventory.ForServiceMonitors(list.Items, desired)
	for _, d := range inv.Create {
		jaeger.Logger().WithFields(log.Fields{
			"service":   d.Name,
			"namespace": d.Namespace,
		}).Debug("creating servicemonitors")
		if err := r.client.Create(ctx, d); err != nil {
			return tracing.HandleError(err, span)
		}
	}

	for _, d := range inv.Update {
		jaeger.Logger().WithFields(log.Fields{
			"service":   d.Name,
			"namespace": d.Namespace,
		}).Debug("updating servicemonitors")
		if err := r.client.Update(ctx, d); err != nil {
			return tracing.HandleError(err, span)
		}
	}

	for _, d := range inv.Delete {
		jaeger.Logger().WithFields(log.Fields{
			"service":   d.Name,
			"namespace": d.Namespace,
		}).Debug("deleting servicemonitors")
		if err := r.client.Delete(ctx, d); err != nil {
			return tracing.HandleError(err, span)
		}
	}

	return nil
}