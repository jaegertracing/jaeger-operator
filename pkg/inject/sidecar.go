package inject

import (
	"fmt"
	"sort"
	"strings"

	log "github.com/sirupsen/logrus"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"

	"github.com/jaegertracing/jaeger-operator/pkg/apis/jaegertracing/v1"
	"github.com/jaegertracing/jaeger-operator/pkg/deployment"
	"github.com/jaegertracing/jaeger-operator/pkg/service"
	"github.com/jaegertracing/jaeger-operator/pkg/util"
)

var (
	// Annotation is the annotation name to look for when deciding whether or not to inject
	Annotation = "sidecar.jaegertracing.io/inject"

	// AnnotationLegacy holds the annotation name we had in the past, which we keep for backwards compatibility
	AnnotationLegacy = "inject-jaeger-agent"

	// LimitCPU is the annotation name for cpu resource limits for a jaeger agent sidecar
	LimitCPU = "jaeger-agent-max-cpu"

	// LimitMem is the annotation name for memory resource limits for a jaeger agent sidecar
	LimitMem = "jaeger-agent-max-memory"
)

const (
	envVarServiceName = "JAEGER_SERVICE_NAME"
	envVarPropagation = "JAEGER_PROPAGATION"
)

// Sidecar adds a new container to the deployment, connecting to the given jaeger instance
func Sidecar(jaeger *v1.Jaeger, dep *appsv1.Deployment) *appsv1.Deployment {
	deployment.NewAgent(jaeger) // we need some initialization from that, but we don't actually need the agent's instance here
	logFields := jaeger.Logger().WithField("deployment", dep.Name)

	if jaeger == nil || (dep.Annotations[Annotation] != jaeger.Name && dep.Annotations[AnnotationLegacy] != jaeger.Name) {
		logFields.Debug("skipping sidecar injection")
	} else {
		decorate(dep)
		logFields.Debug("injecting sidecar")
		dep.Spec.Template.Spec.Containers = append(dep.Spec.Template.Spec.Containers, container(jaeger, dep))
	}

	return dep
}

// Needed determines whether a pod needs to get a sidecar injected or not
func Needed(dep *appsv1.Deployment) bool {
	if dep.Annotations[Annotation] == "" {
		log.WithFields(log.Fields{
			"namespace":  dep.Namespace,
			"deployment": dep.Name,
		}).Debug("annotation not present, not injecting")
		return false
	}

	// this pod is annotated, it should have a sidecar
	// but does it already have one?
	for _, container := range dep.Spec.Template.Spec.Containers {
		if container.Name == "jaeger-agent" { // we don't labels/annotations on containers, so, we rely on its name
			return false
		}
	}

	return true
}

// Select a suitable Jaeger from the JaegerList for the given Pod, or nil of none is suitable
func Select(target *appsv1.Deployment, availableJaegerPods *v1.JaegerList) *v1.Jaeger {
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

func container(jaeger *v1.Jaeger, dep *appsv1.Deployment) corev1.Container {
	args := append(jaeger.Spec.Agent.Options.ToArgs())

	if len(util.FindItem("--reporter.type=", args)) == 0 {
		args = append(args, "--reporter.type=grpc")

		// we only add the grpc host if we are adding the reporter type and there's no explicit value yet
		if len(util.FindItem("--reporter.grpc.host-port=", args)) == 0 {
			args = append(args, fmt.Sprintf("--reporter.grpc.host-port=dns:///%s.%s:14250", service.GetNameForHeadlessCollectorService(jaeger), jaeger.Namespace))
		}
	}

	// ensure we have a consistent order of the arguments
	// see https://github.com/jaegertracing/jaeger-operator/issues/334
	sort.Strings(args)

	// Checking annotations for CPU/Memory limits
	limitCPU := "500m"
	limitMem := "128Mi"

	CPULimitDefault, _ := resource.ParseQuantity(limitCPU)
	MemLimitDefault, _ := resource.ParseQuantity(limitMem)

	if dep.Annotations[LimitCPU] != "" {
		limitCPU = dep.Annotations[LimitCPU]
	}
	if dep.Annotations[LimitMem] != "" {
		limitMem = dep.Annotations[LimitMem]
	}

	CPULimit, err := resource.ParseQuantity(limitCPU)
	if err != nil {
		jaeger.Logger().Debugf("Could not parse quantity for CPU limits: %v, using defaults.", limitCPU)
		CPULimit = CPULimitDefault
	}
	MemLimit, err := resource.ParseQuantity(limitMem)
	if err != nil {
		jaeger.Logger().Debugf("Could not parse quantity for Memory limits: %v, using defaults.", limitMem)
		MemLimit = MemLimitDefault
	}

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
		Resources: corev1.ResourceRequirements{
			Limits: corev1.ResourceList{
				corev1.ResourceLimitsCPU:    CPULimit,
				corev1.ResourceLimitsMemory: MemLimit,
			},
			Requests: corev1.ResourceList{
				corev1.ResourceRequestsCPU:    CPULimit,
				corev1.ResourceRequestsMemory: MemLimit,
			},
		},
	}
}

func decorate(dep *appsv1.Deployment) {
	if app, found := dep.Spec.Template.Labels["app"]; found {
		// Append the namespace to the app name. Using the DNS style "<app>.<namespace>""
		// which also matches with the style used in Istio.
		if len(dep.Namespace) > 0 {
			app += "." + dep.Namespace
		} else {
			app += ".default"
		}
		for i := 0; i < len(dep.Spec.Template.Spec.Containers); i++ {
			if !hasEnv(envVarServiceName, dep.Spec.Template.Spec.Containers[i].Env) {
				dep.Spec.Template.Spec.Containers[i].Env = append(dep.Spec.Template.Spec.Containers[i].Env, corev1.EnvVar{
					Name:  envVarServiceName,
					Value: app,
				})
			}
			if !hasEnv(envVarPropagation, dep.Spec.Template.Spec.Containers[i].Env) {
				dep.Spec.Template.Spec.Containers[i].Env = append(dep.Spec.Template.Spec.Containers[i].Env, corev1.EnvVar{
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
