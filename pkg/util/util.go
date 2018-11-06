package util

import (
	"k8s.io/api/core/v1"
)

// RemoveDuplicatedVolumes defines to remove the duplicated item in slice if the Name of Volume is the same
func RemoveDuplicatedVolumes(volumes []v1.Volume) []v1.Volume {

	var result []v1.Volume

	for i := 0; i < len(volumes); i++ {
		// Scan slice for a previous element of the same value.
		exists := false
		for v := 0; v < i; v++ {
			if volumes[v].Name == volumes[i].Name {
				exists = true
				break
			}
		}
		// If no previous element exists, append this one.
		if !exists {
			result = append(result, volumes[i])
		}
	}
	// Return the new slice.
	return result
}

// RemoveDuplicatedVolumeMounts defines to remove the duplicated item in slice if the Name of Volume is the same
func RemoveDuplicatedVolumeMounts(volumeMounts []v1.VolumeMount) []v1.VolumeMount {

	var result []v1.VolumeMount

	for i := 0; i < len(volumeMounts); i++ {
		// Scan slice for a previous element of the same value.
		exists := false
		for v := 0; v < i; v++ {
			if volumeMounts[v].Name == volumeMounts[i].Name {
				exists = true
				break
			}
		}
		// If no previous element exists, append this one.
		if !exists {
			result = append(result, volumeMounts[i])
		}
	}
	// Return the new slice.
	return result
}
