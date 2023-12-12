package jaeger

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"time"

	esv1 "github.com/openshift/elasticsearch-operator/apis/logging/v1"
	"go.opentelemetry.io/otel"
	appsv1 "k8s.io/api/apps/v1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/util/wait"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"

	v1 "github.com/jaegertracing/jaeger-operator/apis/v1"
	"github.com/jaegertracing/jaeger-operator/pkg/inventory"
	"github.com/jaegertracing/jaeger-operator/pkg/tracing"
)

// ErrElasticsearchRemoved is returned when an ES cluster existed but has been removed
var ErrElasticsearchRemoved = errors.New("Elasticsearch cluster has been removed")

func (r *ReconcileJaeger) applyElasticsearches(ctx context.Context, jaeger v1.Jaeger, desired []esv1.Elasticsearch) error {
	tracer := otel.GetTracerProvider().Tracer(v1.ReconciliationTracer)
	ctx, span := tracer.Start(ctx, "applyElasticsearches")
	defer span.End()

	opts := []client.ListOption{
		client.InNamespace(jaeger.Namespace),
		client.MatchingLabels(map[string]string{
			"app.kubernetes.io/instance": jaeger.Name,
			"app.kubernetes.io/part-of":  "jaeger",
		}),
	}
	list := &esv1.ElasticsearchList{}
	if err := r.rClient.List(ctx, list, opts...); err != nil {
		return tracing.HandleError(err, span)
	}

	inv := inventory.ForElasticsearches(list.Items, desired)
	for i := range inv.Create {
		d := inv.Create[i]
		jaeger.Logger().V(-1).Info(
			"creating elasticsearch",
			"elasticsearch", d.Name,
			"namespace", d.Namespace,
		)
		if err := r.client.Create(ctx, &d); err != nil {
			return tracing.HandleError(err, span)
		}

		if err := waitForAvailableElastic(ctx, r.client, d); err != nil {
			return tracing.HandleError(fmt.Errorf("elasticsearch cluster didn't get to ready state: %w", err), span)
		}
	}

	for i := range inv.Update {
		d := inv.Update[i]
		jaeger.Logger().V(-1).Info(
			"updating elasticsearch",
			"elasticsearch", d.Name,
			"namespace", d.Namespace,
		)
		if err := r.client.Update(ctx, &d); err != nil {
			return tracing.HandleError(err, span)
		}
	}

	for i := range inv.Delete {
		d := inv.Delete[i]
		jaeger.Logger().V(-1).Info(
			"deleting elasticsearch",
			"elasticsearch", d.Name,
			"namespace", d.Namespace,
		)
		if err := r.client.Delete(ctx, &d); err != nil {
			return tracing.HandleError(err, span)
		}
	}

	return nil
}

func waitForAvailableElastic(ctx context.Context, c client.Client, es esv1.Elasticsearch) error {
	tracer := otel.GetTracerProvider().Tracer(v1.ReconciliationTracer)
	ctx, span := tracer.Start(ctx, "waitForAvailableElastic")
	defer span.End()

	var expectedSize int32
	for _, n := range es.Spec.Nodes {
		expectedSize += n.NodeCount
	}

	seen := false
	once := &sync.Once{}
	return wait.PollUntilContextTimeout(
		ctx,
		time.Second,
		2*time.Minute,
		true,
		wait.ConditionWithContextFunc(
			func(context.Context) (done bool, err error) {
				depList := appsv1.DeploymentList{}
				labels := map[string]string{
					"cluster-name": es.Name,
					"component":    "elasticsearch",
				}
				opts := []client.ListOption{
					client.InNamespace(es.Namespace),
					client.MatchingLabels(labels),
				}

				if err = c.List(ctx, &depList, opts...); err != nil {
					if k8serrors.IsNotFound(err) {
						if seen {
							// we have seen this object before, but it doesn't exist anymore!
							// we don't have anything else to do here, break the poll
							log.Log.V(1).Info(
								"Elasticsearch cluster has been removed.",
								"namespace", es.Namespace,
								"name", es.Name,
							)
							return true, ErrElasticsearchRemoved
						}

						// the object might have not been created yet
						log.Log.V(-1).Info(
							"Elasticsearch cluster doesn't exist yet.",
							"namespace", es.Namespace,
							"name", es.Name,
						)
						return false, nil
					}
					return false, tracing.HandleError(err, span)
				}

				seen = true
				availableDep := int32(0)
				for _, d := range depList.Items {
					if d.Status.Replicas == d.Status.AvailableReplicas {
						availableDep++
					}
				}
				ssList := appsv1.StatefulSetList{}
				if err = c.List(ctx, &ssList, opts...); err != nil {
					if k8serrors.IsNotFound(err) {
						// the object might have not been created yet
						log.Log.V(-1).Info(
							"Elasticsearch cluster doesn't exist yet.",
							"namespace", es.Namespace,
							"name", es.Name,
						)
						return false, nil
					}
					return false, tracing.HandleError(err, span)
				}
				ssAvailableRep := int32(0)
				ssReplicas := int32(0)
				for _, s := range ssList.Items {
					ssReplicas += *s.Spec.Replicas
					ssAvailableRep += s.Status.ReadyReplicas
				}
				once.Do(func() {
					log.Log.V(-1).Info(
						"Waiting for Elasticsearch to be available",
						"namespace", es.Namespace,
						"name", es.Name,
						"desiredESNodes", expectedSize,
						"desiredStatefulSetNodes", ssReplicas,
						"availableStatefulSetNodes", ssAvailableRep,
						"desiredDeploymentNodes", expectedSize-ssReplicas,
						"availableDeploymentNodes", availableDep,
					)
				})
				return availableDep+ssAvailableRep == expectedSize, nil
			}))
}
