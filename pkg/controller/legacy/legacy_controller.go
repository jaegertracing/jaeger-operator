package legacy

import (
	"context"

	log "github.com/sirupsen/logrus"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	"sigs.k8s.io/controller-runtime/pkg/source"

	"github.com/jaegertracing/jaeger-operator/pkg/apis/io/v1alpha1"
)

// Add creates a new Jaeger Controller and adds it to the Manager. The Manager will set fields on the Controller
// and Start it when the Manager is Started.
func Add(mgr manager.Manager) error {
	return add(mgr, newReconciler(mgr))
}

// newReconciler returns a new reconcile.Reconciler
func newReconciler(mgr manager.Manager) reconcile.Reconciler {
	return &ReconcileJaeger{client: mgr.GetClient(), scheme: mgr.GetScheme()}
}

// add adds a new Controller to mgr with r as the reconcile.Reconciler
func add(mgr manager.Manager, r reconcile.Reconciler) error {
	// Create a new controller
	c, err := controller.New("legacy-controller", mgr, controller.Options{Reconciler: r})
	if err != nil {
		return err
	}

	// Watch for changes to primary resource Jaeger
	err = c.Watch(&source.Kind{Type: &v1alpha1.Jaeger{}}, &handler.EnqueueRequestForObject{})
	if err != nil {
		log.Debug("Legacy CRD not found. Legacy controller will not be active.")
	}

	return nil
}

var _ reconcile.Reconciler = &ReconcileJaeger{}

// ReconcileJaeger reconciles a Jaeger object
type ReconcileJaeger struct {
	// This client, initialized using mgr.Client() above, is a split client
	// that reads objects from the cache and writes to the apiserver
	client client.Client
	scheme *runtime.Scheme
}

// Reconcile reads that state of the cluster for a Jaeger object and makes changes based on the state read
// and what is in the Jaeger.Spec
// Note:
// The Controller will requeue the Request to be processed again if the returned error is non-nil or
// Result.Requeue is true, otherwise upon completion it will remove the work from the queue.
func (r *ReconcileJaeger) Reconcile(request reconcile.Request) (reconcile.Result, error) {
	// Fetch the Jaeger instance
	instance := &v1alpha1.Jaeger{}
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

	log.WithFields(log.Fields{
		"namespace": request.Namespace,
		"instance":  request.Name,
	}).Warn("Attention: you are using a legacy CR for Jaeger. Use the apiVersion 'jaegertracing.io/v1' instead of 'io.jaegertracing/v1alpha1'")

	// reconcile in a few seconds, to get the status object updated
	return reconcile.Result{}, nil
}
