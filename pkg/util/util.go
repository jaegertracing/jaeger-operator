package util

import (
	corev1 "k8s.io/api/core/v1"

	"github.com/jaegertracing/jaeger-operator/pkg/apis/jaegertracing/v1"
)

// removeDuplicatedVolumes returns a unique list of Volumes based on Volume names. Only the first item is kept.
func removeDuplicatedVolumes(volumes []corev1.Volume) []corev1.Volume {
	var results []corev1.Volume
	existing := map[string]bool{}

	for _, volume := range volumes {
		if existing[volume.Name] {
			continue
		}
		results = append(results, volume)
		existing[volume.Name] = true
	}
	// Return the new slice.
	return results
}

// removeDuplicatedVolumeMounts returns a unique list based on the item names. Only the first item is kept.
func removeDuplicatedVolumeMounts(volumeMounts []corev1.VolumeMount) []corev1.VolumeMount {
	var results []corev1.VolumeMount
	existing := map[string]bool{}

	for _, volumeMount := range volumeMounts {
		if existing[volumeMount.Name] {
			continue
		}
		results = append(results, volumeMount)
		existing[volumeMount.Name] = true
	}
	// Return the new slice.
	return results
}

// Merge returns a merged version of the list of JaegerCommonSpec instances with most specific first
func Merge(commonSpecs []v1.JaegerCommonSpec) *v1.JaegerCommonSpec {
	annotations := make(map[string]string)
	var volumeMounts []corev1.VolumeMount
	var volumes []corev1.Volume
	resources := &corev1.ResourceRequirements{}

	for _, commonSpec := range commonSpecs {
		// Merge annotations
		for k, v := range commonSpec.Annotations {
			// Only use the value if key has not already been used
			if _, ok := annotations[k]; !ok {
				annotations[k] = v
			}
		}
		volumeMounts = append(volumeMounts, commonSpec.VolumeMounts...)
		volumes = append(volumes, commonSpec.Volumes...)

		// Merge resources
		mergeResources(resources, commonSpec.Resources)
	}

	return &v1.JaegerCommonSpec{
		Annotations:  annotations,
		VolumeMounts: removeDuplicatedVolumeMounts(volumeMounts),
		Volumes:      removeDuplicatedVolumes(volumes),
		Resources:    *resources,
	}
}

func mergeResources(resources *corev1.ResourceRequirements, res corev1.ResourceRequirements) {

	for k, v := range res.Limits {
		if _, ok := resources.Limits[k]; !ok {
			if resources.Limits == nil {
				resources.Limits = make(corev1.ResourceList)
			}
			resources.Limits[k] = v
		}
	}

	for k, v := range res.Requests {
		if _, ok := resources.Requests[k]; !ok {
			if resources.Requests == nil {
				resources.Requests = make(corev1.ResourceList)
			}
			resources.Requests[k] = v
		}
	}
}
