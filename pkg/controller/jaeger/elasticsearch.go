package jaeger

import (
	"context"

	log "github.com/sirupsen/logrus"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/jaegertracing/jaeger-operator/pkg/apis/io/v1alpha1"
	"github.com/jaegertracing/jaeger-operator/pkg/inventory"
	esv1alpha1 "github.com/jaegertracing/jaeger-operator/pkg/storage/elasticsearch/v1alpha1"
)

func (r *ReconcileJaeger) applyElasticsearches(jaeger v1alpha1.Jaeger, desired []esv1alpha1.Elasticsearch) error {
	opts := client.MatchingLabels(map[string]string{
		"app.kubernetes.io/instance":   jaeger.Name,
		"app.kubernetes.io/managed-by": "jaeger-operator",
	})
	list := &esv1alpha1.ElasticsearchList{}
	if err := r.client.List(context.Background(), opts, list); err != nil {
		return err
	}

	logFields := log.WithFields(log.Fields{
		"namespace": jaeger.Namespace,
		"instance":  jaeger.Name,
	})

	inv := inventory.ForElasticsearches(list.Items, desired)
	for _, d := range inv.Create {
		logFields.WithField("elasticsearch", d.Name).Debug("creating config maps")
		if err := r.client.Create(context.Background(), &d); err != nil {
			return err
		}
	}

	for _, d := range inv.Update {
		logFields.WithField("configMap", d.Name).Debug("updating config maps")
		if err := r.client.Update(context.Background(), &d); err != nil {
			return err
		}
	}

	for _, d := range inv.Delete {
		logFields.WithField("configMap", d.Name).Debug("deleting config maps")
		if err := r.client.Delete(context.Background(), &d); err != nil {
			return err
		}
	}

	return nil
}
