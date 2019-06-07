package inject

import (
	"context"
	"encoding/json"
	"fmt"
	"sort"
	"strings"

	log "github.com/sirupsen/logrus"
	admission "k8s.io/api/admission/v1beta1"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"

	v1 "github.com/jaegertracing/jaeger-operator/pkg/apis/jaegertracing/v1"
	"github.com/jaegertracing/jaeger-operator/pkg/deployment"
	"github.com/jaegertracing/jaeger-operator/pkg/service"
	"github.com/jaegertracing/jaeger-operator/pkg/util"
)

var (
	// Annotation is the annotation name to look for when deciding whether or not to inject
	Annotation = "sidecar.jaegertracing.io/inject"

	// AnnotationLegacy holds the annotation name we had in the past, which we keep for backwards compatibility
	AnnotationLegacy = "inject-jaeger-agent"
)

const (
	envVarServiceName = "JAEGER_SERVICE_NAME"
	envVarPropagation = "JAEGER_PROPAGATION"
)

// Sidecar adds a new container to the pod, connecting to the given jaeger instance
func Sidecar(jaeger *v1.Jaeger, pod corev1.Pod) corev1.Pod {
	deployment.NewAgent(jaeger) // we need some initialization from that, but we don't actually need the agent's instance here
	logFields := jaeger.Logger().WithField("pod", pod.Name)

	if jaeger == nil || (pod.Annotations[Annotation] != jaeger.Name && pod.Annotations[AnnotationLegacy] != jaeger.Name) {
		logFields.Debug("skipping sidecar injection")
	} else {
		decorate(pod)
		logFields.Debug("injecting sidecar")
		pod.Spec = SidecarIntoPodSpec(jaeger, pod.Spec)
	}

	return pod
}

// Process an admission review, returning an AdmissionResponse with a patch object in case a sidecar is needed
func Process(ar *admission.AdmissionReview, c client.Client) (*admission.AdmissionResponse, error) {
	podGvk := metav1.GroupVersionKind{Group: "", Version: "v1", Kind: "Pod"}
	if podGvk == ar.Request.Kind {
		return processPod(c, ar)
	}

	depGvk := metav1.GroupVersionKind{Group: "apps", Version: "v1", Kind: "Deployment"}
	if depGvk == ar.Request.Kind {
		return processDeployment(ar)
	}

	// unknown type, just ignore it
	log.WithField("gvk", ar.Request.Kind).Info("unknown group/version/kind, ignoring")
	return &admission.AdmissionResponse{Allowed: true}, nil

}

func processPod(c client.Client, ar *admission.AdmissionReview) (*admission.AdmissionResponse, error) {
	var pod corev1.Pod
	if err := json.Unmarshal(ar.Request.Object.Raw, &pod); err != nil {
		log.WithError(err).Warn("couldn't process the admission review's raw object")
		return &admission.AdmissionResponse{
			Result: &metav1.Status{
				Message: err.Error(),
			},
		}, err
	}

	if Needed(pod) {
		jaegerInstances := &v1.JaegerList{}
		opts := &client.ListOptions{}
		err := c.List(context.Background(), opts, jaegerInstances)
		if err != nil {
			log.WithError(err).Error("failed to get the available Jaeger instances")
			return &admission.AdmissionResponse{
				Result: &metav1.Status{
					Message: err.Error(),
				},
			}, err
		}

		jaeger := Select(pod, jaegerInstances)
		if jaeger != nil {
			// prepare the patch operation to return
			patch := map[string]interface{}{
				"op":    "add",
				"path":  "/spec/containers/-",
				"value": container(jaeger),
			}

			log.WithField("patch", patch).Debug("returning patch for pod")
			return getResponseForPatch(patch)
		}

		log.WithFields(log.Fields{
			"pod":       pod.Name,
			"namespace": pod.Namespace,
		}).Info("no suitable Jaeger instances found to inject a sidecar")
	}

	// sidecar not needed, skip
	return &admission.AdmissionResponse{Allowed: true}, nil
}

func processDeployment(ar *admission.AdmissionReview) (*admission.AdmissionResponse, error) {
	var dep appsv1.Deployment
	if err := json.Unmarshal(ar.Request.Object.Raw, &dep); err != nil {
		log.WithError(err).Warn("couldn't process the admission review's raw object")
		return &admission.AdmissionResponse{
			Result: &metav1.Status{
				Message: err.Error(),
			},
		}, err
	}

	// given that a similar event is triggered for the pod inside the deployment,
	// we just need to place a new annotation in the pod, in case
	// the deployment has one of the known annotations
	if _, ok := getAnnotation(dep.Spec.Template.Annotations); ok {
		// the pod has its own annotation, just skip
		log.WithFields(log.Fields{
			"deployment": dep.Name,
			"namespace":  dep.Namespace,
		}).Debug("pod has its own annotation, skipping processing of deployment")

		return &admission.AdmissionResponse{Allowed: true}, nil
	}

	if val, ok := getAnnotation(dep.Annotations); ok {
		// deployment has the annotation, copy it over to the pod
		patch := map[string]interface{}{
			"op":    "add",
			"path":  "/spec/template/metadata/annotations",
			"value": map[string]string{Annotation: val},
		}

		log.WithFields(log.Fields{
			"deployment": dep.Name,
			"namespace":  dep.Namespace,
			"patch":      patch,
		}).Debug("returning a patch for the deployment")

		return getResponseForPatch(patch)
	}

	// we stop processing this review here, as there's nothing for us to do at the deployment level
	return &admission.AdmissionResponse{Allowed: true}, nil
}

// Needed determines whether a pod needs to get a sidecar injected or not
func Needed(pod corev1.Pod) bool {
	if pod.Annotations[Annotation] == "" {
		log.WithFields(log.Fields{
			"namespace":  pod.Namespace,
			"deployment": pod.Name,
		}).Debug("annotation not present, not injecting")
		return false
	}

	// this pod is annotated, it should have a sidecar
	// but does it already have one?
	for _, container := range pod.Spec.Containers {
		if container.Name == "jaeger-agent" { // we don't labels/annotations on containers, so, we rely on its name
			return false
		}
	}

	return true
}

// Select a suitable Jaeger from the JaegerList for the given Pod, or nil of none is suitable
func Select(target corev1.Pod, availableJaegerPods *v1.JaegerList) *v1.Jaeger {
	jaegerName := target.Annotations[Annotation]
	if strings.EqualFold(jaegerName, "true") && len(availableJaegerPods.Items) == 1 {
		// if there's only *one* jaeger within this namespace, then that's what
		// we'll use -- otherwise, we should just not inject, as it's not clear which
		// jaeger instance to use!
		// first, we make sure we normalize the name:
		jaeger := &availableJaegerPods.Items[0]
		target.Annotations[Annotation] = jaeger.Name
		return jaeger
	}

	for _, p := range availableJaegerPods.Items {
		if p.Name == jaegerName {
			// matched the name!
			return &p
		}
	}
	return nil
}

// SidecarIntoPodSpec provides a Container with the Jaeger Agent to be used as a sidecar
func SidecarIntoPodSpec(jaeger *v1.Jaeger, pod corev1.PodSpec) corev1.PodSpec {
	pod.Containers = append(pod.Containers, container(jaeger))
	return pod
}

func container(jaeger *v1.Jaeger) corev1.Container {
	args := append(jaeger.Spec.Agent.Options.ToArgs())

	if len(util.FindItem("--reporter.type=", args)) == 0 {
		args = append(args, "--reporter.type=grpc")

		// we only add the grpc host if we are adding the reporter type and there's no explicit value yet
		if len(util.FindItem("--reporter.grpc.host-port=", args)) == 0 {
			args = append(args, fmt.Sprintf("--reporter.grpc.host-port=dns:///%s.%s:14250", service.GetNameForHeadlessCollectorService(jaeger), jaeger.Namespace))
		}
	}

	zkCompactTrft := util.GetPort("--processor.zipkin-compact.server-host-port=", args, 5775)
	configRest := util.GetPort("--http-server.host-port=", args, 5778)
	jgCompactTrft := util.GetPort("--processor.jaeger-compact.server-host-port=", args, 6831)
	jgBinaryTrft := util.GetPort("--processor.jaeger-binary.server-host-port=", args, 6832)

	commonSpec := util.Merge([]v1.JaegerCommonSpec{jaeger.Spec.Agent.JaegerCommonSpec, jaeger.Spec.JaegerCommonSpec})

	// ensure we have a consistent order of the arguments
	// see https://github.com/jaegertracing/jaeger-operator/issues/334
	sort.Strings(args)

	return corev1.Container{
		Image: jaeger.Spec.Agent.Image,
		Name:  "jaeger-agent",
		Args:  args,
		Ports: []corev1.ContainerPort{
			{
				ContainerPort: zkCompactTrft,
				Name:          "zk-compact-trft",
			},
			{
				ContainerPort: configRest,
				Name:          "config-rest",
			},
			{
				ContainerPort: jgCompactTrft,
				Name:          "jg-compact-trft",
			},
			{
				ContainerPort: jgBinaryTrft,
				Name:          "jg-binary-trft",
			},
		},
		Resources: commonSpec.Resources,
	}
}

func decorate(pod corev1.Pod) {
	app, found := pod.Labels["app.kubernetes.io/instance"]
	if !found {
		app, found = pod.Labels["app.kubernetes.io/name"]
	}
	if !found {
		app, found = pod.Labels["app"]
	}
	if found {
		// Append the namespace to the app name. Using the DNS style "<app>.<namespace>""
		// which also matches with the style used in Istio.
		if len(pod.Namespace) > 0 {
			app += "." + pod.Namespace
		} else {
			app += ".default"
		}
		for i := 0; i < len(pod.Spec.Containers); i++ {
			if !hasEnv(envVarServiceName, pod.Spec.Containers[i].Env) {
				pod.Spec.Containers[i].Env = append(pod.Spec.Containers[i].Env, corev1.EnvVar{
					Name:  envVarServiceName,
					Value: app,
				})
			}
			if !hasEnv(envVarPropagation, pod.Spec.Containers[i].Env) {
				pod.Spec.Containers[i].Env = append(pod.Spec.Containers[i].Env, corev1.EnvVar{
					Name:  envVarPropagation,
					Value: "jaeger,b3",
				})
			}
		}
	}
}

func hasEnv(name string, vars []corev1.EnvVar) bool {
	for i := 0; i < len(vars); i++ {
		if vars[i].Name == name {
			return true
		}
	}
	return false
}

func getAnnotation(annotations map[string]string) (string, bool) {
	if val, ok := annotations[Annotation]; ok {
		return val, ok
	}

	val, ok := annotations[AnnotationLegacy]
	return val, ok
}

func getResponseForPatch(patch map[string]interface{}) (*admission.AdmissionResponse, error) {
	var patches [1]map[string]interface{} // json patches are always arrays
	patches[0] = patch

	jsonPatch, err := json.Marshal(patches)
	if err != nil {
		// is it even possible to fail here?
		return &admission.AdmissionResponse{
			Result: &metav1.Status{
				Message: err.Error(),
			},
		}, err
	}

	patchType := admission.PatchTypeJSONPatch
	return &admission.AdmissionResponse{
		Allowed:   true,
		Patch:     jsonPatch,
		PatchType: &patchType,
	}, nil

}
