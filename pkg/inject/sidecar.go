package inject

import (
	"fmt"
	"strings"

	"github.com/operator-framework/operator-sdk/pkg/sdk"
	"github.com/sirupsen/logrus"
	appsv1 "k8s.io/api/apps/v1"
	"k8s.io/api/core/v1"

	"github.com/jaegertracing/jaeger-operator/pkg/apis/io/v1alpha1"
	"github.com/jaegertracing/jaeger-operator/pkg/deployment"
	"github.com/jaegertracing/jaeger-operator/pkg/service"
)

var (
	// Annotation is the annotation name to look for when deciding whether or not to inject
	Annotation = "inject-jaeger-agent"
)

// injectSidecarInDeployment adds a new container to the deployment, connecting to the given jaeger instance
func injectSidecarInDeployment(dep *appsv1.Deployment, jaeger *v1alpha1.Jaeger) {
	deployment.NewAgent(jaeger) // we need some initialization from that, but we don't actually need the agent's instance here

	if jaeger == nil || dep.Annotations[Annotation] != jaeger.Name {
		logrus.Debugf("Skipping sidecar injection for deployment %v", dep.Name)
	} else {
		logrus.Debugf("Injecting sidecar for pod %v", dep.Name)
		dep.Spec.Template.Spec.Containers = append(dep.Spec.Template.Spec.Containers, container(jaeger))
	}
}

// injectSidecarInStatefulset adds a new container to the deployment, connecting to the given jaeger instance
func injectSidecarInStatefulset(sset *appsv1.StatefulSet, jaeger *v1alpha1.Jaeger) {
	deployment.NewAgent(jaeger) // we need some initialization from that, but we don't actually need the agent's instance here

	if jaeger == nil || sset.Annotations[Annotation] != jaeger.Name {
		logrus.Debugf("Skipping sidecar injection for deployment %v", sset.Name)
	} else {
		logrus.Debugf("Injecting sidecar for pod %v", sset.Name)
		sset.Spec.Template.Spec.Containers = append(sset.Spec.Template.Spec.Containers, container(jaeger))
	}
}

// Sidecar adds a new container to the object, connecting to the given jaeger instance
func Sidecar(obj sdk.Object, jaeger *v1alpha1.Jaeger) {
	switch o := obj.(type) {
	case *appsv1.Deployment:
		injectSidecarInDeployment(o, jaeger)
	case *appsv1.StatefulSet:
		injectSidecarInStatefulset(o, jaeger)
	}
}

// isNeededInDeployment determines whether a pod needs to get a sidecar injected or not
func isNeededInDeployment(dep *appsv1.Deployment) bool {
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

// isNeededInStatefulset determines whether a pod needs to get a sidecar injected or not
func isNeededInStatefulset(sset *appsv1.StatefulSet) bool {
	if sset.Annotations[Annotation] == "" {
		logrus.Debugf("Not needed, annotation not present for %v", sset.Name)
		return false
	}

	// this pod is annotated, it should have a sidecar
	// but does it already have one?
	for _, container := range sset.Spec.Template.Spec.Containers {
		if container.Name == "jaeger-agent" { // we don't labels/annotations on containers, so, we rely on its name
			return false
		}
	}

	return true
}

// Needed determines whether a pod needs to get a sidecar injected or not
func Needed(obj sdk.Object) bool {
	switch o := obj.(type) {
	case *appsv1.Deployment:
		return isNeededInDeployment(o)
	case *appsv1.StatefulSet:
		return isNeededInStatefulset(o)
	}
	return false
}

// Select a suitable Jaeger from the JaegerList for the given Pod, or nil of none is suitable
func SelectForDeployment(target *appsv1.Deployment, availableJaegerPods *v1alpha1.JaegerList) *v1alpha1.Jaeger {
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

// Select a suitable Jaeger from the JaegerList for the given Pod, or nil of none is suitable
func SelectForStatefulset(target *appsv1.StatefulSet, availableJaegerPods *v1alpha1.JaegerList) *v1alpha1.Jaeger {
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

// Select a suitable Jaeger from the JaegerList for the given Pod, or nil of none is suitable
func Select(obj sdk.Object, availableJaegerPods *v1alpha1.JaegerList) *v1alpha1.Jaeger {
	switch o := obj.(type) {
	case *appsv1.Deployment:
		return SelectForDeployment(o, availableJaegerPods)
	case *appsv1.StatefulSet:
		return SelectForStatefulset(o, availableJaegerPods)
	}
	return nil
}

// Return a container for sidecar injection
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
