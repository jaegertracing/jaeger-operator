package jaeger

import (
	"context"

	appsv1 "k8s.io/api/apps/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/jaegertracing/jaeger-operator/pkg/apis/io/v1alpha1"
	"github.com/jaegertracing/jaeger-operator/pkg/inventory"
)

func (r *ReconcileJaeger) applyDaemonSets(jaeger v1alpha1.Jaeger, desired []appsv1.DaemonSet) error {
	opts := client.MatchingLabels(map[string]string{
		"app.kubernetes.io/instance":   jaeger.Name,
		"app.kubernetes.io/managed-by": "jaeger-operator",
	})
	list := &appsv1.DaemonSetList{}
	if err := r.client.List(context.Background(), opts, list); err != nil {
		return err
	}

	inv := inventory.ForDaemonSets(list.Items, desired)
	for _, d := range inv.Create {
		jaeger.Logger().WithField("daemonset", d.Name).Debug("creating daemonset")
		if err := r.client.Create(context.Background(), &d); err != nil {
			return err
		}
	}

	for _, d := range inv.Update {
		jaeger.Logger().WithField("daemonset", d.Name).Debug("updating daemonset")
		if err := r.client.Update(context.Background(), &d); err != nil {
			return err
		}
	}

	for _, d := range inv.Delete {
		jaeger.Logger().WithField("daemonset", d.Name).Debug("deleting daemonset")
		if err := r.client.Delete(context.Background(), &d); err != nil {
			return err
		}
	}

	return nil
}
