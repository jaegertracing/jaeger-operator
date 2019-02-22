package jaeger

import (
	"context"

	log "github.com/sirupsen/logrus"
	"k8s.io/api/core/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/jaegertracing/jaeger-operator/pkg/apis/io/v1alpha1"
	"github.com/jaegertracing/jaeger-operator/pkg/inventory"
)

func (r *ReconcileJaeger) applyAccounts(jaeger v1alpha1.Jaeger, desired []v1.ServiceAccount) error {
	opts := client.MatchingLabels(map[string]string{
		"app.kubernetes.io/instance":   jaeger.Name,
		"app.kubernetes.io/managed-by": "jaeger-operator",
	})
	list := &v1.ServiceAccountList{}
	if err := r.client.List(context.Background(), opts, list); err != nil {
		return err
	}

	logFields := log.WithFields(log.Fields{
		"namespace": jaeger.Namespace,
		"instance":  jaeger.Name,
	})

	inv := inventory.ForAccounts(list.Items, desired)
	for _, d := range inv.Create {
		logFields.WithField("account", d.Name).Debug("creating service account")
		if err := r.client.Create(context.Background(), &d); err != nil {
			return err
		}
	}

	for _, d := range inv.Update {
		logFields.WithField("account", d.Name).Debug("updating service account")
		if err := r.client.Update(context.Background(), &d); err != nil {
			return err
		}
	}

	for _, d := range inv.Delete {
		logFields.WithField("account", d.Name).Debug("deleting service account")
		if err := r.client.Delete(context.Background(), &d); err != nil {
			return err
		}
	}

	return nil
}
