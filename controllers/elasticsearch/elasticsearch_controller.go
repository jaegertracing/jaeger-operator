package elasticsearch

import (
	"context"
	"strings"

	esv1 "github.com/openshift/elasticsearch-operator/apis/logging/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/discovery"
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
	esKindInstalled, err := isOpenShiftElasticsearchController(mgr)
	if err != nil {
		return err
	}
	if esKindInstalled {
		return ctrl.NewControllerManagedBy(mgr).
			For(&esv1.Elasticsearch{}).
			Complete(r)
	}
	return nil
}

const elasticsearchGroup = "logging.openshift.io"
const elasticsearchKind = "Elasticsearch"

func isOpenShiftElasticsearchController(mgr ctrl.Manager) (bool, error) {
	dcl, err := discovery.NewDiscoveryClientForConfig(mgr.GetConfig())
	if err != nil {
		return false, err
	}
	var apiLists []*metav1.APIResourceList
	groupList, err := dcl.ServerGroups()
	if err != nil {
		return false, err
	}

	for _, sg := range groupList.Groups {
		if sg.Name == elasticsearchGroup {
			groupAPIList, err := dcl.ServerResourcesForGroupVersion(sg.PreferredVersion.GroupVersion)
			if err != nil {
				return false, err
			}
			apiLists = append(apiLists, groupAPIList)
		}
	}

	for _, r := range apiLists {
		if strings.HasPrefix(r.GroupVersion, elasticsearchGroup) {
			for _, api := range r.APIResources {
				if api.Kind == elasticsearchKind {
					return true, nil
				}
			}
		}
	}
	return false, nil
}
