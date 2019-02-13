package jaeger

import (
	"context"

	log "github.com/sirupsen/logrus"
	"k8s.io/api/core/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/jaegertracing/jaeger-operator/pkg/apis/io/v1alpha1"
	"github.com/jaegertracing/jaeger-operator/pkg/inventory"
)

func (r *ReconcileJaeger) applyServices(jaeger v1alpha1.Jaeger, desired []v1.Service) error {
	opts := client.MatchingLabels(map[string]string{
		"app.kubernetes.io/instance":   jaeger.Name,
		"app.kubernetes.io/managed-by": "jaeger-operator",
	})
	list := &v1.ServiceList{}
	if err := r.client.List(context.Background(), opts, list); err != nil {
		return err
	}

	inv := inventory.ForServices(list.Items, desired)
	for _, d := range inv.Create {
		log.WithFields(log.Fields{
			"namespace": jaeger.Namespace,
			"instance":  jaeger.Name,
			"service":   d.Name,
		}).Debug("creating service")
		if err := r.client.Create(context.Background(), &d); err != nil {
			return err
		}
	}

	for _, d := range inv.Update {
		log.WithFields(log.Fields{
			"namespace": jaeger.Namespace,
			"instance":  jaeger.Name,
			"service":   d.Name,
		}).Debug("updating service")
		if err := r.client.Update(context.Background(), &d); err != nil {
			return err
		}
	}

	for _, d := range inv.Delete {
		log.WithFields(log.Fields{
			"namespace": jaeger.Namespace,
			"instance":  jaeger.Name,
			"service":   d.Name,
		}).Debug("deleting service")
		if err := r.client.Delete(context.Background(), &d); err != nil {
			return err
		}
	}

	return nil
}
