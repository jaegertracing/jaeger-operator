package jaeger

import (
	"context"
	"time"

	"github.com/pkg/errors"
	"k8s.io/api/apps/v1"
	"k8s.io/apimachinery/pkg/util/wait"
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

	inv := inventory.ForElasticsearches(list.Items, desired)
	for _, d := range inv.Create {
		jaeger.Logger().WithField("elasticsearch", d.Name).Debug("creating elasticsearch")
		if err := r.client.Create(context.Background(), &d); err != nil {
			return err
		}
		if err := waitForAvailableElastic(r.client, d); err != nil {
			return errors.Wrap(err, "elasticsearch cluster didn't get to ready state")
		}
	}

	for _, d := range inv.Update {
		jaeger.Logger().WithField("elasticsearch", d.Name).Debug("updating elasticsearch")
		if err := r.client.Update(context.Background(), &d); err != nil {
			return err
		}
	}

	for _, d := range inv.Delete {
		jaeger.Logger().WithField("elasticsearch", d.Name).Debug("deleting elasticsearch")
		if err := r.client.Delete(context.Background(), &d); err != nil {
			return err
		}
	}

	return nil
}

func waitForAvailableElastic(c client.Client, es esv1alpha1.Elasticsearch) error {
	var expectedSize int32
	for _, n := range es.Spec.Nodes {
		expectedSize += n.NodeCount
	}
	return wait.PollImmediate(time.Second, 2*time.Minute, func() (done bool, err error) {
		depList := v1.DeploymentList{}
		if err = c.List(context.Background(), client.MatchingLabels(es.Labels).InNamespace(es.Namespace), &depList); err != nil {
			return false, err
		}
		available := int32(0)
		for _, d := range depList.Items {
			if d.Status.Replicas == d.Status.AvailableReplicas {
				available++
			}
		}
		return available == expectedSize, nil
	})
	return nil
}
