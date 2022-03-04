package appsv1

import (
	"context"
	"encoding/json"
	"errors"
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
	logger.Debugf("verify pod")

	ns := &corev1.Namespace{}
	err = pi.client.Get(ctx, types.NamespacedName{Name: req.Namespace}, ns)
	// we shouldn't fail if the namespace object can't be obtained
	if err != nil {
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

	if err := pi.client.List(ctx, jaegers, opts...); err != nil {
		logger.WithError(err).Error("failed to get the available Jaeger pods")
		return admission.Errored(http.StatusInternalServerError, err)
	}

	deploy, err := deploymentByPod(ctx, pi.client, pod, req.Namespace)
	if err != nil {
		logger.WithError(err).Error("failed to get the deployment of pod")
		return admission.Errored(http.StatusNotFound, err)
	}

	if inject.PodNeeded(pod, ns) || inject.DeploymentNeeded(deploy, ns) {
		logger.Debugf("add sidecar")
		jaeger := inject.SelectForPod(pod, deploy, ns, jaegers)
		if jaeger != nil && jaeger.GetDeletionTimestamp() == nil {
			logger := logger.WithFields(log.Fields{
				"jaeger":           jaeger.Name,
				"jaeger-namespace": jaeger.Namespace,
			})

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

		logger.Debug("no suitable Jaeger instances found to inject a sidecar")
	} else {
		logger.Debugf("remove sidecar")
		if ok, _ := inject.HasJaegerAgent(pod.Spec.Containers); ok {
			if _, hasLabel := pod.Labels[inject.Label]; hasLabel {
				removeSidecarPod(ctx, pi.client, pod)
			}
		}

		if ok, _ := inject.HasJaegerAgent(deploy.Spec.Template.Spec.Containers); ok {
			if _, hasLabel := deploy.Labels[inject.Label]; hasLabel {
				removeSidecarDeployment(ctx, pi.client, deploy)
			}
		}
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

func deploymentByPod(
	ctx context.Context,
	c client.Client,
	pod *corev1.Pod,
	namespaceName string,
) (*appsv1.Deployment, error) {
	podRefList := pod.GetOwnerReferences()
	if len(podRefList) != 1 || podRefList[0].Kind != "ReplicaSet" {
		return nil, errors.New("missing single ReplicaSet as owner")
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
		return nil, errors.New("failed to get the available Pod ReplicaSet")
	}

	repRefList := replicaSet.GetOwnerReferences()
	if len(repRefList) != 1 || repRefList[0].Kind != "Deployment" {
		return nil, errors.New(
			fmt.Sprintf("could not determine deployment, number of owner: %d", len(repRefList)),
		)
	}

	deployName := repRefList[0].Name
	deployment := &appsv1.Deployment{}
	key = types.NamespacedName{Namespace: namespaceName, Name: deployName}
	if err := c.Get(ctx, key, deployment); err != nil {
		return nil, errors.New("failed to get the available Pod Deployment")
	}

	return deployment, nil

}

func removeSidecarDeployment(ctx context.Context, c client.Client, deploy *appsv1.Deployment) {
	jaegerInstance := deploy.Labels[inject.Label]
	log.WithFields(log.Fields{
		"deployment": deploy.Name,
		"namespace":  deploy.Namespace,
		"jaeger":     jaegerInstance,
	}).Info("Removing Jaeger Agent sidecar from Deployment")
	patch := client.MergeFrom(deploy.DeepCopy())
	inject.CleanSidecar(jaegerInstance, deploy)
	if err := c.Patch(ctx, deploy, patch); err != nil {
		log.WithFields(log.Fields{
			"deploymentName":      deploy.Name,
			"deploymentNamespace": deploy.Namespace,
		}).WithError(err).Error("error cleaning orphaned deployment")
	}
}

func removeSidecarPod(ctx context.Context, c client.Client, pod *corev1.Pod) {
	jaegerInstance := pod.Labels[inject.Label]
	log.WithFields(log.Fields{
		"pod":       pod.Name,
		"namespace": pod.Namespace,
		"jaeger":    jaegerInstance,
	}).Info("Removing Jaeger Agent sidecar from Pod")
	patch := client.MergeFrom(pod.DeepCopy())
	inject.CleanSidecarFromPod(jaegerInstance, pod)
	if err := c.Patch(ctx, pod, patch); err != nil {
		log.WithFields(log.Fields{
			"deploymentName":      pod.Name,
			"deploymentNamespace": pod.Namespace,
		}).WithError(err).Error("error cleaning orphaned deployment")
	}
}
