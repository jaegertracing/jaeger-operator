package elasticsearch

import (
	"context"

	esv1 "github.com/openshift/elasticsearch-operator/apis/logging/v1"
	"go.opentelemetry.io/otel"
	otelattribute "go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"k8s.io/apimachinery/pkg/api/errors"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	v1 "github.com/jaegertracing/jaeger-operator/apis/v1"
	"github.com/jaegertracing/jaeger-operator/pkg/tracing"
)

// ReconcileElasticsearch reconciles a Elasticsearch object
type ReconcileElasticsearch struct {
	// This client, initialized using mgr.Client() above, is a split client
	// that reads objects from the cache and writes to the apiserver
	client client.Client

	// this avoid the cache, which we need to bypass because the default client will attempt to place
	// a watch on Namespace at cluster scope, which isn't desirable to us...
	rClient client.Reader
}

// New creates new Elasticsearch controller
func New(client client.Client, clientReader client.Reader) *ReconcileElasticsearch {
	return &ReconcileElasticsearch{
		client:  client,
		rClient: clientReader,
	}
}

// Reconcile reads that state of the cluster for a Namespace object and makes changes based on the state read
// and what is in the Namespace.Spec
// Note:
// The Controller will requeue the Request to be processed again if the returned error is non-nil or
// Result.Requeue is true, otherwise upon completion it will remove the work from the queue.
func (r *ReconcileElasticsearch) Reconcile(ctx context.Context, request reconcile.Request) (reconcile.Result, error) {
	logger := log.Log.WithValues(
		"namespace", request.Namespace,
		"name", request.Name,
	)
	logger.V(-1).Info("Reconciling Elasticsearch")

	tracer := otel.GetTracerProvider().Tracer(v1.ReconciliationTracer)
	ctx, span := tracer.Start(ctx, "reconcileElasticsearch")
	defer span.End()

	span.SetAttributes(otelattribute.String("name", request.Name), otelattribute.String("namespace", request.Namespace))

	es := &esv1.Elasticsearch{}
	err := r.rClient.Get(ctx, request.NamespacedName, es)
	if err != nil {
		if errors.IsNotFound(err) {
			// Request object not found, could have been deleted after reconcile request.
			// Owned objects are automatically garbage collected. For additional cleanup logic use finalizers.
			// Return and don't requeue
			span.SetStatus(codes.Error, err.Error())
			return reconcile.Result{}, nil
		}
		// Error reading the object - requeue the request.
		return reconcile.Result{}, tracing.HandleError(err, span)
	}

	// Fetch Jaeger instance
	jaegers := &v1.JaegerList{}
	err = r.rClient.List(ctx, jaegers, client.InNamespace(request.Namespace))
	if err != nil {
		return reconcile.Result{}, tracing.HandleError(err, span)
	}

	esNodeCount := v1.OpenShiftElasticsearchNodeCount(es.Spec)
	// iterate over jaeger instances and if ES is used then update node count
	// Jaeger instance will be updated in cluster and Jaeger reconciliation will be triggered to
	// update Jaeger deployments (e.g. --es.num-shards).
	for i := 0; i < len(jaegers.Items); i++ {
		jaeger := &jaegers.Items[i]
		if v1.ShouldInjectOpenShiftElasticsearchConfiguration(jaeger.Spec.Storage) {
			if jaeger.Spec.Storage.Elasticsearch.Name == es.Name && jaeger.Spec.Storage.Elasticsearch.NodeCount != esNodeCount {
				logger.Info(
					"Updating Jaeger CR because OpenShift ES number of nodes changed",
					"jaeger", jaeger.Name,
					"old-es-node-count", jaeger.Spec.Storage.Elasticsearch.NodeCount,
					"new-es-node-count", esNodeCount,
				)
				jaeger.Spec.Storage.Elasticsearch.NodeCount = esNodeCount
				if err := r.client.Update(ctx, jaeger); err != nil {
					logger.Error(err, "failed to update Jaeger instance")
					return reconcile.Result{}, tracing.HandleError(err, span)
				}
			}
		}
	}

	return reconcile.Result{}, nil
}
