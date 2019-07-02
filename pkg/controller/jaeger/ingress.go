package jaeger

import (
	"context"

	log "github.com/sirupsen/logrus"
	"k8s.io/api/extensions/v1beta1"
	"sigs.k8s.io/controller-runtime/pkg/client"

	v1 "github.com/jaegertracing/jaeger-operator/pkg/apis/jaegertracing/v1"
	"github.com/jaegertracing/jaeger-operator/pkg/inventory"
)

func (r *ReconcileJaeger) applyIngresses(jaeger v1.Jaeger, desired []v1beta1.Ingress) error {
	opts := client.InNamespace(jaeger.Namespace).MatchingLabels(map[string]string{
		"app.kubernetes.io/instance":   jaeger.Name,
		"app.kubernetes.io/managed-by": "jaeger-operator",
	})
	list := &v1beta1.IngressList{}
	if err := r.client.List(context.Background(), opts, list); err != nil {
		return err
	}

	inv := inventory.ForIngresses(list.Items, desired)
	for _, d := range inv.Create {
		jaeger.Logger().WithFields(log.Fields{
			"ingress":   d.Name,
			"namespace": d.Namespace,
		}).Debug("creating ingress")
		if err := r.client.Create(context.Background(), &d); err != nil {
			return err
		}
	}

	for _, d := range inv.Update {
		jaeger.Logger().WithFields(log.Fields{
			"ingress":   d.Name,
			"namespace": d.Namespace,
		}).Debug("updating ingress")
		if err := r.client.Update(context.Background(), &d); err != nil {
			return err
		}
	}

	for _, d := range inv.Delete {
		jaeger.Logger().WithFields(log.Fields{
			"ingress":   d.Name,
			"namespace": d.Namespace,
		}).Debug("deleting ingress")
		if err := r.client.Delete(context.Background(), &d); err != nil {
			return err
		}
	}

	return nil
}
