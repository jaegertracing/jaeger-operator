package jaeger

import (
	"context"

	log "github.com/sirupsen/logrus"
	"go.opentelemetry.io/otel/global"
	autoscalingv2beta2 "k8s.io/api/autoscaling/v2beta2"
	"sigs.k8s.io/controller-runtime/pkg/client"

	v1 "github.com/jaegertracing/jaeger-operator/pkg/apis/jaegertracing/v1"
	"github.com/jaegertracing/jaeger-operator/pkg/inventory"
	"github.com/jaegertracing/jaeger-operator/pkg/tracing"
)

func (r *ReconcileJaeger) applyHorizontalPodAutoscalers(ctx context.Context, jaeger v1.Jaeger, desired []autoscalingv2beta2.HorizontalPodAutoscaler) error {
	tracer := global.TraceProvider().GetTracer(v1.ReconciliationTracer)
	ctx, span := tracer.Start(ctx, "applyHorizontalPodAutoscalers")
	defer span.End()

	opts := []client.ListOption{
		client.InNamespace(jaeger.Namespace),
		client.MatchingLabels(map[string]string{
			"app.kubernetes.io/instance":   jaeger.Name,
			"app.kubernetes.io/managed-by": "jaeger-operator",
		}),
	}
	hpaList := &autoscalingv2beta2.HorizontalPodAutoscalerList{}
	if err := r.rClient.List(ctx, hpaList, opts...); err != nil {
		return tracing.HandleError(err, span)
	}

	hpaInventory := inventory.ForHorizontalPodAutoscalers(hpaList.Items, desired)
	for _, d := range hpaInventory.Create {
		jaeger.Logger().WithFields(log.Fields{
			"hpa":       d.Name,
			"namespace": d.Namespace,
		}).Debug("creating hpa")
		if err := r.client.Create(ctx, &d); err != nil {
			return tracing.HandleError(err, span)
		}
	}

	for _, d := range hpaInventory.Update {
		jaeger.Logger().WithFields(log.Fields{
			"hpa":       d.Name,
			"namespace": d.Namespace,
		}).Debug("updating hpa")
		if err := r.client.Update(ctx, &d); err != nil {
			return tracing.HandleError(err, span)
		}
	}

	for _, d := range hpaInventory.Delete {
		jaeger.Logger().WithFields(log.Fields{
			"hpa":       d.Name,
			"namespace": d.Namespace,
		}).Debug("deleting hpa")
		if err := r.client.Delete(ctx, &d); err != nil {
			return tracing.HandleError(err, span)
		}
	}

	return nil
}
