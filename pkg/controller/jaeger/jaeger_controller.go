package jaeger

import (
	"context"
	"fmt"
	"strings"

	log "github.com/sirupsen/logrus"
	"github.com/spf13/viper"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	"sigs.k8s.io/controller-runtime/pkg/source"

	"github.com/jaegertracing/jaeger-operator/pkg/apis/io/v1alpha1"
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
	logFields := log.WithFields(log.Fields{
		"namespace": request.Namespace,
		"instance":  request.Name,
	})

	logFields.Debug("Reconciling Jaeger")

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
		logFields.WithError(err).Error("failed to handle the dependencies")
		return reconcile.Result{}, err
	}

	applied, err := r.apply(*instance, str)
	if err != nil {
		logFields.WithError(err).Error("failed to apply the changes")
		return reconcile.Result{}, err
	}

	if applied {
		logFields.Info("Configured Jaeger instance")
	}

	// we store back the changed CR, so that what is stored reflects what is being used
	if err := r.client.Update(context.Background(), instance); err != nil {
		logFields.WithError(err).Error("failed to store back the current CustomResource")
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

func (r *ReconcileJaeger) apply(jaeger v1alpha1.Jaeger, str strategy.S) (bool, error) {
	if err := r.applyRoles(jaeger, str.Roles()); err != nil {
		return false, err
	}

	if err := r.applyAccounts(jaeger, str.Accounts()); err != nil {
		return false, err
	}

	if err := r.applyRoleBindings(jaeger, str.RoleBindings()); err != nil {
		return false, err
	}

	if err := r.applyConfigMaps(jaeger, str.ConfigMaps()); err != nil {
		return false, err
	}

	if err := r.applySecrets(jaeger, str.Secrets()); err != nil {
		return false, err
	}

	if err := r.applyCronJobs(jaeger, str.CronJobs()); err != nil {
		return false, err
	}

	if err := r.applyDaemonSets(jaeger, str.DaemonSets()); err != nil {
		return false, err
	}

	if err := r.applyDeployments(jaeger, str.Deployments()); err != nil {
		return false, err
	}

	if err := r.applyServices(jaeger, str.Services()); err != nil {
		return false, err
	}

	if strings.EqualFold(viper.GetString("platform"), v1alpha1.FlagPlatformOpenShift) {
		if err := r.applyRoutes(jaeger, str.Routes()); err != nil {
			return false, err
		}
	} else {
		if err := r.applyIngresses(jaeger, str.Ingresses()); err != nil {
			return false, err
		}
	}

	return true, nil
}
