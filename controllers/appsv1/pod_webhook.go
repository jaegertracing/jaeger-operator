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
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/webhook"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"

	v1 "github.com/jaegertracing/jaeger-operator/apis/v1"
	"github.com/jaegertracing/jaeger-operator/pkg/config/ca"
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
// +kubebuilder:webhook:path=/mutate-v1-pod,mutating=true,failurePolicy=fail,groups="",resources=pods,sideEffects=None,verbs=create,versions=v1,name=object.sidecar-injector.jaegertracing.io,admissionReviewVersions=v1
// +kubebuilder:webhook:path=/mutate-v1-pod,mutating=true,failurePolicy=fail,groups="",resources=pods,sideEffects=None,verbs=create,versions=v1,name=component.sidecar-injector.jaegertracing.io,admissionReviewVersions=v1
// +kubebuilder:webhook:path=/mutate-v1-pod,mutating=true,failurePolicy=fail,groups="",resources=pods,sideEffects=None,verbs=create,versions=v1,name=namespace.sidecar-injector.jaegertracing.io,admissionReviewVersions=v1

// podInjector inject Sidecar to Pods
type podInjector struct {
	client  client.Client
	decoder *admission.Decoder
}

// Handle adds a sidecar to a generated pod
func (p *podInjector) Handle(ctx context.Context, req admission.Request) admission.Response {
	logger := log.WithField("namespace", req.Namespace)

	pod := &corev1.Pod{}
	err := p.decoder.Decode(req, pod)
	if err != nil {
		logger.WithError(err).Error("failed to decode pod")
		return admission.Errored(http.StatusBadRequest, err)
	}
	pod.Namespace = req.Namespace // NOTE: namespace is not present in request body

	ns := &corev1.Namespace{}
	err = p.client.Get(ctx, types.NamespacedName{Name: req.Namespace}, ns)
	if err != nil { // we shouldn't fail if the namespace object can't be obtained
		msg := "failed to get the namespace for the pod, skipping injection based on namespace annotation"
		logger.WithError(err).Error(msg)
		return admission.Errored(http.StatusNotFound, err)
	}

	// find jaeger instances
	jaegers := &v1.JaegerList{}
	var opts []client.ListOption

	if viper.GetString(v1.ConfigOperatorScope) == v1.OperatorScopeNamespace {
		opts = append(opts, client.InNamespace(viper.GetString(v1.ConfigWatchNamespace)))
	}

	if err := p.client.List(ctx, jaegers, opts...); err != nil {
		logger.WithError(err).Error("failed to get the available Jaeger pods")
		return admission.Errored(http.StatusInternalServerError, err)
	}

	deploy := &appsv1.Deployment{}
	if _, exist := pod.GetAnnotations()[inject.Annotation]; !exist {
		// NOTE: If pod does not have an annotation, it is checked for backward
		// compatibility whether the corresponding deployment has an annotation.
		d, err := deploymentByPod(ctx, p.client, pod, req.Namespace)
		if err != nil {
			logger.WithError(err).Warn("failed to get the deployment of pod")
		} else {
			logger = logger.WithField("deployment", deploy.GetName())
			deploy = d
		}
	}

	if inject.PodNeeded(pod, ns) && !inject.DeploymentNeeded(deploy, &corev1.Namespace{}) {
		logger.Debug("sidecar needed")
		jaeger := inject.SelectForPod(pod, ns, jaegers)
		if jaeger != nil && jaeger.GetDeletionTimestamp() == nil {
			logger := logger.WithFields(log.Fields{
				"jaeger":           jaeger.Name,
				"jaeger-namespace": jaeger.Namespace,
			})

			if jaeger.Namespace != pod.Namespace {
				if err := reconcileConfigMaps(ctx, p.client, jaeger, pod); err != nil {
					const msg = "failed to reconcile config maps for the namespace"
					logger.WithError(err).Error(msg)
				}
			}

			// a suitable jaeger instance was found! let's inject a sidecar pointing to it then
			// Verified that jaeger instance was found and is not marked for deletion.
			{
				const msg = "injecting Jaeger Agent sidecar"
				logger.Info(msg)
			}

			pod := inject.SidecarPod(jaeger, pod)
			marshaledPod, err := json.Marshal(pod)
			if err != nil {
				return admission.Errored(http.StatusInternalServerError, err)
			}

			return admission.PatchResponseFromRaw(req.Object.Raw, marshaledPod)
		}

		logger.Info("no suitable Jaeger instances found to inject a sidecar")
	} else {
		logger.Debug("sidecar not needed")
	}

	return admission.Allowed("no action necessary")
}

// podInjector implements admission.DecoderInjector.
// A decoder will be automatically injected.

// InjectDecoder injects the decoder.
func (p *podInjector) InjectDecoder(d *admission.Decoder) error {
	p.decoder = d
	return nil
}

func deploymentByPod(
	ctx context.Context,
	c client.Client,
	pod *corev1.Pod,
	namespaceName string,
) (*appsv1.Deployment, error) {
	podRefList := pod.GetOwnerReferences()
	if len(podRefList) != 1 || podRefList[0].Kind != "ReplicaSet" {
		return nil, fmt.Errorf("missing single ReplicaSet as owner")
	}

	replicaName := podRefList[0].Name
	replicaSet := &appsv1.ReplicaSet{}
	logger := log.WithFields(log.Fields{
		"replicasetName":      replicaName,
		"replicasetNamespace": namespaceName,
	})
	logger.Infof("fetch replicaset")

	key := types.NamespacedName{Namespace: namespaceName, Name: replicaName}
	if err := c.Get(ctx, key, replicaSet); err != nil {
		return nil, fmt.Errorf("failed to get the available Pod ReplicaSet")
	}

	repRefList := replicaSet.GetOwnerReferences()
	if len(repRefList) != 1 || repRefList[0].Kind != "Deployment" {
		return nil, fmt.Errorf("could not determine deployment, number of owner: %d", len(repRefList))
	}

	logger.Infof("fetch deployment")
	deployName := repRefList[0].Name
	deployment := &appsv1.Deployment{}
	key = types.NamespacedName{Namespace: namespaceName, Name: deployName}
	if err := c.Get(ctx, key, deployment); err != nil {
		return nil, fmt.Errorf("failed to get the available Pod Deployment")
	}

	return deployment, nil
}

func reconcileConfigMaps(ctx context.Context, c client.Client, jaeger *v1.Jaeger, pod *corev1.Pod) error {
	cms := []*corev1.ConfigMap{}
	if cm := ca.GetTrustedCABundle(jaeger); cm != nil {
		cms = append(cms, cm)
	}
	if cm := ca.GetServiceCABundle(jaeger); cm != nil {
		cms = append(cms, cm)
	}

	for _, cm := range cms {
		// Update the namespace to be the same as the Pod being injected
		cm.Namespace = pod.Namespace
		if err := c.Create(ctx, cm); err != nil {
			if errors.IsAlreadyExists(err) {
				continue
			}
			return err
		}
	}

	return nil
}
