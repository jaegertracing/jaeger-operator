package inject

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/sirupsen/logrus"
	appsv1 "k8s.io/api/apps/v1"
	"k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"

	"github.com/jaegertracing/jaeger-operator/pkg/apis/io/v1alpha1"
	"github.com/jaegertracing/jaeger-operator/pkg/deployment"
	"github.com/jaegertracing/jaeger-operator/pkg/service"
)

var (
	// Annotation is the annotation name to look for when deciding whether or not to inject
	Annotation = "inject-jaeger-agent"
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
func Sidecar(dep *appsv1.Deployment, jaeger *v1alpha1.Jaeger) {
	deployment.NewAgent(jaeger) // we need some initialization from that, but we don't actually need the agent's instance here

	if jaeger == nil || dep.Annotations[Annotation] != jaeger.Name {
		logrus.Debugf("Skipping sidecar injection for deployment %v", dep.Name)
	} else {
		decorate(dep)
		logrus.Debugf("Injecting sidecar for pod %v", dep.Name)
		dep.Spec.Template.Spec.Containers = append(dep.Spec.Template.Spec.Containers, container(dep, jaeger))
	}
}

// Needed determines whether a pod needs to get a sidecar injected or not
func Needed(dep *appsv1.Deployment) bool {
	if dep.Annotations[Annotation] == "" {
		logrus.Debugf("Not needed, annotation not present for %v", dep.Name)
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
func Select(target *appsv1.Deployment, availableJaegerPods *v1alpha1.JaegerList) *v1alpha1.Jaeger {
	jaegerName := target.Annotations[Annotation]
	if strings.ToLower(jaegerName) == "true" && len(availableJaegerPods.Items) == 1 {
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

func container(dep *appsv1.Deployment, jaeger *v1alpha1.Jaeger) v1.Container {
	args := append(jaeger.Spec.Agent.Options.ToArgs(), fmt.Sprintf("--collector.host-port=%s:14267", service.GetNameForCollectorService(jaeger)))

	// Checking annotations for CPU/Memory limits
	limitCPU := "500"
	limitMem := "128"
	if dep.Annotations[LimitCPU] != "" {
		limitCPU = dep.Annotations[LimitCPU]
	}
	if dep.Annotations[LimitMem] != "" {
		limitMem = dep.Annotations[LimitMem]
	}

	CPULimit, _ := strconv.Atoi(limitCPU)
	MemLimit, _ := strconv.Atoi(limitMem)

	return v1.Container{
		Image: jaeger.Spec.Agent.Image,
		Name:  "jaeger-agent",
		Args:  args,
		Ports: []v1.ContainerPort{
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
		Resources: v1.ResourceRequirements{
			Limits: v1.ResourceList{
				v1.ResourceLimitsCPU:    *resource.NewQuantity(int64(CPULimit), resource.BinarySI),
				v1.ResourceLimitsMemory: *resource.NewQuantity(int64(MemLimit), resource.DecimalSI),
			},
			Requests: v1.ResourceList{
				v1.ResourceRequestsCPU:    *resource.NewQuantity(int64(CPULimit), resource.BinarySI),
				v1.ResourceRequestsMemory: *resource.NewQuantity(int64(MemLimit), resource.DecimalSI),
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
				dep.Spec.Template.Spec.Containers[i].Env = append(dep.Spec.Template.Spec.Containers[i].Env, v1.EnvVar{
					Name:  envVarServiceName,
					Value: app,
				})
			}
			if !hasEnv(envVarPropagation, dep.Spec.Template.Spec.Containers[i].Env) {
				dep.Spec.Template.Spec.Containers[i].Env = append(dep.Spec.Template.Spec.Containers[i].Env, v1.EnvVar{
					Name:  envVarPropagation,
					Value: "jaeger,b3",
				})
			}
		}
	}
}

func hasEnv(name string, vars []v1.EnvVar) bool {
	for i := 0; i < len(vars); i++ {
		if vars[i].Name == name {
			return true
		}
	}
	return false
}
