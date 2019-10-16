package jaeger

import (
	"context"
	"time"

	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	log "github.com/sirupsen/logrus"
	corev1 "k8s.io/api/apps/v1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/util/wait"
	"sigs.k8s.io/controller-runtime/pkg/client"

	v1 "github.com/jaegertracing/jaeger-operator/pkg/apis/jaegertracing/v1"
	"github.com/jaegertracing/jaeger-operator/pkg/inventory"
	esv1 "github.com/jaegertracing/jaeger-operator/pkg/storage/elasticsearch/v1"
)

var (
	// ErrElasticsearchRemoved is returned when an ES cluster existed but has been removed
	ErrElasticsearchRemoved = errors.New("Elasticsearch cluster has been removed")
)

func (r *ReconcileJaeger) applyElasticsearches(jaeger v1.Jaeger, desired []esv1.Elasticsearch) error {
	opts := []client.ListOption{
		client.InNamespace(jaeger.Namespace),
		client.MatchingLabels(map[string]string{
			"app.kubernetes.io/instance": jaeger.Name,
			"app.kubernetes.io/part-of":  "jaeger",
		}),
	}
	list := &esv1.ElasticsearchList{}
	if err := r.client.List(context.Background(), list, opts...); err != nil {
		return err
	}

	inv := inventory.ForElasticsearches(list.Items, desired)
	for _, d := range inv.Create {
		jaeger.Logger().WithFields(log.Fields{
			"elasticsearch": d.Name,
			"namespace":     d.Namespace,
		}).Debug("creating elasticsearch")
		if err := r.client.Create(context.Background(), &d); err != nil {
			return err
		}
		if err := waitForAvailableElastic(r.client, d); err != nil {
			return errors.Wrap(err, "elasticsearch cluster didn't get to ready state")
		}
	}

	for _, d := range inv.Update {
		jaeger.Logger().WithFields(log.Fields{
			"elasticsearch": d.Name,
			"namespace":     d.Namespace,
		}).Debug("updating elasticsearch")
		if err := r.client.Update(context.Background(), &d); err != nil {
			return err
		}
	}

	for _, d := range inv.Delete {
		jaeger.Logger().WithFields(log.Fields{
			"elasticsearch": d.Name,
			"namespace":     d.Namespace,
		}).Debug("deleting elasticsearch")
		if err := r.client.Delete(context.Background(), &d); err != nil {
			return err
		}
	}

	return nil
}

func waitForAvailableElastic(c client.Client, es esv1.Elasticsearch) error {
	var expectedSize int32
	for _, n := range es.Spec.Nodes {
		expectedSize += n.NodeCount
	}

	seen := false
	return wait.PollImmediate(time.Second, 2*time.Minute, func() (done bool, err error) {
		depList := corev1.DeploymentList{}
		labels := map[string]string{
			"cluster-name": es.Name,
			"component":    "elasticsearch",
		}
		opts := []client.ListOption{
			client.InNamespace(es.Namespace),
			client.MatchingLabels(labels),
		}

		if err = c.List(context.Background(), &depList, opts...); err != nil {
			if k8serrors.IsNotFound(err) {
				if seen {
					// we have seen this object before, but it doesn't exist anymore!
					// we don't have anything else to do here, break the poll
					log.WithFields(log.Fields{
						"namespace": es.Namespace,
						"name":      es.Name,
					}).Warn("Elasticsearch cluster has been removed.")
					return true, ErrElasticsearchRemoved
				}

				// the object might have not been created yet
				log.WithFields(log.Fields{
					"namespace": es.Namespace,
					"name":      es.Name,
				}).Debug("Elasticsearch cluster doesn't exist yet.")
				return false, nil
			}
			return false, err
		}

		seen = true
		availableDep := int32(0)
		for _, d := range depList.Items {
			if d.Status.Replicas == d.Status.AvailableReplicas {
				availableDep++
			}
		}
		ssList := corev1.StatefulSetList{}
		if err = c.List(context.Background(), &ssList, opts...); err != nil {
			if k8serrors.IsNotFound(err) {
				// the object might have not been created yet
				log.WithFields(log.Fields{
					"namespace": es.Namespace,
					"name":      es.Name,
				}).Debug("Elasticsearch cluster doesn't exist yet.")
				return false, nil
			}
			return false, err
		}
		ssAvailableRep := int32(0)
		ssReplicas := int32(0)
		for _, s := range ssList.Items {
			ssReplicas += *s.Spec.Replicas
			ssAvailableRep += s.Status.ReadyReplicas
		}
		logrus.WithFields(logrus.Fields{
			"namespace":                 es.Namespace,
			"name":                      es.Name,
			"desiredESNodes":            expectedSize,
			"desiredStatefulSetNodes":   ssReplicas,
			"availableStatefulSetNodes": ssAvailableRep,
			"desiredDeploymentNodes":    expectedSize - ssReplicas,
			"availableDeploymentNodes":  availableDep,
		}).Debug("Waiting for Elasticsearch to be available")
		return availableDep+ssAvailableRep == expectedSize, nil
	})
}
