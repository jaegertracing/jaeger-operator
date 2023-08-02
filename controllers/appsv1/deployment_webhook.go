package appsv1

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"

	"github.com/spf13/viper"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/webhook"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"

	v1 "github.com/jaegertracing/jaeger-operator/apis/v1"
	"github.com/jaegertracing/jaeger-operator/pkg/config/ca"
	"github.com/jaegertracing/jaeger-operator/pkg/inject"
	"github.com/jaegertracing/jaeger-operator/pkg/tracing"
)

var _ webhook.AdmissionHandler = (*deploymentInterceptor)(nil)

// NewDeploymentInterceptorWebhook creates a new deployment mutating webhook to be registered
func NewDeploymentInterceptorWebhook(c client.Client, decoder *admission.Decoder) webhook.AdmissionHandler {
	return &deploymentInterceptor{
		client:  c,
		decoder: decoder,
	}
}

// You need to ensure the path here match the path in the marker.
// +kubebuilder:webhook:path=/mutate-v1-deployment,mutating=true,failurePolicy=ignore,groups="apps",resources=deployments,sideEffects=None,verbs=create;update,versions=v1,name=deployment.sidecar-injector.jaegertracing.io,admissionReviewVersions=v1

// +kubebuilder:rbac:groups=core,resources=namespaces,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=core,resources=namespaces/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=apps,resources=deployments,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=apps,resources=deployments/status,verbs=get;update;patch

// deploymentInterceptor label pods if Sidecar is specified in deployment
type deploymentInterceptor struct {
	client  client.Client
	decoder *admission.Decoder
}

func (d *deploymentInterceptor) shouldHandleDeployment(req admission.Request) bool {
	if namespaces := viper.GetString(v1.ConfigWatchNamespace); namespaces != v1.WatchAllNamespaces {
		for _, ns := range strings.Split(namespaces, ",") {
			if strings.EqualFold(ns, req.Namespace) {
				return true
			}
		}
		return false
	}
	return true
}

// Handle adds a label to a generated pod if deployment or namespace provide annotaion
func (d *deploymentInterceptor) Handle(ctx context.Context, req admission.Request) admission.Response {
	tracer := otel.GetTracerProvider().Tracer(v1.ReconciliationTracer)
	ctx, span := tracer.Start(ctx, "reconcileDeployment")
	span.SetAttributes(
		attribute.String("kind", req.Kind.String()),
		attribute.String("name", req.Name),
		attribute.String("namespace", req.Namespace),
	)

	if !d.shouldHandleDeployment(req) {
		return admission.Allowed("not watching in namespace, we do not touch the deployment")
	}

	defer span.End()

	logger := log.Log.WithValues("namespace", req.Namespace)
	logger.V(-1).Info("verify deployment")

	dep := &appsv1.Deployment{}
	err := d.decoder.Decode(req, dep)
	if err != nil {
		logger.Error(err, "failed to decode deployment")
		return admission.Errored(http.StatusBadRequest, err)
	}

	if dep.Labels["app"] == "jaeger" && dep.Labels["app.kubernetes.io/component"] != "query" {
		// Don't touch jaeger deployments
		return admission.Allowed("is jaeger deployment, we do not touch it")
	}

	ns := &corev1.Namespace{}
	err = d.client.Get(ctx, types.NamespacedName{Name: req.Namespace}, ns)
	// we shouldn't fail if the namespace object can't be obtained
	if err != nil {
		msg := "failed to get the namespace for the deployment, skipping injection based on namespace annotation"
		logger.Error(err, msg)
		span.AddEvent(msg, trace.WithAttributes(attribute.String("error", err.Error())))
	}

	jaegers := &v1.JaegerList{}
	opts := []client.ListOption{}

	if viper.GetString(v1.ConfigOperatorScope) == v1.OperatorScopeNamespace {
		opts = append(opts, client.InNamespace(viper.GetString(v1.ConfigWatchNamespace)))
	}

	if err := d.client.List(ctx, jaegers, opts...); err != nil {
		logger.Error(err, "failed to get the available Jaeger pods")
		return admission.Errored(http.StatusInternalServerError, tracing.HandleError(err, span))
	}

	if inject.Needed(dep, ns) {
		jaeger := inject.Select(dep, ns, jaegers)
		if jaeger != nil && jaeger.GetDeletionTimestamp() == nil {
			logger := logger.WithValues(
				"jaeger", jaeger.Name,
				"jaeger-namespace", jaeger.Namespace,
			)
			if jaeger.Namespace != dep.Namespace {
				if err := reconcileConfigMaps(ctx, d.client, jaeger, dep); err != nil {
					const msg = "failed to reconcile config maps for the namespace"
					logger.Error(err, msg)
					span.AddEvent(msg)
				}
			}

			// a suitable jaeger instance was found! let's inject a sidecar pointing to it then
			// Verified that jaeger instance was found and is not marked for deletion.
			{
				msg := "injecting Jaeger Agent sidecar"
				logger.Info(msg)
				span.AddEvent(msg)
			}

			envConfigMaps := corev1.ConfigMapList{}
			d.client.List(ctx, &envConfigMaps, client.InNamespace(dep.Namespace))
			dep = inject.Sidecar(jaeger, dep, inject.WithEnvFromConfigMaps(inject.GetConfigMapsMatchedEnvFromInDeployment(*dep, envConfigMaps.Items)))
			marshaledDeploy, err := json.Marshal(dep)
			if err != nil {
				return admission.Errored(http.StatusInternalServerError, tracing.HandleError(err, span))
			}

			return admission.PatchResponseFromRaw(req.Object.Raw, marshaledDeploy)
		}

		const msg = "no suitable Jaeger instances found to inject a sidecar"
		span.AddEvent(msg)
		logger.V(-1).Info(msg)
		return admission.Allowed(msg)
	}

	if hasAgent, _ := inject.HasJaegerAgent(dep); hasAgent {
		if _, hasLabel := dep.Labels[inject.Label]; hasLabel {
			const msg = "remove sidecar"
			logger.Info(msg)
			span.AddEvent(msg)
			inject.CleanSidecar(dep.Labels[inject.Label], dep)
			marshaledDeploy, err := json.Marshal(dep)
			if err != nil {
				return admission.Errored(http.StatusInternalServerError, tracing.HandleError(err, span))
			}

			return admission.PatchResponseFromRaw(req.Object.Raw, marshaledDeploy)
		}
	}
	return admission.Allowed("no action needed")
}

// deploymentInterceptor implements admission.DecoderInjector.
// A decoder will be automatically injected.

// InjectDecoder injects the decoder.
func (d *deploymentInterceptor) InjectDecoder(decoder *admission.Decoder) error {
	d.decoder = decoder
	return nil
}

func reconcileConfigMaps(ctx context.Context, cl client.Client, jaeger *v1.Jaeger, dep *appsv1.Deployment) error {
	tracer := otel.GetTracerProvider().Tracer(v1.ReconciliationTracer)
	ctx, span := tracer.Start(ctx, "reconcileConfigMaps")
	defer span.End()

	cms := []*corev1.ConfigMap{}
	if cm := ca.GetTrustedCABundle(jaeger); cm != nil {
		cms = append(cms, cm)
	}
	if cm := ca.GetServiceCABundle(jaeger); cm != nil {
		cms = append(cms, cm)
	}

	for _, cm := range cms {
		if err := reconcileConfigMap(ctx, cl, cm, dep); err != nil {
			return tracing.HandleError(err, span)
		}
	}

	return nil
}

func reconcileConfigMap(ctx context.Context, cl client.Client, cm *corev1.ConfigMap, dep *appsv1.Deployment) error {
	tracer := otel.GetTracerProvider().Tracer(v1.ReconciliationTracer)
	ctx, span := tracer.Start(ctx, "reconcileConfigMap")
	defer span.End()

	// Update the namespace to be the same as the Deployment being injected
	cm.Namespace = dep.Namespace
	span.SetAttributes(attribute.String("name", cm.Name), attribute.String("namespace", cm.Namespace))

	if err := cl.Create(ctx, cm); err != nil {
		if errors.IsAlreadyExists(err) {
			span.AddEvent("config map exists already")
		} else {
			return tracing.HandleError(err, span)
		}
	}

	return nil
}
