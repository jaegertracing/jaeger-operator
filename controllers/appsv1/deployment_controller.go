package appsv1

import (
	"context"

	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/source"

	appsv1 "k8s.io/api/apps/v1"
	"k8s.io/apimachinery/pkg/runtime"

	v1 "github.com/jaegertracing/jaeger-operator/apis/v1"
	"github.com/jaegertracing/jaeger-operator/pkg/controller/deployment"
)

// DeploymentReconciler reconciles a Deployment object
type DeploymentReconciler struct {
	reconcilier *deployment.ReconcileDeployment
}

// NewDeploymentReconciler creates a new deployment reconcilier controller
func NewDeploymentReconciler(client client.Client, clientReader client.Reader, scheme *runtime.Scheme) *DeploymentReconciler {
	return &DeploymentReconciler{
		reconcilier: deployment.New(client, clientReader, scheme),
	}
}

// Reconcile deployment resource
// +kubebuilder:rbac:groups=apps,resources=deployments,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=apps,resources=deployments/status,verbs=get;update;patch

func (r *DeploymentReconciler) Reconcile(ctx context.Context, request ctrl.Request) (ctrl.Result, error) {
	return r.reconcilier.Reconcile(ctx, request)
}

// SetupWithManager sets up the controller with the Manager.
func (r *DeploymentReconciler) SetupWithManager(mgr ctrl.Manager) error {
	err := ctrl.NewControllerManagedBy(mgr).
		For(&appsv1.Deployment{}).
		Watches(&source.Kind{Type: &v1.Jaeger{}}, handler.EnqueueRequestsFromMapFunc(r.reconcilier.SyncOnJaegerChanges)).
		Complete(r)
	return err
}
