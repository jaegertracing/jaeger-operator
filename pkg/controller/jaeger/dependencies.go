package jaeger

import (
	"context"
	"time"

	log "github.com/sirupsen/logrus"
	batchv1 "k8s.io/api/batch/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/wait"

	"github.com/jaegertracing/jaeger-operator/pkg/strategy"
)

func (r *ReconcileJaeger) handleDependencies(str strategy.S) error {
	for _, dep := range str.Dependencies() {
		err := r.client.Create(context.Background(), &dep)
		if err != nil && !apierrors.IsAlreadyExists(err) {
			return err
		}

		// default to 2 minutes, in case we get a null pointer
		deadline := time.Duration(int64(120))
		if nil != dep.Spec.ActiveDeadlineSeconds {
			// we probably want to add a couple of seconds to this deadline, but for now, this should be sufficient
			deadline = time.Duration(int64(*dep.Spec.ActiveDeadlineSeconds))
		}

		return wait.PollImmediate(time.Second, deadline*time.Second, func() (done bool, err error) {
			batch := &batchv1.Job{}
			if err = r.client.Get(context.Background(), types.NamespacedName{Name: dep.Name, Namespace: dep.Namespace}, batch); err != nil {
				if k8serrors.IsNotFound(err) {
					// the object might have not been created yet
					log.WithFields(log.Fields{
						"namespace": dep.Namespace,
						"name":      dep.Name,
					}).Debug("Dependency doesn't exist yet.")
					return false, nil
				}
				return false, err
			}

			// for now, we just assume each batch job has one pod
			if batch.Status.Succeeded != 1 {
				log.WithFields(log.Fields{
					"namespace": dep.Namespace,
					"name":      dep.Name,
				}).Debug("Waiting for dependency to complete")
				return false, nil
			}

			return true, nil
		})
	}

	return nil
}
