package jaeger

import (
	"context"

	"k8s.io/api/extensions/v1beta1"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/jaegertracing/jaeger-operator/pkg/apis/io/v1alpha1"
	"github.com/jaegertracing/jaeger-operator/pkg/inventory"
)

func (r *ReconcileJaeger) applyIngresses(jaeger v1alpha1.Jaeger, desired []v1beta1.Ingress) error {
	opts := client.MatchingLabels(map[string]string{
		"app.kubernetes.io/instance":   jaeger.Name,
		"app.kubernetes.io/managed-by": "jaeger-operator",
	})
	list := &v1beta1.IngressList{}
	if err := r.client.List(context.Background(), opts, list); err != nil {
		return err
	}

	inv := inventory.ForIngresses(list.Items, desired)
	for _, d := range inv.Create {
		jaeger.Logger().WithField("ingress", d.Name).Debug("creating ingress")
		if err := r.client.Create(context.Background(), &d); err != nil {
			return err
		}
	}

	for _, d := range inv.Update {
		jaeger.Logger().WithField("ingress", d.Name).Debug("updating ingress")
		if err := r.client.Update(context.Background(), &d); err != nil {
			return err
		}
	}

	for _, d := range inv.Delete {
		jaeger.Logger().WithField("ingress", d.Name).Debug("deleting ingress")
		if err := r.client.Delete(context.Background(), &d); err != nil {
			return err
		}
	}

	return nil
}
