package inject

import (
	appsv1 "k8s.io/api/apps/v1"

	"github.com/jaegertracing/jaeger-operator/pkg/autodetect"
	"github.com/jaegertracing/jaeger-operator/pkg/config/ca"
)

var (
	// AnnotationRev is the annotation name to look for when deciding whether or not to inject
	AnnotationRev = "sidecar.jaegertracing.io/revision"
	// Annotation is the annotation name to look for when deciding whether or not to inject
	Annotation = "sidecar.jaegertracing.io/inject"
	// Label is the label name the operator put on injected deployments.
	Label = "sidecar.jaegertracing.io/injected"
	// AnnotationLegacy holds the annotation name we had in the past, which we keep for backwards compatibility
	AnnotationLegacy = "inject-jaeger-agent"
	// PrometheusDefaultAnnotations is a map containing annotations for prometheus to be inserted at sidecar in case it doesn't have any
	PrometheusDefaultAnnotations = map[string]string{
		"prometheus.io/scrape": "true",
		"prometheus.io/port":   "14271",
	}
)

// CleanSidecar of  deployments  associated with the jaeger instance.
func CleanSidecar(instanceName string, deployment *appsv1.Deployment) {
	delete(deployment.Labels, Label)
	for c := 0; c < len(deployment.Spec.Template.Spec.Containers); c++ {
		if deployment.Spec.Template.Spec.Containers[c].Name == "jaeger-agent" {
			// delete jaeger-agent container
			deployment.Spec.Template.Spec.Containers = append(deployment.Spec.Template.Spec.Containers[:c], deployment.Spec.Template.Spec.Containers[c+1:]...)
			break
		}
	}
	if autodetect.OperatorConfiguration.GetPlatform() == autodetect.OpenShiftPlatform {
		names := map[string]bool{
			ca.TrustedCANameFromString(instanceName): true,
			ca.ServiceCANameFromString(instanceName): true,
		}
		// Remove the managed volumes, if present
		for v := 0; v < len(deployment.Spec.Template.Spec.Volumes); v++ {
			if _, ok := names[deployment.Spec.Template.Spec.Volumes[v].Name]; ok {
				// delete managed volume
				deployment.Spec.Template.Spec.Volumes = append(deployment.Spec.Template.Spec.Volumes[:v], deployment.Spec.Template.Spec.Volumes[v+1:]...)
				v--
			}
		}
	}
}
