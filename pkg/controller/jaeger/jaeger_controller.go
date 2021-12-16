package jaeger

import (
	"context"
	"fmt"
	"reflect"
	"strings"
	"time"

	osv1 "github.com/openshift/api/route/v1"
	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/viper"
	"go.opentelemetry.io/otel"
	otelattribute "go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	corev1 "k8s.io/api/core/v1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	v1 "github.com/jaegertracing/jaeger-operator/apis/v1"
	"github.com/jaegertracing/jaeger-operator/pkg/storage"
	"github.com/jaegertracing/jaeger-operator/pkg/strategy"
	"github.com/jaegertracing/jaeger-operator/pkg/tracing"
)

// ReconcileJaeger reconciles a Jaeger object
type ReconcileJaeger struct {
	// This client, initialized using mgr.Client() above, is a split client
	// that reads objects from the cache and writes to the apiserver
	client          client.Client
	rClient         client.Reader
	scheme          *runtime.Scheme
	strategyChooser func(context.Context, *v1.Jaeger) strategy.S
}

// New creates new jaeger controller
func New(client client.Client, clientReader client.Reader, scheme *runtime.Scheme) *ReconcileJaeger {
	return &ReconcileJaeger{
		client:          client,
		rClient:         clientReader,
		scheme:          scheme,
		strategyChooser: defaultStrategyChooser,
	}
}

// Reconcile reads that state of the cluster for a Jaeger object and makes changes based on the state read
// and what is in the Jaeger.Spec
// Note:
// The Controller will requeue the Request to be processed again if the returned error is non-nil or
// Result.Requeue is true, otherwise upon completion it will remove the work from the queue.
func (r *ReconcileJaeger) Reconcile(request reconcile.Request) (reconcile.Result, error) {
	ctx := context.Background()

	tracer := otel.GetTracerProvider().Tracer(v1.ReconciliationTracer)
	ctx, span := tracer.Start(ctx, "reconcile")
	defer span.End()

	execution := time.Now().UTC()

	span.SetAttributes(
		otelattribute.String("name", request.Name),
		otelattribute.String("namespace", request.Namespace))
	log.WithFields(log.Fields{
		"namespace": request.Namespace,
		"instance":  request.Name,
		"execution": execution,
	}).Debug("Reconciling Jaeger")

	// Fetch the Jaeger instance
	instance := &v1.Jaeger{}
	err := r.rClient.Get(ctx, request.NamespacedName, instance)
	if err != nil {
		if k8serrors.IsNotFound(err) {
			// Request object not found, could have been deleted after reconcile request.
			// Owned objects are automatically garbage collected. For additional cleanup logic use finalizers.
			// Return and don't requeue
			if err := r.cleanConfigMaps(ctx, request.Name); err != nil {
				return reconcile.Result{}, tracing.HandleError(err, span)
			}
			return reconcile.Result{}, nil
		}
		// Error reading the object - requeue the request.
		return reconcile.Result{}, tracing.HandleError(err, span)
	}

	logFields := instance.Logger().WithField("execution", execution)

	if err := validate(instance); err != nil {
		instance.Logger().WithError(err).Error("failed to validate")
		span.SetStatus(codes.Error, err.Error())
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
			}).Trace("skipping CR as we are not owners")
			return reconcile.Result{}, nil
		}
	} else {
		if instance.Labels == nil {
			instance.Labels = map[string]string{}
		}

		instance.Labels[v1.LabelOperatedBy] = identity
		if err := r.client.Update(ctx, instance); err != nil {
			// update the status to "Failed"
			instance.Status.Phase = v1.JaegerPhaseFailed
			if err := r.client.Status().Update(ctx, instance); err != nil {
				// we let it return the real error later
				logFields.WithError(err).Error("failed to store the failed status into the current CustomResource after setting the identity")
			}

			logFields.WithField(
				"operator-identity", identity,
			).WithError(err).Error("failed to set this operator as the manager of the instance")
			return reconcile.Result{}, tracing.HandleError(err, span)
		}

		logFields.WithField("operator-identity", identity).Debug("configured this operator as the owner of the CR")
		return reconcile.Result{}, nil
	}

	// workaround for https://github.com/jaegertracing/jaeger-operator/pull/558
	instance.APIVersion = fmt.Sprintf("%s/%s", v1.GroupVersion.Group, v1.GroupVersion.Version)
	instance.Kind = "Jaeger"

	originalInstance := *instance

	str := r.runStrategyChooser(ctx, instance)

	updated, err := r.apply(ctx, *instance, str)
	if err != nil {
		logFields.WithError(err).Error("failed to apply the changes")
		// We need to make sure that the instance still exist, so we can update the status to failed. Could happen that the instance
		// is deleted before the deployment is stable, in this case apply will return "deployment has been removed" error,
		// but we can't write that status on an unexisting instance.
		if k8serrors.IsNotFound(r.rClient.Get(ctx, request.NamespacedName, &v1.Jaeger{})) {
			return reconcile.Result{}, nil
		}
		// update the status to "Failed"
		instance.Status.Phase = v1.JaegerPhaseFailed
		if err := r.client.Status().Update(ctx, instance); err != nil {
			// we let it return the real error later
			logFields.WithError(err).Error("failed to store the failed status into the current CustomResource after the reconciliation")
		}

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

	// set the status version to the updated instance version if versions doesn't match
	if updated.Status.Version != originalInstance.Status.Version || instance.Status.Phase != v1.JaegerPhaseRunning {
		instance.Status.Phase = v1.JaegerPhaseRunning
		instance.Status.Version = updated.Status.Version
		if err := r.client.Status().Update(ctx, instance); err != nil {
			logFields.WithError(err).Error("failed to store the running status into the current CustomResource")
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

func (r *ReconcileJaeger) runStrategyChooser(ctx context.Context, instance *v1.Jaeger) strategy.S {
	if nil == r.strategyChooser {
		return defaultStrategyChooser(ctx, instance)
	}

	return r.strategyChooser(ctx, instance)
}

func defaultStrategyChooser(ctx context.Context, instance *v1.Jaeger) strategy.S {
	return strategy.For(ctx, instance)
}

func (r *ReconcileJaeger) apply(ctx context.Context, jaeger v1.Jaeger, str strategy.S) (v1.Jaeger, error) {
	tracer := otel.GetTracerProvider().Tracer(v1.ReconciliationTracer)
	ctx, span := tracer.Start(ctx, "apply")
	defer span.End()

	jaeger, err := r.applyUpgrades(ctx, jaeger)
	if err != nil {
		return jaeger, tracing.HandleError(err, span)
	}

	// ES cert handling requires secrets from environment
	// therefore running this here and not in the strategy
	if storage.ShouldDeployElasticsearch(jaeger.Spec.Storage) {
		opts := client.MatchingLabels(map[string]string{
			"app.kubernetes.io/instance":   jaeger.Name,
			"app.kubernetes.io/managed-by": "jaeger-operator",
		})
		secrets := &corev1.SecretList{}
		if err := r.rClient.List(ctx, secrets, opts); err != nil {
			jaeger.Status.Phase = v1.JaegerPhaseFailed
			if err := r.client.Status().Update(ctx, &jaeger); err != nil {
				// we let it return the real error later
				jaeger.Logger().WithError(err).Error("failed to store the failed status into the current CustomResource after preconditions")
			}
			return jaeger, tracing.HandleError(err, span)
		}
		secretsForNamespace := r.getSecretsForNamespace(secrets.Items, jaeger.Namespace)

		es := &storage.ElasticsearchDeployment{Jaeger: &jaeger, CertScript: "./scripts/cert_generation.sh", Secrets: secretsForNamespace}
		err = es.CreateCerts()
		if err != nil {
			es.Jaeger.Logger().WithError(err).Error("failed to create Elasticsearch certificates, Elasticsearch won't be deployed")
			return jaeger, err
		}
		str = str.WithSecrets(append(str.Secrets(), es.ExtractSecrets()...))
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

	if err := r.applyAccounts(ctx, jaeger, str.Accounts()); err != nil {
		return jaeger, tracing.HandleError(err, span)
	}

	// storage dependencies have to be deployed after ES is ready
	if err := r.handleDependencies(ctx, str); err != nil {
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
		routes := osv1.RouteList{}
		err = r.rClient.List(ctx, &routes, client.InNamespace(jaeger.Namespace))
		if err == nil {
			if err := r.applyConsoleLinks(ctx, jaeger, str.ConsoleLinks(routes.Items)); err != nil {
				jaeger.Logger().WithError(tracing.HandleError(err, span)).Warn("failed to reconcile console links")
			}
		} else {
			jaeger.Logger().WithError(tracing.HandleError(err, span)).Warn("failed to obtain a list of routes to reconcile consolelinks")
		}
	} else {
		if err := r.applyIngresses(ctx, jaeger, str.Ingresses()); err != nil {
			return jaeger, tracing.HandleError(err, span)
		}
	}

	if err := r.applyHorizontalPodAutoscalers(ctx, jaeger, str.HorizontalPodAutoscalers()); err != nil {
		// we don't want to fail the whole reconciliation when this fails
		jaeger.Logger().WithError(tracing.HandleError(err, span)).Warn("failed to reconcile pod autoscalers")
		return jaeger, nil
	}

	// we apply the daemonsets after everything else, to increase the chances of having services and deployments
	// ready by the time the daemonset is started, so that it gets at least one collector to connect to
	if err := r.applyDaemonSets(ctx, jaeger, str.DaemonSets()); err != nil {
		return jaeger, tracing.HandleError(err, span)
	}

	return jaeger, nil
}

func (r ReconcileJaeger) getSecretsForNamespace(secrets []corev1.Secret, namespace string) []corev1.Secret {
	var secretsForNamespace []corev1.Secret
	for _, secret := range secrets {
		if secret.Namespace == namespace {
			secretsForNamespace = append(secretsForNamespace, secret)
		}
	}
	return secretsForNamespace
}
