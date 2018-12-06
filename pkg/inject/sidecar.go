package inject

import (
	"fmt"
	"strings"

	"github.com/sirupsen/logrus"
	appsv1 "k8s.io/api/apps/v1"
	"k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"

	"github.com/jaegertracing/jaeger-operator/pkg/apis/io/v1alpha1"
	"github.com/jaegertracing/jaeger-operator/pkg/deployment"
	"github.com/jaegertracing/jaeger-operator/pkg/service"
)

var (
	// Annotation is the annotation name to look for when deciding whether or not to inject
	Annotation = "inject-jaeger-agent"
)

const (
	envVarServiceName = "JAEGER_SERVICE_NAME"
	envVarPropagation = "JAEGER_PROPAGATION"
)

// Sidecar adds a new container to the deployment, connecting to the given jaeger instance
func Sidecar(obj runtime.Object, jaeger *v1alpha1.Jaeger) {
	deployment.NewAgent(jaeger) // we need some initialization from that, but we don't actually need the agent's instance here

	switch o := obj.(type) {
	case *appsv1.Deployment:
	case *appsv1.StatefulSet:
		if jaeger == nil || o.Annotations[Annotation] != jaeger.Name {
			logrus.Debugf("Skipping sidecar injection for instance %v", o.Name)
		} else {
			decorate(o)
			logrus.Debugf("Injecting sidecar for pod %v", o.Name)
			o.Spec.Template.Spec.Containers = append(o.Spec.Template.Spec.Containers, container(jaeger))
		}
	}
}

// Needed determines whether a pod needs to get a sidecar injected or not
func Needed(Name string, Annotations map[string]string, Containers []v1.Container) bool {
	if Annotations[Annotation] == "" {
		logrus.Debugf("Not needed, annotation not present for %v", Name)
		return false
	}

	// this pod is annotated, it should have a sidecar
	// but does it already have one?
	for _, container := range Containers {
		if container.Name == "jaeger-agent" { // we don't labels/annotations on containers, so, we rely on its name
			return false
		}
	}

	return true
}

// Select a suitable Jaeger from the JaegerList for the given Pod, or nil of none is suitable
func Select(Annotations map[string]string, availableJaegerPods *v1alpha1.JaegerList) *v1alpha1.Jaeger {
	jaegerName := Annotations[Annotation]
	if strings.ToLower(jaegerName) == "true" && len(availableJaegerPods.Items) == 1 {
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

func container(jaeger *v1alpha1.Jaeger) v1.Container {
	args := append(jaeger.Spec.Agent.Options.ToArgs(), fmt.Sprintf("--collector.host-port=%s:14267", service.GetNameForCollectorService(jaeger)))
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
	}
}

func decorate(obj runtime.Object) {
	switch o := obj.(type) {
	case *appsv1.Deployment:
	case *appsv1.StatefulSet:
		if app, found := o.Spec.Template.Labels["app"]; found {
			// Append the namespace to the app name. Using the DNS style "<app>.<namespace>""
			// which also matches with the style used in Istio.
			if len(o.Namespace) > 0 {
				app += "." + o.Namespace
			} else {
				app += ".default"
			}
			for i := 0; i < len(o.Spec.Template.Spec.Containers); i++ {
				if !hasEnv(envVarServiceName, o.Spec.Template.Spec.Containers[i].Env) {
					o.Spec.Template.Spec.Containers[i].Env = append(o.Spec.Template.Spec.Containers[i].Env, v1.EnvVar{
						Name:  envVarServiceName,
						Value: app,
					})
				}
				if !hasEnv(envVarPropagation, o.Spec.Template.Spec.Containers[i].Env) {
					o.Spec.Template.Spec.Containers[i].Env = append(o.Spec.Template.Spec.Containers[i].Env, v1.EnvVar{
						Name:  envVarPropagation,
						Value: "jaeger,b3",
					})
				}
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
