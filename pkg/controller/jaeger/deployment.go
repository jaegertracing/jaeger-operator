package jaeger

import (
	"context"
	"time"

	log "github.com/sirupsen/logrus"
	appsv1 "k8s.io/api/apps/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/wait"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/jaegertracing/jaeger-operator/pkg/apis/io/v1alpha1"
	"github.com/jaegertracing/jaeger-operator/pkg/inventory"
)

func (r *ReconcileJaeger) applyDeployments(jaeger v1alpha1.Jaeger, desired []appsv1.Deployment) error {
	opts := client.MatchingLabels(map[string]string{
		"app.kubernetes.io/instance":   jaeger.Name,
		"app.kubernetes.io/managed-by": "jaeger-operator",
	})
	depList := &appsv1.DeploymentList{}
	if err := r.client.List(context.Background(), opts, depList); err != nil {
		return err
	}

	logFields := log.WithFields(log.Fields{
		"namespace": jaeger.Namespace,
		"instance":  jaeger.Name,
	})

	// we now traverse the list, so that we end up with three lists:
	// 1) deployments that are on both `desired` and `existing` (update)
	// 2) deployments that are only on `desired` (create)
	// 3) deployments that are only on `existing` (delete)
	depInventory := inventory.ForDeployments(depList.Items, desired)
	for _, d := range depInventory.Create {
		logFields.WithField("deployment", d.Name).Debug("creating deployment")
		if err := r.client.Create(context.Background(), &d); err != nil {
			return err
		}
	}

	for _, d := range depInventory.Update {
		logFields.WithField("deployment", d.Name).Debug("updating deployment")
		if err := r.client.Update(context.Background(), &d); err != nil {
			return err
		}
	}

	// wait for the created and updated pods to stabilize, before we move on with
	// the removal of the old deployments
	for _, d := range depInventory.Create {
		if err := r.waitForStability(d); err != nil {
			return err
		}
	}
	for _, d := range depInventory.Update {
		if err := r.waitForStability(d); err != nil {
			return err
		}
	}

	for _, d := range depInventory.Delete {
		logFields.WithField("deployment", d.Name).Debug("deleting deployment")
		if err := r.client.Delete(context.Background(), &d); err != nil {
			return err
		}
	}

	return nil
}

func (r *ReconcileJaeger) waitForStability(dep appsv1.Deployment) error {
	// TODO: decide what's a good timeout... the first cold run might take a while to download
	// the images, subsequent runs should take only a few seconds
	return wait.PollImmediate(time.Second, 5*time.Minute, func() (done bool, err error) {
		d := &appsv1.Deployment{}
		if err := r.client.Get(context.Background(), types.NamespacedName{Name: dep.Name, Namespace: dep.Namespace}, d); err != nil {
			return false, err
		}

		if d.Status.ReadyReplicas != d.Status.Replicas {
			log.WithFields(log.Fields{
				"namespace": dep.Namespace,
				"name":      dep.Name,
				"ready":     d.Status.ReadyReplicas,
				"desired":   d.Status.Replicas,
			}).Debug("Waiting for deployment to estabilize")
			return false, nil
		}

		return true, nil
	})
}
