package jaeger

import (
	"context"

	"go.opentelemetry.io/otel"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	autoscalingv2 "k8s.io/api/autoscaling/v2"
	autoscalingv2beta2 "k8s.io/api/autoscaling/v2beta2"

	"github.com/spf13/viper"

	v1 "github.com/jaegertracing/jaeger-operator/apis/v1"
	"github.com/jaegertracing/jaeger-operator/pkg/inventory"
	"github.com/jaegertracing/jaeger-operator/pkg/tracing"
)

func (r *ReconcileJaeger) applyHorizontalPodAutoscalers(ctx context.Context, jaeger v1.Jaeger, desired []runtime.Object) error {
	tracer := otel.GetTracerProvider().Tracer(v1.ReconciliationTracer)
	ctx, span := tracer.Start(ctx, "applyHorizontalPodAutoscalers")
	defer span.End()

	opts := []client.ListOption{
		client.InNamespace(jaeger.Namespace),
		client.MatchingLabels(map[string]string{
			"app.kubernetes.io/instance":   jaeger.Name,
			"app.kubernetes.io/managed-by": "jaeger-operator",
		}),
	}

	autoscalingVersion := viper.GetString(v1.FlagAutoscalingVersion)

	if autoscalingVersion == v1.FlagAutoscalingVersionV2Beta2 {
		hpaList := &autoscalingv2beta2.HorizontalPodAutoscalerList{}

		if err := r.rClient.List(ctx, hpaList, opts...); err != nil {
			return tracing.HandleError(err, span)
		}

		var existing []runtime.Object
		for _, i := range hpaList.Items {
			existing = append(existing, i.DeepCopyObject())
		}

		hpaInventory := inventory.ForHorizontalPodAutoscalers(existing, desired)
		for i := range hpaInventory.Create {
			d := hpaInventory.Create[i].(*autoscalingv2beta2.HorizontalPodAutoscaler)
			jaeger.Logger().V(-1).Info(
				"creating hpa",
				"hpa", d.Name,
				"namespace", d.Namespace,
			)
			if err := r.client.Create(ctx, d); err != nil {
				return tracing.HandleError(err, span)
			}
		}

		for i := range hpaInventory.Update {
			d := hpaInventory.Update[i].(*autoscalingv2beta2.HorizontalPodAutoscaler)
			jaeger.Logger().V(-1).Info(
				"updating hpa",
				"hpa", d.Name,
				"namespace", d.Namespace,
			)
			if err := r.client.Update(ctx, d); err != nil {
				return tracing.HandleError(err, span)
			}
		}

		for i := range hpaInventory.Delete {
			d := hpaInventory.Delete[i].(*autoscalingv2beta2.HorizontalPodAutoscaler)
			jaeger.Logger().V(-1).Info(
				"deleting hpa",
				"hpa", d.Name,
				"namespace", d.Namespace,
			)
			if err := r.client.Delete(ctx, d); err != nil {
				return tracing.HandleError(err, span)
			}
		}
	} else {
		hpaList := &autoscalingv2.HorizontalPodAutoscalerList{}

		if err := r.rClient.List(ctx, hpaList, opts...); err != nil {
			return tracing.HandleError(err, span)
		}

		var existing []runtime.Object
		for _, i := range hpaList.Items {
			existing = append(existing, i.DeepCopyObject())
		}

		hpaInventory := inventory.ForHorizontalPodAutoscalers(existing, desired)
		for i := range hpaInventory.Create {
			d := hpaInventory.Create[i].(*autoscalingv2.HorizontalPodAutoscaler)
			jaeger.Logger().V(-1).Info(
				"creating hpa",
				"hpa", d.Name,
				"namespace", d.Namespace,
			)
			if err := r.client.Create(ctx, d); err != nil {
				return tracing.HandleError(err, span)
			}
		}

		for i := range hpaInventory.Update {
			d := hpaInventory.Update[i].(*autoscalingv2.HorizontalPodAutoscaler)
			jaeger.Logger().V(-1).Info(
				"updating hpa",
				"hpa", d.Name,
				"namespace", d.Namespace,
			)
			if err := r.client.Update(ctx, d); err != nil {
				return tracing.HandleError(err, span)
			}
		}

		for i := range hpaInventory.Delete {
			d := hpaInventory.Delete[i].(*autoscalingv2.HorizontalPodAutoscaler)
			jaeger.Logger().V(-1).Info(
				"deleting hpa",
				"hpa", d.Name,
				"namespace", d.Namespace,
			)
			if err := r.client.Delete(ctx, d); err != nil {
				return tracing.HandleError(err, span)
			}
		}
	}

	return nil
}
