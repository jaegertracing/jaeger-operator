package jaeger

import (
	"context"

	log "github.com/sirupsen/logrus"
	"k8s.io/api/core/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/jaegertracing/jaeger-operator/pkg/apis/io/v1alpha1"
	"github.com/jaegertracing/jaeger-operator/pkg/inventory"
)

func (r *ReconcileJaeger) applyConfigMaps(jaeger v1alpha1.Jaeger, desired []v1.ConfigMap) error {
	opts := client.MatchingLabels(map[string]string{
		"app.kubernetes.io/instance":   jaeger.Name,
		"app.kubernetes.io/managed-by": "jaeger-operator",
	})
	list := &v1.ConfigMapList{}
	if err := r.client.List(context.Background(), opts, list); err != nil {
		return err
	}

	inv := inventory.ForConfigMaps(list.Items, desired)
	for _, d := range inv.Create {
		log.WithFields(log.Fields{
			"namespace": jaeger.Namespace,
			"instance":  jaeger.Name,
			"configMap": d.Name,
		}).Debug("creating config maps")
		if err := r.client.Create(context.Background(), &d); err != nil {
			return err
		}
	}

	for _, d := range inv.Update {
		log.WithFields(log.Fields{
			"namespace": jaeger.Namespace,
			"instance":  jaeger.Name,
			"configMap": d.Name,
		}).Debug("updating config maps")
		if err := r.client.Update(context.Background(), &d); err != nil {
			return err
		}
	}

	for _, d := range inv.Delete {
		log.WithFields(log.Fields{
			"namespace": jaeger.Namespace,
			"instance":  jaeger.Name,
			"configMap": d.Name,
		}).Debug("deleting config maps")
		if err := r.client.Delete(context.Background(), &d); err != nil {
			return err
		}
	}

	return nil
}
