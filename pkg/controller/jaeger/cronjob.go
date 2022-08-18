package jaeger

import (
	"context"

	"github.com/spf13/viper"
	batchv1 "k8s.io/api/batch/v1"
	"k8s.io/apimachinery/pkg/runtime"

	"go.opentelemetry.io/otel"
	batchv1beta1 "k8s.io/api/batch/v1beta1"
	"sigs.k8s.io/controller-runtime/pkg/client"

	v1 "github.com/jaegertracing/jaeger-operator/apis/v1"
	"github.com/jaegertracing/jaeger-operator/pkg/inventory"
	"github.com/jaegertracing/jaeger-operator/pkg/tracing"
)

func (r *ReconcileJaeger) applyCronJobs(ctx context.Context, jaeger v1.Jaeger, desired []runtime.Object) error {
	tracer := otel.GetTracerProvider().Tracer(v1.ReconciliationTracer)
	ctx, span := tracer.Start(ctx, "applyCronJobs")
	defer span.End()

	opts := []client.ListOption{
		client.InNamespace(jaeger.Namespace),
		client.MatchingLabels(map[string]string{
			"app.kubernetes.io/instance":   jaeger.Name,
			"app.kubernetes.io/managed-by": "jaeger-operator",
		}),
	}

	cronjobsVersion := viper.GetString(v1.FlagCronJobsVersion)
	if cronjobsVersion == v1.FlagCronJobsVersionBatchV1Beta1 {
		list := &batchv1beta1.CronJobList{}
		if err := r.rClient.List(ctx, list, opts...); err != nil {
			return tracing.HandleError(err, span)
		}

		var existing []runtime.Object
		for _, i := range list.Items {
			existing = append(existing, i.DeepCopyObject())
		}

		inv := inventory.ForCronJobs(existing, desired)
		for _, d1 := range inv.Create {
			d := d1.(*batchv1beta1.CronJob)
			jaeger.Logger().V(-1).Info(
				"creating cronjob",
				"cronjob", d.Name,
				"namespace", d.Namespace,
			)
			if err := r.client.Create(ctx, d); err != nil {
				return tracing.HandleError(err, span)
			}
		}

		for _, d1 := range inv.Update {
			d := d1.(*batchv1beta1.CronJob)
			jaeger.Logger().V(-1).Info(
				"updating cronjob",
				"cronjob", d.Name,
				"namespace", d.Namespace,
			)
			if err := r.client.Update(ctx, d); err != nil {
				return tracing.HandleError(err, span)
			}
		}

		for _, d1 := range inv.Delete {
			d := d1.(*batchv1beta1.CronJob)
			jaeger.Logger().V(-1).Info(
				"deleting cronjob",
				"cronjob", d.Name,
				"namespace", d.Namespace,
			)
			if err := r.client.Delete(ctx, d); err != nil {
				return tracing.HandleError(err, span)
			}
		}
	} else {
		list := &batchv1.CronJobList{}
		if err := r.rClient.List(ctx, list, opts...); err != nil {
			return tracing.HandleError(err, span)
		}
		var existing []runtime.Object
		for _, i := range list.Items {
			var z runtime.Object = i.DeepCopyObject()
			existing = append(existing, z)
		}

		inv := inventory.ForCronJobs(existing, desired)
		for _, d1 := range inv.Create {
			d := d1.(*batchv1.CronJob)
			jaeger.Logger().V(-1).Info(
				"creating cronjob",
				"cronjob", d.Name,
				"namespace", d.Namespace,
			)
			if err := r.client.Create(ctx, d); err != nil {
				return tracing.HandleError(err, span)
			}
		}

		for _, d1 := range inv.Update {
			d := d1.(*batchv1.CronJob)
			jaeger.Logger().V(-1).Info(
				"updating cronjob",
				"cronjob", d.Name,
				"namespace", d.Namespace,
			)
			if err := r.client.Update(ctx, d); err != nil {
				return tracing.HandleError(err, span)
			}
		}

		for _, d1 := range inv.Delete {
			d := d1.(*batchv1.CronJob)
			jaeger.Logger().V(-1).Info(
				"deleting cronjob",
				"cronjob", d.Name,
				"namespace", d.Namespace,
			)
			if err := r.client.Delete(ctx, d); err != nil {
				return tracing.HandleError(err, span)
			}
		}
	}

	return nil
}
