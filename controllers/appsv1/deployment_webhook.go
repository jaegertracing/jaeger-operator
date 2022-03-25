package appsv1

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"

	log "github.com/sirupsen/logrus"
	"github.com/spf13/viper"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/webhook"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"

	v1 "github.com/jaegertracing/jaeger-operator/apis/v1"
	"github.com/jaegertracing/jaeger-operator/pkg/inject"
)

var (
	_ admission.DecoderInjector = (*deploymentInterceptor)(nil)
	_ webhook.AdmissionHandler  = (*deploymentInterceptor)(nil)
)

// NewDeploymentInterceptorWebhook creates a new deployment mutating webhook to be registered
func NewDeploymentInterceptorWebhook(c client.Client) webhook.AdmissionHandler {
	return &deploymentInterceptor{
		client: c,
	}
}

// You need to ensure the path here match the path in the marker.
// +kubebuilder:webhook:path=/mutate-v1-deployment,mutating=true,failurePolicy=fail,groups="apps",resources=deployments,sideEffects=None,verbs=create,versions=v1,name=deployment.sidecar-injector.jaegertracing.io,admissionReviewVersions=v1

// +kubebuilder:rbac:groups=core,resources=namespaces,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=core,resources=namespaces/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=apps,resources=deployments,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=apps,resources=deployments/status,verbs=get;update;patch

// deploymentInterceptor label pods if Sidecar is specified in deployment
type deploymentInterceptor struct {
	client  client.Client
	decoder *admission.Decoder
}

// Handle adds a label to a generated pod if deployment or namespace provide annotaion
func (d *deploymentInterceptor) Handle(ctx context.Context, req admission.Request) admission.Response {
	logger := log.WithField("namespace", req.Namespace)
	logger.Debug("verify deployment")

	deploy := &appsv1.Deployment{}
	err := d.decoder.Decode(req, deploy)
	if err != nil {
		logger.WithError(err).Error("failed to decode deployment")
		return admission.Errored(http.StatusBadRequest, err)
	}

	ns := &corev1.Namespace{}
	err = d.client.Get(ctx, types.NamespacedName{Name: req.Namespace}, ns)
	if err != nil { // we shouldn't fail if the namespace object can't be obtained
		msg := "failed to get the namespace for the pod, skipping injection based on namespace annotation"
		logger.WithError(err).Error(msg)
		return admission.Errored(http.StatusNotFound, err)
	}

	jaegers := &v1.JaegerList{}
	opts := []client.ListOption{}

	if viper.GetString(v1.ConfigOperatorScope) == v1.OperatorScopeNamespace {
		opts = append(opts, client.InNamespace(viper.GetString(v1.ConfigWatchNamespace)))
	}

	if err := d.client.List(ctx, jaegers, opts...); err != nil {
		logger.WithError(err).Error("failed to get the available Jaeger pods")
		return admission.Errored(http.StatusInternalServerError, err)
	}

	if inject.DeploymentNeeded(deploy, ns) {
		logger.Debug("sidecar needed, check if pod has annotation")
		if _, ok := deploy.Spec.Template.Annotations[inject.Annotation]; ok {
			logger.Warnf("pod template already provides annotation %s", inject.Annotation)
			return admission.Allowed("pod template already provides annotation")
		}
		jaeger := inject.Select(deploy, ns, jaegers)
		if jaeger != nil && jaeger.GetDeletionTimestamp() == nil {
			logger := logger.WithFields(log.Fields{
				"jaeger":           jaeger.Name,
				"jaeger-namespace": jaeger.Namespace,
			})
			logger.Info("assign config map")

			if jaeger.Namespace != deploy.Namespace {
				fmt.Println("TODO")
				// TODO
				// if err := reconcileConfigMaps(ctx, d.client, jaeger, nil); err != nil {
				//	msg := "failed to reconcile config maps for the namespace"
				//	logger.WithError(err).Error(msg)
				// }
			}

			// a suitable jaeger instance was found! let's inject a sidecar pointing to it then
			// Verified that jaeger instance was found and is not marked for deletion.
			{
				const msg = "injecting Jaeger Agent sidecar"
				logger.Info(msg)
			}

			logger.Info("inject sidecar")
			deploy = inject.Sidecar(jaeger, deploy)
			logger.Info("marshal")
			marshaledDeploy, err := json.Marshal(deploy)
			if err != nil {
				return admission.Errored(http.StatusInternalServerError, err)
			}

			logger.Info("patch")
			return admission.PatchResponseFromRaw(req.Object.Raw, marshaledDeploy)
		}
		logger.Info("no suitable Jaeger instances found to inject a sidecar")
	}

	return admission.Allowed("no need to update PodTemplateSpec")
}

// deploymentInterceptor implements admission.DecoderInjector.
// A decoder will be automatically injected.

// InjectDecoder injects the decoder.
func (d *deploymentInterceptor) InjectDecoder(decoder *admission.Decoder) error {
	d.decoder = decoder
	return nil
}
