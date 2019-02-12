package jaeger

import (
	"context"
	"fmt"
	"time"

	log "github.com/sirupsen/logrus"
	appsv1 "k8s.io/api/apps/v1"
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
	"github.com/jaegertracing/jaeger-operator/pkg/inventory"
	"github.com/jaegertracing/jaeger-operator/pkg/strategy"
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
	strategyChooser func(*v1alpha1.Jaeger) strategy.S
}

// Reconcile reads that state of the cluster for a Jaeger object and makes changes based on the state read
// and what is in the Jaeger.Spec
// Note:
// The Controller will requeue the Request to be processed again if the returned error is non-nil or
// Result.Requeue is true, otherwise upon completion it will remove the work from the queue.
func (r *ReconcileJaeger) Reconcile(request reconcile.Request) (reconcile.Result, error) {
	log.WithFields(log.Fields{
		"namespace": request.Namespace,
		"instance":  request.Name,
	}).Debug("Reconciling Jaeger")

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

	str := r.runStrategyChooser(instance)

	// wait for all the dependencies to succeed
	if err := r.handleDependencies(str); err != nil {
		log.WithFields(log.Fields{
			"namespace": instance.Namespace,
			"instance":  instance.Name,
		}).WithError(err).Error("failed to handle the dependencies")
		return reconcile.Result{}, err
	}

	applied, err := r.apply(*instance, str)
	if err != nil {
		log.WithFields(log.Fields{
			"namespace": instance.Namespace,
			"instance":  instance.Name,
		}).WithError(err).Error("failed to apply the changes")
		return reconcile.Result{}, err
	}

	if applied {
		log.WithFields(log.Fields{
			"namespace": instance.Namespace,
			"instance":  instance.Name,
		}).Info("Configured Jaeger instance")
	}

	// we store back the changed CR, so that what is stored reflects what is being used
	if err := r.client.Update(context.Background(), instance); err != nil {
		log.WithFields(log.Fields{
			"namespace": instance.Namespace,
			"instance":  instance.Name,
		}).WithError(err).Error("failed to update")
		return reconcile.Result{}, err
	}

	return reconcile.Result{}, nil
}

func (r *ReconcileJaeger) runStrategyChooser(instance *v1alpha1.Jaeger) strategy.S {
	if nil == r.strategyChooser {
		return defaultStrategyChooser(instance)
	}

	return r.strategyChooser(instance)
}

func defaultStrategyChooser(instance *v1alpha1.Jaeger) strategy.S {
	return strategy.For(context.Background(), instance)
}

func (r *ReconcileJaeger) handleDependencies(str strategy.S) error {
	for _, dep := range str.Dependencies() {
		err := r.client.Create(context.Background(), &dep)
		if err != nil && !apierrors.IsAlreadyExists(err) {
			return err
		}

		// default to 2 minutes, in case we get a null pointer
		deadline := time.Duration(int64(120))
		if nil != dep.Spec.ActiveDeadlineSeconds {
			// we probably want to add a couple of seconds to this deadline, but for now, this should be sufficient
			deadline = time.Duration(int64(*dep.Spec.ActiveDeadlineSeconds))
		}

		return wait.PollImmediate(time.Second, deadline*time.Second, func() (done bool, err error) {
			batch := &batchv1.Job{}
			if err = r.client.Get(context.Background(), types.NamespacedName{Name: dep.Name, Namespace: dep.Namespace}, batch); err != nil {
				return false, err
			}

			// for now, we just assume each batch job has one pod
			if batch.Status.Succeeded != 1 {
				log.WithFields(log.Fields{
					"namespace": dep.Namespace,
					"name":      dep.Name,
				}).Debug("Waiting for dependency to complete")
				return false, nil
			}

			return true, nil
		})
	}

	return nil
}

func (r *ReconcileJaeger) apply(jaeger v1alpha1.Jaeger, str strategy.S) (bool, error) {
	if err := r.applyDeployments(jaeger, str.Deployments()); err != nil {
		return false, err
	}

	return true, nil
}

func (r *ReconcileJaeger) applyDeployments(jaeger v1alpha1.Jaeger, desired []appsv1.Deployment) error {
	opts := client.MatchingLabels(map[string]string{
		"app.kubernetes.io/instance":   jaeger.Name,
		"app.kubernetes.io/managed-by": "jaeger-operator",
	})
	depList := &appsv1.DeploymentList{}
	if err := r.client.List(context.Background(), opts, depList); err != nil {
		return err
	}

	// we now traverse the list, so that we end up with three lists:
	// 1) deployments that are on both `desired` and `existing` (update)
	// 2) deployments that are only on `desired` (create)
	// 3) deployments that are only on `existing` (delete)
	depInventory := inventory.ForDeployments(depList.Items, desired)
	for _, d := range depInventory.Create {
		log.WithFields(log.Fields{
			"namespace":  jaeger.Namespace,
			"instance":   jaeger.Name,
			"deployment": d.Name,
		}).Debug("creating deployment")
		if err := r.client.Create(context.Background(), &d); err != nil {
			return err
		}
	}

	for _, d := range depInventory.Update {
		log.WithFields(log.Fields{
			"namespace":  jaeger.Namespace,
			"instance":   jaeger.Name,
			"deployment": d.Name,
		}).Debug("updating deployment")
		if err := r.client.Update(context.Background(), &d); err != nil {
			return err
		}
	}

	for _, d := range depInventory.Create {
		if err := r.waitForStability(d); err != nil {
			return err
		}
	}
	for _, d := range depInventory.Update {
		if err := r.waitForStability(d); err != nil {
			return err
		}
	}

	for _, d := range depInventory.Delete {
		log.WithFields(log.Fields{
			"namespace":  jaeger.Namespace,
			"instance":   jaeger.Name,
			"deployment": d.Name,
		}).Debug("deleting deployment")
		if err := r.client.Delete(context.Background(), &d); err != nil {
			return err
		}
	}

	return nil
}

func (r *ReconcileJaeger) waitForStability(dep appsv1.Deployment) error {
	return wait.PollImmediate(time.Second, 5*time.Second, func() (done bool, err error) {
		d := &appsv1.Deployment{}
		if err := r.client.Get(context.Background(), types.NamespacedName{Name: dep.Name, Namespace: dep.Namespace}, d); err != nil {
			return false, err
		}

		if d.Status.ReadyReplicas != d.Status.Replicas {
			log.WithFields(log.Fields{
				"namespace": dep.Namespace,
				"name":      dep.Name,
				"ready":     d.Status.ReadyReplicas,
				"desired":   d.Status.Replicas,
			}).Debug("Waiting for deployment to estabilize")
			return false, nil
		}

		return true, nil
	})
}
