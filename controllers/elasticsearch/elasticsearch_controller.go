package elasticsearch

import (
	"context"

	esv1 "github.com/openshift/elasticsearch-operator/apis/logging/v1"
	"k8s.io/client-go/discovery"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/jaegertracing/jaeger-operator/pkg/autodetect"
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
	esCRDInstalled, err := isOpenShiftESCRDAvailable(mgr)
	if err != nil {
		return err
	}
	if esCRDInstalled {
		return ctrl.NewControllerManagedBy(mgr).
			For(&esv1.Elasticsearch{}).
			Complete(r)
	}
	return nil
}

const elasticsearchGroup = "logging.openshift.io"

func isOpenShiftESCRDAvailable(mgr ctrl.Manager) (bool, error) {
	dcl, err := discovery.NewDiscoveryClientForConfig(mgr.GetConfig())
	if err != nil {
		return false, err
	}
	apiLists, err := autodetect.AvailableAPIs(dcl, map[string]bool{elasticsearchGroup: true})
	if err != nil {
		return false, err
	}
	return autodetect.IsElasticsearchOperatorAvailable(apiLists), nil
}
