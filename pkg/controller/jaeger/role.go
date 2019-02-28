package jaeger

import (
	"context"

	rbacv1 "k8s.io/api/rbac/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/jaegertracing/jaeger-operator/pkg/apis/io/v1alpha1"
	"github.com/jaegertracing/jaeger-operator/pkg/inventory"
)

func (r *ReconcileJaeger) applyRoles(jaeger v1alpha1.Jaeger, desired []rbacv1.Role) error {
	opts := client.MatchingLabels(map[string]string{
		"app.kubernetes.io/instance":   jaeger.Name,
		"app.kubernetes.io/managed-by": "jaeger-operator",
	})
	list := &rbacv1.RoleList{}
	if err := r.client.List(context.Background(), opts, list); err != nil {
		return err
	}

	inv := inventory.ForRoles(list.Items, desired)
	for _, d := range inv.Create {
		jaeger.Logger().WithField("role", d.Name).Debug("creating role")
		if err := r.client.Create(context.Background(), &d); err != nil {
			return err
		}
	}

	for _, d := range inv.Update {
		jaeger.Logger().WithField("role", d.Name).Debug("updating role")
		if err := r.client.Update(context.Background(), &d); err != nil {
			return err
		}
	}

	for _, d := range inv.Delete {
		jaeger.Logger().WithField("role", d.Name).Debug("deleting role")
		if err := r.client.Delete(context.Background(), &d); err != nil {
			return err
		}
	}

	return nil
}
