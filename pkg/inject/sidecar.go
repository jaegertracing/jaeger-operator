package inject

import (
	"fmt"
	"strings"

	log "github.com/sirupsen/logrus"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"

	"github.com/jaegertracing/jaeger-operator/pkg/apis/jaegertracing/v1"
	"github.com/jaegertracing/jaeger-operator/pkg/deployment"
	"github.com/jaegertracing/jaeger-operator/pkg/service"
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

// Sidecar adds a new container to the deployment, connecting to the given jaeger instance
func Sidecar(jaeger *v1.Jaeger, namespace string, annotations map[string]string, podSpecTemplate *corev1.PodTemplateSpec) {
	deployment.NewAgent(jaeger) // we need some initialization from that, but we don't actually need the agent's instance here
	// logFields := jaeger.Logger().WithField("deployment", dep.Name)

	if jaeger == nil || annotations[Annotation] != jaeger.Name {
		jaeger.Logger().WithField("Skipping sidecar injection for instance %v", name)
	} else {
		decorate(podSpecTemplate, namespace)
		jaeger.Logger().WithField("Injecting sidecar for pod %v", name)
		podSpecTemplate.Spec.Containers = append(podSpecTemplate.Spec.Containers, container(jaeger))
	}
}

// Needed determines whether a pod needs to get a sidecar injected or not
func Needed(namespace string, name string, annotations map[string]string, containers []corev1.Container) bool {
	if annotations[Annotation] == "" {
		log.WithFields(log.Fields{
			"namespace":  namespace,
			"deployment": name,
		}).Debug("annotation not present, not injecting")
	}

	// this pod is annotated, it should have a sidecar
	// but does it already have one?
	for _, container := range containers {
		if container.Name == "jaeger-agent" { // we don't labels/annotations on containers, so, we rely on its name
			return false
		}
	}

	return true
}

// Select a suitable Jaeger from the JaegerList for the given Pod, or nil of none is suitable
func Select(Annotations map[string]string, availableJaegerPods *v1.JaegerList) *v1.Jaeger {
	jaegerName := Annotations[Annotation]
	if strings.EqualFold(jaegerName, "true") && len(availableJaegerPods.Items) == 1 {
		// if there's only *one* jaeger within this namespace, then that's what
		// we'll use -- otherwise, we should just not inject, as it's not clear which
		// jaeger instance to use!
		// first, we make sure we normalize the name:
		jaeger := &availableJaegerPods.Items[0]
		Annotations[Annotation] = jaeger.Name
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

func container(jaeger *v1.Jaeger) corev1.Container {
	args := append(jaeger.Spec.Agent.Options.ToArgs(), fmt.Sprintf("--collector.host-port=%s.%s:14267", service.GetNameForCollectorService(jaeger), jaeger.Namespace))
	return corev1.Container{
		Image: jaeger.Spec.Agent.Image,
		Name:  "jaeger-agent",
		Args:  args,
		Ports: []corev1.ContainerPort{
			{
				ContainerPort: 5775,
				Name:          "zk-compact-trft",
			},
			{
				ContainerPort: 5778,
				Name:          "config-rest",
			},
			{
				ContainerPort: 6831,
				Name:          "jg-compact-trft",
			},
			{
				ContainerPort: 6832,
				Name:          "jg-binary-trft",
			},
		},
	}
}

func decorate(podTemplateSpec *corev1.PodTemplateSpec, namespace string) {
	if app, found := podTemplateSpec.Labels["app"]; found {
		// Append the namespace to the app name. Using the DNS style "<app>.<namespace>""
		// which also matches with the style used in Istio.
		if len(namespace) > 0 {
			app += "." + namespace
		} else {
			app += ".default"
		}
		for i := 0; i < len(podTemplateSpec.Spec.Containers); i++ {
			if !hasEnv(envVarServiceName, podTemplateSpec.Spec.Containers[i].Env) {
				podTemplateSpec.Spec.Containers[i].Env = append(podTemplateSpec.Spec.Containers[i].Env, corev1.EnvVar{
					Name:  envVarServiceName,
					Value: app,
				})
			}
			if !hasEnv(envVarPropagation, podTemplateSpec.Spec.Containers[i].Env) {
				podTemplateSpec.Spec.Containers[i].Env = append(podTemplateSpec.Spec.Containers[i].Env, corev1.EnvVar{
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
