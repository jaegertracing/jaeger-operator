package appsv1

import (
	"context"
	"encoding/json"
	"net/http"

	log "github.com/sirupsen/logrus"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/webhook"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"

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

	if inject.DeploymentNeeded(deploy, ns) {
		logger.Debug("update pod annotation")
		if deploy.Spec.Template.Labels == nil {
			deploy.Spec.Template.Labels = make(map[string]string, 0)
		}
		if _, ok := deploy.Spec.Template.Labels[inject.Label]; ok {
			logger.Warnf("pod template already provides label %s", inject.Label)
			return admission.Allowed("pod template already provides label")
		}
		if deploy.Spec.Template.Annotations == nil {
			deploy.Spec.Template.Annotations = make(map[string]string)
		}

		if _, ok := deploy.Spec.Template.Annotations[inject.Annotation]; !ok {
			deploy.Spec.Template.Annotations[inject.Annotation] = "true"
			deploy.Spec.Template.Annotations[inject.AnnotationManagedBy] = deploy.Name
		} else {
			logger.Warnf("annotation in PodTemplateSpec already present")
		}

		marshaledDeploy, err := json.Marshal(deploy)
		if err != nil {
			return admission.Errored(http.StatusInternalServerError, err)
		}

		return admission.PatchResponseFromRaw(req.Object.Raw, marshaledDeploy)
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
