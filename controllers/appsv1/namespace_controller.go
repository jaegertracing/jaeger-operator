package appsv1

import (
	"context"

	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"

	"github.com/jaegertracing/jaeger-operator/pkg/controller/namespace"
)

// NamespaceReconciler reconciles a Deployment object
type NamespaceReconciler struct {
	reconcilier *namespace.ReconcileNamespace
}

// NewNamespaceReconciler creates a new namespace reconcilier controller
func NewNamespaceReconciler(client client.Client, clientReader client.Reader, scheme *runtime.Scheme) *NamespaceReconciler {
	return &NamespaceReconciler{
		reconcilier: namespace.New(client, clientReader, scheme),
	}
}

// +kubebuilder:rbac:groups=core,resources=namespaces,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=core,resources=namespaces/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=apps,resources=deployments,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=apps,resources=deployments/status,verbs=get;update;patch

// Reconcile namespace resource
func (r *NamespaceReconciler) Reconcile(ctx context.Context, request ctrl.Request) (ctrl.Result, error) {
	return r.reconcilier.Reconcile(request)
}

// SetupWithManager sets up the controller with the Manager.
func (r *NamespaceReconciler) SetupWithManager(mgr ctrl.Manager) error {
	err := ctrl.NewControllerManagedBy(mgr).
		For(&corev1.Namespace{}).
		Complete(r)
	return err
}
