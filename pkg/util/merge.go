// Copyright The Jaeger Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.
package util

import (
	corev1 "k8s.io/api/core/v1"

	v1 "github.com/jaegertracing/jaeger-operator/apis/jaegertracing/v2"
)

// RemoveDuplicatedVolumes returns a unique list of Volumes based on Volume names. Only the first item is kept.
func RemoveDuplicatedVolumes(volumes []corev1.Volume) []corev1.Volume {
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

// RemoveDuplicatedVolumeMounts returns a unique list based on the item names. Only the first item is kept.
func RemoveDuplicatedVolumeMounts(volumeMounts []corev1.VolumeMount) []corev1.VolumeMount {
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

// Merge returns a merged version of the list of JaegerCommonSpec instances with most specific first.
func Merge(commonSpecs ...v1.JaegerCommonSpec) v1.JaegerCommonSpec {
	annotations := make(map[string]string)
	labels := make(map[string]string)
	var volumeMounts []corev1.VolumeMount
	var volumes []corev1.Volume
	resources := &corev1.ResourceRequirements{}
	var affinity *corev1.Affinity
	var tolerations []corev1.Toleration
	var securityContext *corev1.PodSecurityContext
	var serviceAccount string

	for _, commonSpec := range commonSpecs {
		// Merge annotations
		for k, v := range commonSpec.Annotations {
			// Only use the value if key has not already been used
			if _, ok := annotations[k]; !ok {
				annotations[k] = v
			}
		}
		// Merge labels
		for k, v := range commonSpec.Labels {
			// Only use the value if key has not already been used
			if _, ok := labels[k]; !ok {
				labels[k] = v
			}
		}
		volumeMounts = append(volumeMounts, commonSpec.VolumeMounts...)
		volumes = append(volumes, commonSpec.Volumes...)

		// Merge resources
		MergeResources(resources, commonSpec.Resources)

		// Set the affinity based on the most specific definition available
		if affinity == nil {
			affinity = commonSpec.Affinity
		}

		tolerations = append(tolerations, commonSpec.Tolerations...)

		if securityContext == nil {
			securityContext = commonSpec.SecurityContext
		}

		if serviceAccount == "" {
			serviceAccount = commonSpec.ServiceAccount
		}
	}

	return v1.JaegerCommonSpec{
		Annotations:     annotations,
		Labels:          labels,
		VolumeMounts:    RemoveDuplicatedVolumeMounts(volumeMounts),
		Volumes:         RemoveDuplicatedVolumes(volumes),
		Resources:       *resources,
		Affinity:        affinity,
		Tolerations:     tolerations,
		SecurityContext: securityContext,
		ServiceAccount:  serviceAccount,
	}
}

// MergeResources returns a merged version of two resource requirements.
func MergeResources(resources *corev1.ResourceRequirements, res corev1.ResourceRequirements) {

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
