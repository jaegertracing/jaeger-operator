package jaeger

import (
	"context"

	log "github.com/sirupsen/logrus"
	rbacv1 "k8s.io/api/rbac/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/jaegertracing/jaeger-operator/pkg/apis/io/v1alpha1"
	"github.com/jaegertracing/jaeger-operator/pkg/inventory"
)

func (r *ReconcileJaeger) applyRoleBindings(jaeger v1alpha1.Jaeger, desired []rbacv1.RoleBinding) error {
	opts := client.MatchingLabels(map[string]string{
		"app.kubernetes.io/instance":   jaeger.Name,
		"app.kubernetes.io/managed-by": "jaeger-operator",
	})
	list := &rbacv1.RoleBindingList{}
	if err := r.client.List(context.Background(), opts, list); err != nil {
		return err
	}

	logFields := log.WithFields(log.Fields{
		"namespace": jaeger.Namespace,
		"instance":  jaeger.Name,
	})

	inv := inventory.ForRoleBindings(list.Items, desired)
	for _, d := range inv.Create {
		logFields.WithField("rolebinding", d.Name).Debug("creating RoleBinding")
		if err := r.client.Create(context.Background(), &d); err != nil {
			return err
		}
	}

	for _, d := range inv.Update {
		logFields.WithField("rolebinding", d.Name).Debug("updating RoleBinding")
		if err := r.client.Update(context.Background(), &d); err != nil {
			return err
		}
	}

	for _, d := range inv.Delete {
		logFields.WithField("rolebinding", d.Name).Debug("deleting RoleBinding")
		if err := r.client.Delete(context.Background(), &d); err != nil {
			return err
		}
	}

	return nil
}
