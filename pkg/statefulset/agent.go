package statefulset

import (
	"strings"

	"github.com/sirupsen/logrus"
	appsv1 "k8s.io/api/apps/v1"
	"k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/jaegertracing/jaeger-operator/pkg/apis/io/v1alpha1"
	"github.com/jaegertracing/jaeger-operator/pkg/deployment"
	"github.com/jaegertracing/jaeger-operator/pkg/inject"
)

var (
	// Annotation is the annotation name to look for when deciding whether or not to inject
	Annotation = "inject-jaeger-agent"
)

// Sidecar adds a new container to the StatefulSet, connecting to the given jaeger instance
func Sidecar(statefulset *appsv1.StatefulSet, jaeger *v1alpha1.Jaeger) {
	deployment.NewAgent(jaeger) // we need some initialization from that, but we don't actually need the agent's instance here

	if jaeger == nil || statefulset.Annotations[Annotation] != jaeger.Name {
		logrus.Debugf("Skipping sidecar injection for statefulest: %v", statefulset.Name)
	} else {
		logrus.Debugf("Injecting sidecar in statefulset %v", statefulset.Name)
		statefulset.Spec.Template.Spec.Containers = append(statefulset.Spec.Template.Spec.Containers, inject.Container(jaeger))
	}
}

// Needed determines whether a pod needs to get a sidecar injected or not
func Needed(sset *appsv1.StatefulSet) bool {
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

// Select a suitable Jaeger from the JaegerList for the given Pod, or nil of none is suitable
func Select(target *appsv1.StatefulSet, availableJaegerPods *v1alpha1.JaegerList) *v1alpha1.Jaeger {
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
