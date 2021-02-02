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
	"testing"

	"github.com/stretchr/testify/assert"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"

	v2 "github.com/jaegertracing/jaeger-operator/apis/jaegertracing/v2"
)

func TestRemoveDuplicatedVolumes(t *testing.T) {
	volumes := []corev1.Volume{{
		Name:         "volume1",
		VolumeSource: corev1.VolumeSource{HostPath: &corev1.HostPathVolumeSource{Path: "/data1"}},
	}, {
		Name:         "volume2",
		VolumeSource: corev1.VolumeSource{HostPath: &corev1.HostPathVolumeSource{Path: "/data2"}},
	}, {
		Name:         "volume1",
		VolumeSource: corev1.VolumeSource{HostPath: &corev1.HostPathVolumeSource{Path: "/data3"}},
	}}

	assert.Len(t, RemoveDuplicatedVolumes(volumes), 2)
	assert.Equal(t, "volume1", volumes[0].Name)
	assert.Equal(t, "/data1", volumes[0].VolumeSource.HostPath.Path)
	assert.Equal(t, "volume2", volumes[1].Name)
}

func TestRemoveDuplicatedVolumeMounts(t *testing.T) {
	volumeMounts := []corev1.VolumeMount{{
		Name:     "data1",
		ReadOnly: false,
	}, {
		Name:     "data2",
		ReadOnly: false,
	}, {
		Name:     "data1",
		ReadOnly: true,
	}}

	assert.Len(t, RemoveDuplicatedVolumeMounts(volumeMounts), 2)
	assert.Equal(t, "data1", volumeMounts[0].Name)
	assert.Equal(t, false, volumeMounts[0].ReadOnly)
	assert.Equal(t, "data2", volumeMounts[1].Name)
}

func TestMergeAnnotations(t *testing.T) {
	generalSpec := v2.JaegerCommonSpec{
		Annotations: map[string]string{
			"name":  "operator",
			"hello": "jaeger",
		},
	}
	specificSpec := v2.JaegerCommonSpec{
		Annotations: map[string]string{
			"hello":                "world", // Override general annotation
			"prometheus.io/scrape": "false",
		},
	}

	merged := Merge(specificSpec, generalSpec)

	assert.Equal(t, "operator", merged.Annotations["name"])
	assert.Equal(t, "world", merged.Annotations["hello"])
	assert.Equal(t, "false", merged.Annotations["prometheus.io/scrape"])
}

func TestMergeLabels(t *testing.T) {
	generalSpec := v2.JaegerCommonSpec{
		Labels: map[string]string{
			"name":  "operator",
			"hello": "jaeger",
		},
	}
	specificSpec := v2.JaegerCommonSpec{
		Labels: map[string]string{
			"hello":   "world", // Override general annotation
			"another": "false",
		},
	}

	merged := Merge(specificSpec, generalSpec)

	assert.Equal(t, "operator", merged.Labels["name"])
	assert.Equal(t, "world", merged.Labels["hello"])
	assert.Equal(t, "false", merged.Labels["another"])
}

func TestMergeMountVolumes(t *testing.T) {
	generalSpec := v2.JaegerCommonSpec{
		VolumeMounts: []corev1.VolumeMount{{
			Name:     "data1",
			ReadOnly: true,
		}},
	}
	specificSpec := v2.JaegerCommonSpec{
		VolumeMounts: []corev1.VolumeMount{{
			Name:     "data1",
			ReadOnly: false,
		}, {
			Name:     "data2",
			ReadOnly: false,
		}},
	}

	merged := Merge(specificSpec, generalSpec)

	assert.Equal(t, "data1", merged.VolumeMounts[0].Name)
	assert.Equal(t, false, merged.VolumeMounts[0].ReadOnly)
	assert.Equal(t, "data2", merged.VolumeMounts[1].Name)
}

func TestMergeVolumes(t *testing.T) {
	generalSpec := v2.JaegerCommonSpec{
		Volumes: []corev1.Volume{{
			Name:         "volume1",
			VolumeSource: corev1.VolumeSource{HostPath: &corev1.HostPathVolumeSource{Path: "/data3"}},
		}},
	}
	specificSpec := v2.JaegerCommonSpec{
		Volumes: []corev1.Volume{{
			Name:         "volume1",
			VolumeSource: corev1.VolumeSource{HostPath: &corev1.HostPathVolumeSource{Path: "/data1"}},
		}, {
			Name:         "volume2",
			VolumeSource: corev1.VolumeSource{HostPath: &corev1.HostPathVolumeSource{Path: "/data2"}},
		}},
	}

	merged := Merge(specificSpec, generalSpec)

	assert.Equal(t, "volume1", merged.Volumes[0].Name)
	assert.Equal(t, "/data1", merged.Volumes[0].VolumeSource.HostPath.Path)
	assert.Equal(t, "volume2", merged.Volumes[1].Name)
}

func TestMergeResourceLimits(t *testing.T) {
	generalSpec := v2.JaegerCommonSpec{
		Resources: corev1.ResourceRequirements{
			Limits: corev1.ResourceList{
				corev1.ResourceLimitsCPU:              *resource.NewQuantity(1024, resource.BinarySI),
				corev1.ResourceLimitsEphemeralStorage: *resource.NewQuantity(123, resource.DecimalSI),
			},
		},
	}
	specificSpec := v2.JaegerCommonSpec{
		Resources: corev1.ResourceRequirements{
			Limits: corev1.ResourceList{
				corev1.ResourceLimitsCPU:    *resource.NewQuantity(2048, resource.BinarySI),
				corev1.ResourceLimitsMemory: *resource.NewQuantity(1024, resource.BinarySI),
			},
		},
	}

	merged := Merge(specificSpec, generalSpec)

	assert.Equal(t, *resource.NewQuantity(2048, resource.BinarySI), merged.Resources.Limits[corev1.ResourceLimitsCPU])
	assert.Equal(t, *resource.NewQuantity(1024, resource.BinarySI), merged.Resources.Limits[corev1.ResourceLimitsMemory])
	assert.Equal(t, *resource.NewQuantity(123, resource.DecimalSI), merged.Resources.Limits[corev1.ResourceLimitsEphemeralStorage])
}

func TestMergeResourceRequests(t *testing.T) {
	generalSpec := v2.JaegerCommonSpec{
		Resources: corev1.ResourceRequirements{
			Requests: corev1.ResourceList{
				corev1.ResourceRequestsCPU:              *resource.NewQuantity(1024, resource.BinarySI),
				corev1.ResourceRequestsEphemeralStorage: *resource.NewQuantity(123, resource.DecimalSI),
			},
		},
	}
	specificSpec := v2.JaegerCommonSpec{
		Resources: corev1.ResourceRequirements{
			Requests: corev1.ResourceList{
				corev1.ResourceRequestsCPU:    *resource.NewQuantity(2048, resource.BinarySI),
				corev1.ResourceRequestsMemory: *resource.NewQuantity(1024, resource.BinarySI),
			},
		},
	}

	merged := Merge(specificSpec, generalSpec)

	assert.Equal(t, *resource.NewQuantity(2048, resource.BinarySI), merged.Resources.Requests[corev1.ResourceRequestsCPU])
	assert.Equal(t, *resource.NewQuantity(1024, resource.BinarySI), merged.Resources.Requests[corev1.ResourceRequestsMemory])
	assert.Equal(t, *resource.NewQuantity(123, resource.DecimalSI), merged.Resources.Requests[corev1.ResourceRequestsEphemeralStorage])
}
