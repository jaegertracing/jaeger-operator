package jaeger

import (
	"context"

	log "github.com/sirupsen/logrus"
	batchv1beta1 "k8s.io/api/batch/v1beta1"
	"sigs.k8s.io/controller-runtime/pkg/client"

	v1 "github.com/jaegertracing/jaeger-operator/pkg/apis/jaegertracing/v1"
	"github.com/jaegertracing/jaeger-operator/pkg/inventory"
)

func (r *ReconcileJaeger) applyCronJobs(jaeger v1.Jaeger, desired []batchv1beta1.CronJob) error {
	opts := client.InNamespace(jaeger.Namespace).MatchingLabels(map[string]string{
		"app.kubernetes.io/instance":   jaeger.Name,
		"app.kubernetes.io/managed-by": "jaeger-operator",
	})
	list := &batchv1beta1.CronJobList{}
	if err := r.client.List(context.Background(), opts, list); err != nil {
		return err
	}

	inv := inventory.ForCronJobs(list.Items, desired)
	for _, d := range inv.Create {
		jaeger.Logger().WithFields(log.Fields{
			"cronjob":   d.Name,
			"namespace": d.Namespace,
		}).Debug("creating cronjob")
		if err := r.client.Create(context.Background(), &d); err != nil {
			return err
		}
	}

	for _, d := range inv.Update {
		jaeger.Logger().WithFields(log.Fields{
			"cronjob":   d.Name,
			"namespace": d.Namespace,
		}).Debug("updating cronjob")
		if err := r.client.Update(context.Background(), &d); err != nil {
			return err
		}
	}

	for _, d := range inv.Delete {
		jaeger.Logger().WithFields(log.Fields{
			"cronjob":   d.Name,
			"namespace": d.Namespace,
		}).Debug("deleting cronjob")
		if err := r.client.Delete(context.Background(), &d); err != nil {
			return err
		}
	}

	return nil
}
