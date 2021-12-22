package appsv1

import (
	"context"
	"encoding/json"
	"net/http"

	log "github.com/sirupsen/logrus"
	"github.com/spf13/viper"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/webhook"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"

	v1 "github.com/jaegertracing/jaeger-operator/apis/v1"
	"github.com/jaegertracing/jaeger-operator/pkg/inject"
)

var (
	_ admission.DecoderInjector = (*podInjector)(nil)
	_ webhook.AdmissionHandler  = (*podInjector)(nil)
)

// NewPodInjectorWebhook creates a new pod injector webhook to be registered
func NewPodInjectorWebhook(c client.Client) webhook.AdmissionHandler {
	return &podInjector{
		client: c,
	}
}

// You need to ensure the path here match the path in the marker.
// +kubebuilder:webhook:path=/mutate-v1-pod,mutating=true,failurePolicy=fail,groups="",resources=pods,sideEffects=None,verbs=create,versions=v1,name=object.sidecar-injector.jaegertracing.io,admissionReviewVersions=v1;v1beta1
// +kubebuilder:webhook:path=/mutate-v1-pod,mutating=true,failurePolicy=fail,groups="",resources=pods,sideEffects=None,verbs=create,versions=v1,name=namespace.sidecar-injector.jaegertracing.io,admissionReviewVersions=v1;v1beta1
// +kubebuilder:webhook:path=/mutate-v1-pod,mutating=true,failurePolicy=fail,groups="",resources=pods,sideEffects=None,verbs=create,versions=v1,name=component.sidecar-injector.jaegertracing.io,admissionReviewVersions=v1;v1beta1

// podInjector inject Sidecar to Pods
type podInjector struct {
	client  client.Client
	decoder *admission.Decoder
}

// Handle adds a sidecar to a generated pod
func (pi *podInjector) Handle(ctx context.Context, req admission.Request) admission.Response {
	pod := &corev1.Pod{}

	logger := log.WithFields(log.Fields{
		"namespace": req.Namespace,
		"name":      req.Name,
	})

	err := pi.decoder.Decode(req, pod)
	if err != nil {
		logger.WithError(err).Error("failed to decode pod")
		return admission.Errored(http.StatusBadRequest, err)
	}

	ns := &corev1.Namespace{}
	err = pi.client.Get(ctx, types.NamespacedName{Name: req.Namespace}, ns)
	// we shouldn't fail if the namespace object can't be obtained
	if err != nil {
		msg := "failed to get the namespace for the pod, skipping injection based on namespace annotation"
		logger.WithError(err).Debug(msg)
	}

	// find jaeger instances
	jaegers := &v1.JaegerList{}
	var opts []client.ListOption

	if viper.GetString(v1.ConfigOperatorScope) == v1.OperatorScopeNamespace {
		opts = append(opts, client.InNamespace(viper.GetString(v1.ConfigWatchNamespace)))
	}

	if err := pi.client.List(ctx, jaegers, opts...); err != nil {
		logger.WithError(err).Error("failed to get the available Jaeger pods")
		return admission.Errored(http.StatusInternalServerError, err)
	}

	if inject.PodNeeded(pod, ns) {
		jaeger := inject.Select(pod, ns, jaegers)
		if jaeger != nil && jaeger.GetDeletionTimestamp() == nil {
			logger := logger.WithFields(log.Fields{
				"jaeger":           jaeger.Name,
				"jaeger-namespace": jaeger.Namespace,
			})

			// a suitable jaeger instance was found! let's inject a sidecar pointing to it then
			// Verified that jaeger instance was found and is not marked for deletion.
			{
				msg := "injecting Jaeger Agent sidecar"
				logger.Info(msg)
			}

			pod := inject.SidecarPod(jaeger, pod)
			marshaledPod, err := json.Marshal(pod)
			if err != nil {
				return admission.Errored(http.StatusInternalServerError, err)
			}

			return admission.PatchResponseFromRaw(req.Object.Raw, marshaledPod)
		}

		msg := "no suitable Jaeger instances found to inject a sidecar"
		logger.Debug(msg)
	}

	return admission.Allowed("jaeger is not necessary")
}

// podInjector implements admission.DecoderInjector.
// A decoder will be automatically injected.

// InjectDecoder injects the decoder.
func (pi *podInjector) InjectDecoder(d *admission.Decoder) error {
	pi.decoder = d
	return nil
}
