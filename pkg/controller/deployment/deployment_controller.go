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

	"github.com/jaegertracing/jaeger-operator/pkg/apis/io/v1alpha1"
	"github.com/jaegertracing/jaeger-operator/pkg/inject"
)

// Add creates a new Controller for Deployment and StatefulSets and adds it to the Manager. The Manager will set fields on the Controller
// and Start it when the Manager is Started.
func Add(mgr manager.Manager) error {
	return add(mgr, newReconciler(mgr))
}

// newReconciler returns a new reconcile.Reconciler
func newReconciler(mgr manager.Manager) reconcile.Reconciler {
	return &ReconcileObject{client: mgr.GetClient(), scheme: mgr.GetScheme()}
}

// add adds a new Controller to mgr with r as the reconcile.Reconciler
func add(mgr manager.Manager, r reconcile.Reconciler) error {
	// Create a new controller
	c, err := controller.New("object-controller", mgr, controller.Options{Reconciler: r})
	if err != nil {
		return err
	}

	// Watch for changes to primary resource Deployment
	err = c.Watch(&source.Kind{Type: &appsv1.Deployment{}}, &handler.EnqueueRequestForObject{})
	if err != nil {
		return err
	}

	// Watch for changes to primary resource StatefulSet
	err = c.Watch(&source.Kind{Type: &appsv1.StatefulSet{}}, &handler.EnqueueRequestForObject{})
	if err != nil {
		return err
	}

	return nil
}

var _ reconcile.Reconciler = &ReconcileObject{}

// ReconcileObject reconciles a Kubernetes object
type ReconcileObject struct {
	// This client, initialized using mgr.Client() above, is a split client
	// that reads objects from the cache and writes to the apiserver
	client client.Client
	scheme *runtime.Scheme
}

// Reconcile reads that state of the cluster for a Kubernetes object and makes changes based on the
// state read and what is in the Object Spec
// Note:
// The Controller will requeue the Request to be processed again if the returned error is non-nil or
// Result.Requeue is true, otherwise upon completion it will remove the work from the queue.
func (r *ReconcileObject) Reconcile(request reconcile.Request) (reconcile.Result, error) {
	log.WithFields(log.Fields{
		"namespace": request.Namespace,
		"name":      request.Name,
	}).Print("Reconciling the Object")

	// Fetch the object instance
	instance := *runtime.Object{}
	err := r.client.Get(context.Background(), request.NamespacedName, *instance)
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

	if inject.Needed(instance.Name, instance.Annotations, instance.Spec.Template.Spec.Containers) {
		pods := &v1alpha1.JaegerList{}
		opts := &client.ListOptions{Namespace: instance.Namespace}
		err := r.client.List(context.Background(), opts, pods)
		if err != nil {
			log.WithError(err).Error("failed to get the available Jaeger pods")
			return reconcile.Result{}, err
		}

		jaeger := inject.Select(instance.Annotations, pods)
		if jaeger != nil {
			// a suitable jaeger instance was found! let's inject a sidecar pointing to it then
			log.WithFields(log.Fields{"deployment": instance.Name, "jaeger": jaeger.Name}).Info("Injecting Jaeger Agent sidecar")
			inject.Sidecar(instance, jaeger)
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
