package elasticsearch

import (
	"context"

	esv1 "github.com/openshift/elasticsearch-operator/apis/logging/v1"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/jaegertracing/jaeger-operator/pkg/controller/elasticsearch"
)

// ElasticsearchReconciler reconciles a Deployment object
type ElasticsearchReconciler struct {
	reconcilier *elasticsearch.ReconcileElasticsearch
}

// NewElasticsearchReconciler creates a new deployment reconciler controller
func NewElasticsearchReconciler(client client.Client, clientReader client.Reader) *ElasticsearchReconciler {
	return &ElasticsearchReconciler{
		reconcilier: elasticsearch.New(client, clientReader),
	}
}

// +kubebuilder:rbac:groups=logging.openshift.io,resources=elasticsearch,verbs=get;list;watch;create;update;patch;delete

// Reconcile deployment resource
func (r *ElasticsearchReconciler) Reconcile(ctx context.Context, request ctrl.Request) (ctrl.Result, error) {
	return r.reconcilier.Reconcile(ctx, request)
}

// SetupWithManager sets up the controller with the Manager.
func (r *ElasticsearchReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&esv1.Elasticsearch{}).
		Complete(r)
}
