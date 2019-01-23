package jaeger

import (
	"context"
	"fmt"

	log "github.com/sirupsen/logrus"
	appsv1 "k8s.io/api/apps/v1"
	"k8s.io/apimachinery/pkg/types"

	"github.com/jaegertracing/jaeger-operator/pkg/apis/io/v1alpha1"
)

// UpdateReplicaSize synchronizes the desired value from the CR with the derived objects
func (r *ReconcileJaeger) UpdateReplicaSize(instance v1alpha1.Jaeger) error {
	// we have an explicit inventory of object types that uses this size:
	// * JaegerAllInOneSpec
	// * JaegerCollectorSpec
	// * JaegerIngesterSpec
	// * JaegerQuerySpec

	if err := r.updateReplicaSizeAllInOne(instance); err != nil {
		log.WithField("instance", instance.Name).WithError(err).Error("failed to synchronize the replica size for all-in-one")
		return err
	}

	if err := r.updateReplicaSizeCollector(instance); err != nil {
		log.WithField("instance", instance.Name).WithError(err).Error("failed to synchronize the replica size for collector")
		return err
	}

	if err := r.updateReplicaSizeIngester(instance); err != nil {
		log.WithField("instance", instance.Name).WithError(err).Error("failed to synchronize the replica size for ingester")
		return err
	}

	if err := r.updateReplicaSizeQuery(instance); err != nil {
		log.WithField("instance", instance.Name).WithError(err).Error("failed to synchronize the replica size for query")
		return err
	}

	return nil
}

func (r *ReconcileJaeger) updateReplicaSizeAllInOne(instance v1alpha1.Jaeger) error {
	return r.updateReplicaSizeDeployment(instance, instance.Name, instance.Spec.AllInOne.Size)
}

func (r *ReconcileJaeger) updateReplicaSizeCollector(instance v1alpha1.Jaeger) error {
	name := fmt.Sprintf("%s-collector", instance.Name) // TODO: code duplicated from deployment/collector.go
	return r.updateReplicaSizeDeployment(instance, name, instance.Spec.Collector.Size)
}

func (r *ReconcileJaeger) updateReplicaSizeIngester(instance v1alpha1.Jaeger) error {
	name := fmt.Sprintf("%s-ingester", instance.Name) // TODO: code duplicated from deployment/ingester.go
	return r.updateReplicaSizeDeployment(instance, name, instance.Spec.Ingester.Size)
}

func (r *ReconcileJaeger) updateReplicaSizeQuery(instance v1alpha1.Jaeger) error {
	name := fmt.Sprintf("%s-query", instance.Name) // TODO: code duplicated from deployment/query.go
	return r.updateReplicaSizeDeployment(instance, name, instance.Spec.Query.Size)
}

func (r *ReconcileJaeger) updateReplicaSizeDeployment(instance v1alpha1.Jaeger, name string, size *int32) error {
	if nil != size {
		dep := &appsv1.Deployment{}

		if err := r.client.Get(context.Background(), types.NamespacedName{Name: name, Namespace: instance.Namespace}, dep); err != nil {
			log.WithFields(log.Fields{
				"instance":   instance.Name,
				"deployment": name,
			}).WithError(err).Error("failed to get deployment")
			return err
		}

		if *dep.Spec.Replicas != *size {
			log.WithFields(log.Fields{
				"instance":   instance.Name,
				"deployment": name,
				"new-size":   *size,
				"old-size":   *dep.Spec.Replicas,
			}).Info("Replica size change detected")

			dep.Spec.Replicas = size

			if err := r.client.Update(context.Background(), dep); err != nil {
				log.WithFields(log.Fields{
					"instance":   instance.Name,
					"deployment": name,
				}).WithError(err).Error("failed to update deployment")
				return err
			}
			return nil
		}
	}

	return nil
}
