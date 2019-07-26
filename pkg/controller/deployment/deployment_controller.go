package deployment

import (
	"context"

	log "github.com/sirupsen/logrus"
	appsv1 "k8s.io/api/apps/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	"sigs.k8s.io/controller-runtime/pkg/source"

	"github.com/jaegertracing/jaeger-operator/pkg/apis/jaegertracing/v1"
	"github.com/jaegertracing/jaeger-operator/pkg/inject"
)

// Add creates a new Deployment Controller and adds it to the Manager. The Manager will set fields on the Controller
// and Start it when the Manager is Started.
func Add(mgr manager.Manager) error {
	return add(mgr, newReconciler(mgr))
}

// newReconciler returns a new reconcile.Reconciler
func newReconciler(mgr manager.Manager) reconcile.Reconciler {
	return &ReconcileDeployment{client: mgr.GetClient(), scheme: mgr.GetScheme()}
}

// add adds a new Controller to mgr with r as the reconcile.Reconciler
func add(mgr manager.Manager, r reconcile.Reconciler) error {
	// Create a new controller
	c, err := controller.New("deployment-controller", mgr, controller.Options{Reconciler: r})
	if err != nil {
		return err
	}

	// Watch for changes to primary resource Deployment
	err = c.Watch(&source.Kind{Type: &appsv1.Deployment{}}, &handler.EnqueueRequestForObject{})
	if err != nil {
		return err
	}

	return nil
}

var _ reconcile.Reconciler = &ReconcileDeployment{}

// ReconcileDeployment reconciles a Deployment object
type ReconcileDeployment struct {
	// This client, initialized using mgr.Client() above, is a split client
	// that reads objects from the cache and writes to the apiserver
	client client.Client
	scheme *runtime.Scheme
}

// Reconcile reads that state of the cluster for a Deployment object and makes changes based on the state read
// and what is in the Deployment.Spec
// Note:
// The Controller will requeue the Request to be processed again if the returned error is non-nil or
// Result.Requeue is true, otherwise upon completion it will remove the work from the queue.
func (r *ReconcileDeployment) Reconcile(request reconcile.Request) (reconcile.Result, error) {
	log.WithFields(log.Fields{
		"namespace": request.Namespace,
		"name":      request.Name,
	}).Debug("Reconciling Deployment")

	// Fetch the Deployment instance
	instance := &appsv1.Deployment{}
	err := r.client.Get(context.Background(), request.NamespacedName, instance)
	if err != nil {
		if errors.IsNotFound(err) {
			// Request object not found, could have been deleted after reconcile request.
			// Owned objects are automatically garbage collected. For additional cleanup logic use finalizers.
			// Return and don't requeue
			return reconcile.Result{}, nil
		}
		// Error reading the object - requeue the request.
		return reconcile.Result{}, err
	}

	if inject.Needed(instance) {
		pods := &v1.JaegerList{}
		opts := &client.ListOptions{}
		err := r.client.List(context.Background(), opts, pods)
		if err != nil {
			log.WithError(err).Error("failed to get the available Jaeger pods")
			return reconcile.Result{}, err
		}

		jaeger := inject.Select(instance, pods)
		if jaeger != nil && jaeger.GetDeletionTimestamp() == nil {
			// a suitable jaeger instance was found! let's inject a sidecar pointing to it then
			// Verified that jaeger instance was found and is not marked for deletion.
			log.WithFields(log.Fields{
				"deployment":       instance.Name,
				"namespace":        instance.Namespace,
				"jaeger":           jaeger.Name,
				"jaeger-namespace": jaeger.Namespace,
			}).Info("Injecting Jaeger Agent sidecar")
			instance = inject.Sidecar(jaeger, instance)
			if err := r.client.Update(context.Background(), instance); err != nil {
				log.WithField("deployment", instance).WithError(err).Error("failed to update")
				return reconcile.Result{}, err
			}
		} else {
			log.WithField("deployment", instance.Name).Info("No suitable Jaeger instances found to inject a sidecar")
		}
	}

	return reconcile.Result{}, nil
}
