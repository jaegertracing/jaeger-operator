package jaeger

import (
	"context"
	"fmt"
	"time"

	log "github.com/sirupsen/logrus"
	batchv1 "k8s.io/api/batch/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/wait"
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
	c, err := controller.New("jaeger-controller", mgr, controller.Options{Reconciler: r})
	if err != nil {
		return err
	}

	// Watch for changes to primary resource Jaeger
	err = c.Watch(&source.Kind{Type: &v1alpha1.Jaeger{}}, &handler.EnqueueRequestForObject{})
	if err != nil {
		return err
	}

	return nil
}

var _ reconcile.Reconciler = &ReconcileJaeger{}

// ReconcileJaeger reconciles a Jaeger object
type ReconcileJaeger struct {
	// This client, initialized using mgr.Client() above, is a split client
	// that reads objects from the cache and writes to the apiserver
	client          client.Client
	scheme          *runtime.Scheme
	strategyChooser func(*v1alpha1.Jaeger) Controller
}

// Reconcile reads that state of the cluster for a Jaeger object and makes changes based on the state read
// and what is in the Jaeger.Spec
// Note:
// The Controller will requeue the Request to be processed again if the returned error is non-nil or
// Result.Requeue is true, otherwise upon completion it will remove the work from the queue.
func (r *ReconcileJaeger) Reconcile(request reconcile.Request) (reconcile.Result, error) {
	log.WithFields(log.Fields{
		"namespace": request.Namespace,
		"name":      request.Name,
	}).Print("Reconciling Jaeger")

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

	// workaround for https://github.com/kubernetes-sigs/controller-runtime/issues/202
	// see also: https://github.com/kubernetes-sigs/controller-runtime/pull/212
	// once there's a version incorporating the PR above, the manual setting of the GKV can be removed
	instance.APIVersion = fmt.Sprintf("%s/%s", v1alpha1.SchemeGroupVersion.Group, v1alpha1.SchemeGroupVersion.Version)
	instance.Kind = "Jaeger"

	ctrl := r.runStrategyChooser(instance)

	// wait for all the dependencies to succeed
	if err := r.handleDependencies(ctrl); err != nil {
		return reconcile.Result{}, err
	}

	created, err := r.handleCreate(ctrl)
	if err != nil {
		log.WithField("instance", instance).WithError(err).Error("failed to create")
		return reconcile.Result{}, err
	}

	if created {
		log.WithField("name", instance.Name).Info("Configured Jaeger instance")
	}

	if err := r.handleUpdate(ctrl); err != nil {
		return reconcile.Result{}, err
	}

	// we store back the changed CR, so that what is stored reflects what is being used
	if err := r.client.Update(context.Background(), instance); err != nil {
		log.WithError(err).Error("failed to update")
		return reconcile.Result{}, err
	}

	return reconcile.Result{}, nil
}

func (r *ReconcileJaeger) runStrategyChooser(instance *v1alpha1.Jaeger) Controller {
	if nil == r.strategyChooser {
		return defaultStrategyChooser(instance)
	}

	return r.strategyChooser(instance)
}

func defaultStrategyChooser(instance *v1alpha1.Jaeger) Controller {
	return NewController(context.Background(), instance)
}

func (r *ReconcileJaeger) handleCreate(ctrl Controller) (bool, error) {
	objs := ctrl.Create()
	created := false
	for _, obj := range objs {
		err := r.client.Create(context.Background(), obj)
		if err != nil && !apierrors.IsAlreadyExists(err) {
			log.WithError(err).Error("failed to create")
			return false, err
		}

		if err == nil {
			created = true
		}
	}

	return created, nil
}

func (r *ReconcileJaeger) handleUpdate(ctrl Controller) error {
	objs := ctrl.Update()
	for _, obj := range objs {
		if err := r.client.Update(context.Background(), obj); err != nil {
			log.WithError(err).Error("failed to update")
			return err
		}
	}

	return nil
}

func (r *ReconcileJaeger) handleDependencies(ctrl Controller) error {
	for _, dep := range ctrl.Dependencies() {
		err := r.client.Create(context.Background(), &dep)
		if err != nil && !apierrors.IsAlreadyExists(err) {
			log.WithError(err).Error("failed to create")
			return err
		}

		// we probably want to add a couple of seconds to this deadline, but for now, this should be sufficient
		deadline := time.Duration(*dep.Spec.ActiveDeadlineSeconds)
		return wait.Poll(time.Second, deadline*time.Second, func() (done bool, err error) {
			batch := &batchv1.Job{}
			err = r.client.Get(context.Background(), types.NamespacedName{Name: dep.Name, Namespace: dep.Namespace}, batch)
			if err != nil {
				log.WithField("dependency", dep.Name).WithError(err).Error("failed to get the status of the dependency")
				return false, err
			}

			// for now, we just assume each batch job has one pod
			if batch.Status.Succeeded != 1 {
				log.WithField("dependency", dep.Name).Info("Waiting for dependency to complete")
				return false, nil
			}

			return true, nil
		})
	}

	return nil
}
