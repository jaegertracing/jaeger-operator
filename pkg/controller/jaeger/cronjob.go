package jaeger

import (
	"context"

	log "github.com/sirupsen/logrus"
	"go.opentelemetry.io/otel/global"
	batchv1beta1 "k8s.io/api/batch/v1beta1"
	"sigs.k8s.io/controller-runtime/pkg/client"

	v1 "github.com/jaegertracing/jaeger-operator/pkg/apis/jaegertracing/v1"
	"github.com/jaegertracing/jaeger-operator/pkg/inventory"
	"github.com/jaegertracing/jaeger-operator/pkg/tracing"
)

func (r *ReconcileJaeger) applyCronJobs(ctx context.Context, jaeger v1.Jaeger, desired []batchv1beta1.CronJob) error {
	tracer := global.TraceProvider().GetTracer(v1.ReconciliationTracer)
	ctx, span := tracer.Start(ctx, "applyCronJobs")
	defer span.End()

	opts := []client.ListOption{
		client.InNamespace(jaeger.Namespace),
		client.MatchingLabels(map[string]string{
			"app.kubernetes.io/instance":   jaeger.Name,
			"app.kubernetes.io/managed-by": "jaeger-operator",
		}),
	}
	list := &batchv1beta1.CronJobList{}
	if err := r.rClient.List(ctx, list, opts...); err != nil {
		return tracing.HandleError(err, span)
	}

	inv := inventory.ForCronJobs(list.Items, desired)
	for _, d := range inv.Create {
		jaeger.Logger().WithFields(log.Fields{
			"cronjob":   d.Name,
			"namespace": d.Namespace,
		}).Debug("creating cronjob")
		if err := r.client.Create(ctx, &d); err != nil {
			return tracing.HandleError(err, span)
		}
	}

	for _, d := range inv.Update {
		jaeger.Logger().WithFields(log.Fields{
			"cronjob":   d.Name,
			"namespace": d.Namespace,
		}).Debug("updating cronjob")
		if err := r.client.Update(ctx, &d); err != nil {
			return tracing.HandleError(err, span)
		}
	}

	for _, d := range inv.Delete {
		jaeger.Logger().WithFields(log.Fields{
			"cronjob":   d.Name,
			"namespace": d.Namespace,
		}).Debug("deleting cronjob")
		if err := r.client.Delete(ctx, &d); err != nil {
			return tracing.HandleError(err, span)
		}
	}

	return nil
}
