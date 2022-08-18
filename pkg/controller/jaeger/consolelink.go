package jaeger

import (
	"context"

	osconsolev1 "github.com/openshift/api/console/v1"
	"github.com/spf13/viper"
	"go.opentelemetry.io/otel"
	"sigs.k8s.io/controller-runtime/pkg/client"

	v1 "github.com/jaegertracing/jaeger-operator/apis/v1"
	"github.com/jaegertracing/jaeger-operator/pkg/inventory"
	"github.com/jaegertracing/jaeger-operator/pkg/tracing"
)

func (r *ReconcileJaeger) applyConsoleLinks(ctx context.Context, jaeger v1.Jaeger, desired []osconsolev1.ConsoleLink) error {
	tracer := otel.GetTracerProvider().Tracer(v1.ReconciliationTracer)
	ctx, span := tracer.Start(ctx, "applyConsoleLinks")
	defer span.End()

	if viper.GetString(v1.ConfigOperatorScope) != v1.OperatorScopeCluster {
		jaeger.Logger().V(-2).Info("console link skipped, operator isn't cluster-wide")
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
	for i := range inv.Create {
		d := inv.Create[i]
		jaeger.Logger().V(-1).Info(
			"creating console link",
			"consoleLink", d.Name,
			"namespace", d.Namespace,
		)
		if err := r.client.Create(ctx, &d); err != nil {
			return tracing.HandleError(err, span)
		}
	}

	for i := range inv.Update {
		d := inv.Update[i]
		jaeger.Logger().V(-1).Info(
			"updating console link",
			"consoleLink", d.Name,
			"namespace", d.Namespace,
		)
		if err := r.client.Update(ctx, &d); err != nil {
			return tracing.HandleError(err, span)
		}
	}

	for i := range inv.Delete {
		d := inv.Delete[i]
		jaeger.Logger().V(-1).Info(
			"deleting console link",
			"consoleLink", d.Name,
			"namespace", d.Namespace,
		)
		if err := r.client.Delete(ctx, &d); err != nil {
			return tracing.HandleError(err, span)
		}
	}

	return nil
}
