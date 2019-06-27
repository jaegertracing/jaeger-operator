package jaeger

import (
	"context"
	"fmt"
	"reflect"
	"strings"
	"time"

	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/viper"
	corev1 "k8s.io/api/core/v1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	"sigs.k8s.io/controller-runtime/pkg/source"

	v1 "github.com/jaegertracing/jaeger-operator/pkg/apis/jaegertracing/v1"
	"github.com/jaegertracing/jaeger-operator/pkg/autodetect"
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

	if d, err := autodetect.New(mgr); err != nil {
		log.WithError(err).Warn("failed to start the background process to auto-detect the operator capabilities")
	} else {
		d.Start()
	}

	// Watch for changes to primary resource Jaeger
	err = c.Watch(&source.Kind{Type: &v1.Jaeger{}}, &handler.EnqueueRequestForObject{})
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
	strategyChooser func(*v1.Jaeger) strategy.S
}

// Reconcile reads that state of the cluster for a Jaeger object and makes changes based on the state read
// and what is in the Jaeger.Spec
// Note:
// The Controller will requeue the Request to be processed again if the returned error is non-nil or
// Result.Requeue is true, otherwise upon completion it will remove the work from the queue.
func (r *ReconcileJaeger) Reconcile(request reconcile.Request) (reconcile.Result, error) {
	execution := time.Now().UTC()

	log.WithFields(log.Fields{
		"namespace": request.Namespace,
		"instance":  request.Name,
		"execution": execution,
	}).Debug("Reconciling Jaeger")

	// Fetch the Jaeger instance
	instance := &v1.Jaeger{}
	err := r.client.Get(context.Background(), request.NamespacedName, instance)
	if err != nil {
		if k8serrors.IsNotFound(err) {
			// Request object not found, could have been deleted after reconcile request.
			// Owned objects are automatically garbage collected. For additional cleanup logic use finalizers.
			// Return and don't requeue
			return reconcile.Result{}, nil
		}
		// Error reading the object - requeue the request.
		return reconcile.Result{}, err
	}

	if err := validate(instance); err != nil {
		instance.Logger().WithError(err).Error("Failed to validate")
		return reconcile.Result{}, err
	}

	// workaround for https://github.com/kubernetes-sigs/controller-runtime/issues/202
	// see also: https://github.com/kubernetes-sigs/controller-runtime/pull/212
	// once there's a version incorporating the PR above, the manual setting of the GKV can be removed
	instance.APIVersion = fmt.Sprintf("%s/%s", v1.SchemeGroupVersion.Group, v1.SchemeGroupVersion.Version)
	instance.Kind = "Jaeger"

	originalInstance := *instance

	opts := client.MatchingLabels(map[string]string{
		"app.kubernetes.io/instance":   instance.Name,
		"app.kubernetes.io/managed-by": "jaeger-operator",
	})
	list := &corev1.SecretList{}
	if err := r.client.List(context.Background(), opts, list); err != nil {
		return reconcile.Result{}, err
	}
	str := r.runStrategyChooser(instance, list.Items)

	logFields := instance.Logger().WithField("execution", execution)

	if err := r.apply(*instance, str); err != nil {
		logFields.WithError(err).Error("failed to apply the changes")
		return reconcile.Result{}, err
	}

	if !reflect.DeepEqual(originalInstance, *instance) {
		// we store back the changed CR, so that what is stored reflects what is being used
		if err := r.client.Update(context.Background(), instance); err != nil {
			logFields.WithError(err).Error("failed to store back the current CustomResource")
			return reconcile.Result{}, err
		}
	}

	log.WithFields(log.Fields{
		"namespace": request.Namespace,
		"instance":  request.Name,
		"execution": execution,
	}).Debug("Reconciling Jaeger completed - reschedule in 5 seconds")

	return reconcile.Result{}, nil
}

// validate validates CR before processing it
func validate(jaeger *v1.Jaeger) error {
	if jaeger.Spec.Storage.EsRollover.ReadTTL != "" {
		if _, err := time.ParseDuration(jaeger.Spec.Storage.EsRollover.ReadTTL); err != nil {
			return errors.Wrap(err, "failed to parse esRollover.readTTL to time.Duration")
		}
	}
	return nil
}

func (r *ReconcileJaeger) runStrategyChooser(instance *v1.Jaeger, secrets []corev1.Secret) strategy.S {
	if nil == r.strategyChooser {
		return defaultStrategyChooser(instance, secrets)
	}

	return r.strategyChooser(instance)
}

func defaultStrategyChooser(instance *v1.Jaeger, secrets []corev1.Secret) strategy.S {
	return strategy.For(context.Background(), instance, secrets)
}

func (r *ReconcileJaeger) apply(jaeger v1.Jaeger, str strategy.S) error {
	// secrets have to be created before ES - they are mounted to the ES pod
	if err := r.applySecrets(jaeger, str.Secrets()); err != nil {
		return err
	}

	elasticsearches := str.Elasticsearches()
	if strings.EqualFold(viper.GetString("es-provision"), v1.FlagProvisionElasticsearchTrue) {
		if err := r.applyElasticsearches(jaeger, elasticsearches); err != nil {
			return err
		}
	} else if len(elasticsearches) > 0 {
		log.WithFields(log.Fields{
			"namespace": jaeger.Namespace,
			"instance":  jaeger.Name,
		}).Warn("An Elasticsearch cluster should be provisioned, but provisioning is disabled for this Jaeger Operator")
	}

	// storage dependencies have to be deployed after ES is ready
	if err := r.handleDependencies(str); err != nil {
		return errors.Wrap(err, "failed to handler dependencies")
	}

	if err := r.applyAccounts(jaeger, str.Accounts()); err != nil {
		return err
	}

	if err := r.applyClusterRoleBindingBindings(jaeger, str.ClusterRoleBindings()); err != nil {
		return err
	}

	if err := r.applyConfigMaps(jaeger, str.ConfigMaps()); err != nil {
		return err
	}

	if err := r.applyCronJobs(jaeger, str.CronJobs()); err != nil {
		return err
	}

	if err := r.applyDaemonSets(jaeger, str.DaemonSets()); err != nil {
		return err
	}

	// seems counter intuitive to have services created *before* deployments,
	// but some resources used by deployments are created by services, such as TLS certs
	// for the oauth proxy, if one is used
	if err := r.applyServices(jaeger, str.Services()); err != nil {
		return err
	}

	if err := r.applyDeployments(jaeger, str.Deployments()); err != nil {
		return err
	}

	if strings.EqualFold(viper.GetString("platform"), v1.FlagPlatformOpenShift) {
		if err := r.applyRoutes(jaeger, str.Routes()); err != nil {
			return err
		}
	} else {
		if err := r.applyIngresses(jaeger, str.Ingresses()); err != nil {
			return err
		}
	}

	return nil
}
