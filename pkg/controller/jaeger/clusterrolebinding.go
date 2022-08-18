package jaeger

import (
	"context"

	"github.com/spf13/viper"
	"go.opentelemetry.io/otel"
	rbac "k8s.io/api/rbac/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"

	v1 "github.com/jaegertracing/jaeger-operator/apis/v1"
	"github.com/jaegertracing/jaeger-operator/pkg/inventory"
	"github.com/jaegertracing/jaeger-operator/pkg/tracing"
)

func (r *ReconcileJaeger) applyClusterRoleBindingBindings(ctx context.Context, jaeger v1.Jaeger, desired []rbac.ClusterRoleBinding) error {
	tracer := otel.GetTracerProvider().Tracer(v1.ReconciliationTracer)
	ctx, span := tracer.Start(ctx, "applyClusterRoleBindingBindings")
	defer span.End()

	if viper.GetString(v1.ConfigOperatorScope) != v1.OperatorScopeCluster {
		jaeger.Logger().V(-2).Info("cluster role binding skipped, operator isn't cluster-wide")
		return nil
	}

	opts := client.MatchingLabels(map[string]string{
		"app.kubernetes.io/instance":   jaeger.Name,
		"app.kubernetes.io/managed-by": "jaeger-operator",
	})
	list := &rbac.ClusterRoleBindingList{}
	if err := r.rClient.List(ctx, list, opts); err != nil {
		return tracing.HandleError(err, span)
	}

	inv := inventory.ForClusterRoleBindings(list.Items, desired)
	for i := range inv.Create {
		d := inv.Create[i]
		jaeger.Logger().V(-1).Info(
			"creating cluster role binding",
			"clusteRoleBinding", d.Name,
			"namespace", d.Namespace,
		)
		if err := r.client.Create(ctx, &d); err != nil {
			return tracing.HandleError(err, span)
		}
	}

	for i := range inv.Update {
		d := inv.Update[i]
		jaeger.Logger().V(-1).Info(
			"updating cluster role binding",
			"clusteRoleBinding", d.Name,
			"namespace", d.Namespace,
		)
		if err := r.client.Update(ctx, &d); err != nil {
			return tracing.HandleError(err, span)
		}
	}

	for i := range inv.Delete {
		d := inv.Delete[i]
		jaeger.Logger().V(-1).Info(
			"deleting cluster role binding",
			"clusteRoleBinding", d.Name,
			"namespace", d.Namespace,
		)
		if err := r.client.Delete(ctx, &d); err != nil {
			return tracing.HandleError(err, span)
		}
	}

	return nil
}
