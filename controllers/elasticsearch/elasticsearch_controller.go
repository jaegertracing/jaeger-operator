package elasticsearch

import (
	"context"

	esv1 "github.com/openshift/elasticsearch-operator/apis/logging/v1"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/jaegertracing/jaeger-operator/pkg/controller/elasticsearch"
)

// Reconciler reconciles a Deployment object
type Reconciler struct {
	reconcilier *elasticsearch.ReconcileElasticsearch
}

// NewReconciler creates a new deployment reconciler controller
func NewReconciler(client client.Client, clientReader client.Reader) *Reconciler {
	return &Reconciler{
		reconcilier: elasticsearch.New(client, clientReader),
	}
}

// +kubebuilder:rbac:groups=logging.openshift.io,resources=elasticsearch,verbs=get;list;watch;create;update;patch;delete

// Reconcile deployment resource
func (r *Reconciler) Reconcile(ctx context.Context, request ctrl.Request) (ctrl.Result, error) {
	return r.reconcilier.Reconcile(ctx, request)
}

// SetupWithManager sets up the controller with the Manager.
func (r *Reconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&esv1.Elasticsearch{}).
		Complete(r)
}
