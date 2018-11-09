package util

import (
	"github.com/jaegertracing/jaeger-operator/pkg/apis/io/v1alpha1"
	"k8s.io/api/core/v1"
)

// removeDuplicatedVolumes returns a unique list of Volumes based on Volume names. Only the first item is kept.
func removeDuplicatedVolumes(volumes []v1.Volume) []v1.Volume {
	var results []v1.Volume
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
func removeDuplicatedVolumeMounts(volumeMounts []v1.VolumeMount) []v1.VolumeMount {
	var results []v1.VolumeMount
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
func Merge(commonSpecs []v1alpha1.JaegerCommonSpec) *v1alpha1.JaegerCommonSpec {
	annotations := make(map[string]string)
	var volumeMounts []v1.VolumeMount
	var volumes []v1.Volume

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
	}

	return &v1alpha1.JaegerCommonSpec{
		Annotations:  annotations,
		VolumeMounts: removeDuplicatedVolumeMounts(volumeMounts),
		Volumes:      removeDuplicatedVolumes(volumes),
	}
}
