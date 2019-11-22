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
	"go.opentelemetry.io/otel/api/key"
	"go.opentelemetry.io/otel/global"
	"google.golang.org/grpc/codes"
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
	"github.com/jaegertracing/jaeger-operator/pkg/tracing"
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
	strategyChooser func(context.Context, *v1.Jaeger) strategy.S
}

// Reconcile reads that state of the cluster for a Jaeger object and makes changes based on the state read
// and what is in the Jaeger.Spec
// Note:
// The Controller will requeue the Request to be processed again if the returned error is non-nil or
// Result.Requeue is true, otherwise upon completion it will remove the work from the queue.
func (r *ReconcileJaeger) Reconcile(request reconcile.Request) (reconcile.Result, error) {
	ctx := context.Background()

	tracer := global.TraceProvider().GetTracer(v1.ReconciliationTracer)
	ctx, span := tracer.Start(ctx, "reconcile")
	defer span.End()

	execution := time.Now().UTC()

	span.SetAttributes(key.String("name", request.Name), key.String("namespace", request.Namespace))
	log.WithFields(log.Fields{
		"namespace": request.Namespace,
		"instance":  request.Name,
		"execution": execution,
	}).Debug("Reconciling Jaeger")

	// Fetch the Jaeger instance
	instance := &v1.Jaeger{}
	err := r.client.Get(ctx, request.NamespacedName, instance)
	if err != nil {
		if k8serrors.IsNotFound(err) {
			// Request object not found, could have been deleted after reconcile request.
			// Owned objects are automatically garbage collected. For additional cleanup logic use finalizers.
			// Return and don't requeue
			span.SetStatus(codes.NotFound)
			return reconcile.Result{}, nil
		}
		// Error reading the object - requeue the request.
		return reconcile.Result{}, tracing.HandleError(err, span)
	}

	logFields := instance.Logger().WithField("execution", execution)

	if err := validate(instance); err != nil {
		instance.Logger().WithError(err).Error("Failed to validate")
		span.SetAttribute(key.String("error", err.Error()))
		span.SetStatus(codes.InvalidArgument)
		return reconcile.Result{}, err
	}

	// note: we need a namespace-scoped owner identity, which makes the `OwnerReference`
	// not suitable for this purpose
	identity := viper.GetString(v1.ConfigIdentity)
	if val, found := instance.Labels[v1.LabelOperatedBy]; found {
		if val != identity {
			// if we are not the ones managing this instance, skip the reconciliation
			log.WithFields(log.Fields{
				"our-identity":   identity,
				"owner-identity": val,
			}).Debug("skipping CR as we are not owners")
			return reconcile.Result{}, nil
		}
	} else {
		if instance.Labels == nil {
			instance.Labels = map[string]string{}
		}

		instance.Labels[v1.LabelOperatedBy] = identity
		if err := r.client.Update(ctx, instance); err != nil {
			logFields.WithField(
				"operator-identity", identity,
			).WithError(err).Error("failed to set this operator as the manager of the instance")
			return reconcile.Result{}, tracing.HandleError(err, span)
		}

		logFields.WithField("operator-identity", identity).Debug("configured this operator as the owner of the CR")
		return reconcile.Result{}, nil
	}

	// workaround for https://github.com/jaegertracing/jaeger-operator/pull/558
	instance.APIVersion = fmt.Sprintf("%s/%s", v1.SchemeGroupVersion.Group, v1.SchemeGroupVersion.Version)
	instance.Kind = "Jaeger"

	originalInstance := *instance

	opts := client.MatchingLabels(map[string]string{
		"app.kubernetes.io/instance":   instance.Name,
		"app.kubernetes.io/managed-by": "jaeger-operator",
	})
	list := &corev1.SecretList{}
	if err := r.client.List(ctx, list, opts); err != nil {
		return reconcile.Result{}, tracing.HandleError(err, span)
	}
	str := r.runStrategyChooser(ctx, instance, list.Items)

	updated, err := r.apply(ctx, *instance, str)
	if err != nil {
		logFields.WithError(err).Error("failed to apply the changes")
		return reconcile.Result{}, tracing.HandleError(err, span)
	}
	instance = &updated

	if !reflect.DeepEqual(originalInstance, *instance) {
		// we store back the changed CR, so that what is stored reflects what is being used
		if err := r.client.Update(ctx, instance); err != nil {
			logFields.WithError(err).Error("failed to store back the current CustomResource")
			return reconcile.Result{}, tracing.HandleError(err, span)
		}
	}

	log.WithFields(log.Fields{
		"namespace": request.Namespace,
		"instance":  request.Name,
		"execution": execution,
	}).Debug("Reconciling Jaeger completed")

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

func (r *ReconcileJaeger) runStrategyChooser(ctx context.Context, instance *v1.Jaeger, secrets []corev1.Secret) strategy.S {
	if nil == r.strategyChooser {
		return defaultStrategyChooser(ctx, instance, secrets)
	}

	return r.strategyChooser(ctx, instance)
}

func defaultStrategyChooser(ctx context.Context, instance *v1.Jaeger, secrets []corev1.Secret) strategy.S {
	return strategy.For(ctx, instance, secrets)
}

func (r *ReconcileJaeger) apply(ctx context.Context, jaeger v1.Jaeger, str strategy.S) (v1.Jaeger, error) {
	tracer := global.TraceProvider().GetTracer(v1.ReconciliationTracer)
	ctx, span := tracer.Start(ctx, "apply")
	defer span.End()

	jaeger, err := r.applyUpgrades(ctx, jaeger)
	if err != nil {
		return jaeger, tracing.HandleError(err, span)
	}

	// secrets have to be created before ES - they are mounted to the ES pod
	if err := r.applySecrets(ctx, jaeger, str.Secrets()); err != nil {
		return jaeger, tracing.HandleError(err, span)
	}

	elasticsearches := str.Elasticsearches()
	if strings.EqualFold(viper.GetString("es-provision"), v1.FlagProvisionElasticsearchYes) {
		if err := r.applyElasticsearches(ctx, jaeger, elasticsearches); err != nil {
			return jaeger, tracing.HandleError(err, span)
		}
	} else if len(elasticsearches) > 0 {
		log.WithFields(log.Fields{
			"namespace": jaeger.Namespace,
			"instance":  jaeger.Name,
		}).Warn("An Elasticsearch cluster should be provisioned, but provisioning is disabled for this Jaeger Operator")
	}

	kafkas := str.Kafkas()
	kafkaUsers := str.KafkaUsers()
	if strings.EqualFold(viper.GetString("kafka-provision"), v1.FlagProvisionKafkaYes) {
		if err := r.applyKafkas(ctx, jaeger, kafkas); err != nil {
			return jaeger, tracing.HandleError(err, span)
		}

		if err := r.applyKafkaUsers(ctx, jaeger, kafkaUsers); err != nil {
			return jaeger, tracing.HandleError(err, span)
		}
	} else if len(kafkas) > 0 || len(kafkaUsers) > 0 {
		log.WithFields(log.Fields{
			"namespace": jaeger.Namespace,
			"instance":  jaeger.Name,
		}).Warn("A Kafka cluster should be provisioned, but provisioning is disabled for this Jaeger Operator")
	}

	// storage dependencies have to be deployed after ES is ready
	if err := r.handleDependencies(ctx, str); err != nil {
		return jaeger, tracing.HandleError(err, span)
	}

	if err := r.applyAccounts(ctx, jaeger, str.Accounts()); err != nil {
		return jaeger, tracing.HandleError(err, span)
	}

	if err := r.applyClusterRoleBindingBindings(ctx, jaeger, str.ClusterRoleBindings()); err != nil {
		return jaeger, tracing.HandleError(err, span)
	}

	if err := r.applyConfigMaps(ctx, jaeger, str.ConfigMaps()); err != nil {
		return jaeger, tracing.HandleError(err, span)
	}

	if err := r.applyCronJobs(ctx, jaeger, str.CronJobs()); err != nil {
		return jaeger, tracing.HandleError(err, span)
	}

	if err := r.applyDaemonSets(ctx, jaeger, str.DaemonSets()); err != nil {
		return jaeger, tracing.HandleError(err, span)
	}

	// seems counter intuitive to have services created *before* deployments,
	// but some resources used by deployments are created by services, such as TLS certs
	// for the oauth proxy, if one is used
	if err := r.applyServices(ctx, jaeger, str.Services()); err != nil {
		return jaeger, tracing.HandleError(err, span)
	}

	if err := r.applyDeployments(ctx, jaeger, str.Deployments()); err != nil {
		return jaeger, tracing.HandleError(err, span)
	}

	if strings.EqualFold(viper.GetString("platform"), v1.FlagPlatformOpenShift) {
		if err := r.applyRoutes(ctx, jaeger, str.Routes()); err != nil {
			return jaeger, tracing.HandleError(err, span)
		}
	} else {
		if err := r.applyIngresses(ctx, jaeger, str.Ingresses()); err != nil {
			return jaeger, tracing.HandleError(err, span)
		}
	}

	return jaeger, nil
}
