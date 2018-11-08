package util

import (
	"k8s.io/api/core/v1"
)

// RemoveDuplicatedVolumes returns a unique list of Volumes based on Volume names. Only the first item is kept.
func RemoveDuplicatedVolumes(volumes []v1.Volume) []v1.Volume {
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

// RemoveDuplicatedVolumeMounts returns a unique list based on the item names. Only the first item is kept.
func RemoveDuplicatedVolumeMounts(volumeMounts []v1.VolumeMount) []v1.VolumeMount {
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
