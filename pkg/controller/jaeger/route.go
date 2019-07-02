package jaeger

import (
	"context"

	osv1 "github.com/openshift/api/route/v1"
	log "github.com/sirupsen/logrus"
	"sigs.k8s.io/controller-runtime/pkg/client"

	v1 "github.com/jaegertracing/jaeger-operator/pkg/apis/jaegertracing/v1"
	"github.com/jaegertracing/jaeger-operator/pkg/inventory"
)

func (r *ReconcileJaeger) applyRoutes(jaeger v1.Jaeger, desired []osv1.Route) error {
	opts := client.InNamespace(jaeger.Namespace).MatchingLabels(map[string]string{
		"app.kubernetes.io/instance":   jaeger.Name,
		"app.kubernetes.io/managed-by": "jaeger-operator",
	})
	list := &osv1.RouteList{}
	if err := r.client.List(context.Background(), opts, list); err != nil {
		return err
	}

	inv := inventory.ForRoutes(list.Items, desired)
	for _, d := range inv.Create {
		jaeger.Logger().WithFields(log.Fields{
			"route":     d.Name,
			"namespace": d.Namespace,
		}).Debug("creating route")
		if err := r.client.Create(context.Background(), &d); err != nil {
			return err
		}
	}

	for _, d := range inv.Update {
		jaeger.Logger().WithFields(log.Fields{
			"route":     d.Name,
			"namespace": d.Namespace,
		}).Debug("updating route")
		if err := r.client.Update(context.Background(), &d); err != nil {
			return err
		}
	}

	for _, d := range inv.Delete {
		jaeger.Logger().WithFields(log.Fields{
			"route":     d.Name,
			"namespace": d.Namespace,
		}).Debug("deleting route")
		if err := r.client.Delete(context.Background(), &d); err != nil {
			return err
		}
	}

	return nil
}
