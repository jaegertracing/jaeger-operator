package jaeger

import (
	"context"

	osv1 "github.com/openshift/api/route/v1"
	"go.opentelemetry.io/otel"
	"sigs.k8s.io/controller-runtime/pkg/client"

	v1 "github.com/jaegertracing/jaeger-operator/apis/v1"
	"github.com/jaegertracing/jaeger-operator/pkg/inventory"
	"github.com/jaegertracing/jaeger-operator/pkg/tracing"
)

func (r *ReconcileJaeger) applyRoutes(ctx context.Context, jaeger v1.Jaeger, desired []osv1.Route) error {
	tracer := otel.GetTracerProvider().Tracer(v1.ReconciliationTracer)
	ctx, span := tracer.Start(ctx, "applyRoutes")
	defer span.End()

	opts := []client.ListOption{
		client.InNamespace(jaeger.Namespace),
		client.MatchingLabels(map[string]string{
			"app.kubernetes.io/instance":   jaeger.Name,
			"app.kubernetes.io/managed-by": "jaeger-operator",
		}),
	}
	list := &osv1.RouteList{}
	if err := r.rClient.List(ctx, list, opts...); err != nil {
		return tracing.HandleError(err, span)
	}

	inv := inventory.ForRoutes(list.Items, desired)
	for i := range inv.Create {
		d := inv.Create[i]
		jaeger.Logger().V(-1).Info(
			"creating route",
			"route", d.Name,
			"namespace", d.Namespace,
		)
		if err := r.client.Create(ctx, &d); err != nil {
			return tracing.HandleError(err, span)
		}
	}

	for i := range inv.Update {
		d := inv.Update[i]
		jaeger.Logger().V(-1).Info(
			"updating route",
			"route", d.Name,
			"namespace", d.Namespace,
		)
		if err := r.client.Update(ctx, &d); err != nil {
			return tracing.HandleError(err, span)
		}
	}

	for i := range inv.Delete {
		d := inv.Delete[i]
		jaeger.Logger().V(-1).Info(
			"deleting route",
			"route", d.Name,
			"namespace", d.Namespace,
		)
		if err := r.client.Delete(ctx, &d); err != nil {
			return tracing.HandleError(err, span)
		}
	}

	return nil
}
