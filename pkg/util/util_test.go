package util

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"

	"github.com/jaegertracing/jaeger-operator/pkg/apis/io/v1alpha1"
)

func TestRemoveDuplicatedVolumes(t *testing.T) {
	volumes := []v1.Volume{
		v1.Volume{
			Name:         "volume1",
			VolumeSource: v1.VolumeSource{HostPath: &v1.HostPathVolumeSource{Path: "/data1"}},
		},
		v1.Volume{
			Name:         "volume2",
			VolumeSource: v1.VolumeSource{HostPath: &v1.HostPathVolumeSource{Path: "/data2"}},
		},
		v1.Volume{
			Name:         "volume1",
			VolumeSource: v1.VolumeSource{HostPath: &v1.HostPathVolumeSource{Path: "/data3"}},
		},
	}

	assert.Len(t, removeDuplicatedVolumes(volumes), 2)
	assert.Equal(t, "volume1", volumes[0].Name)
	assert.Equal(t, "/data1", volumes[0].VolumeSource.HostPath.Path)
	assert.Equal(t, "volume2", volumes[1].Name)
}

func TestRemoveDuplicatedVolumeMounts(t *testing.T) {
	volumeMounts := []v1.VolumeMount{
		v1.VolumeMount{
			Name:     "data1",
			ReadOnly: false,
		},
		v1.VolumeMount{
			Name:     "data2",
			ReadOnly: false,
		},
		v1.VolumeMount{
			Name:     "data1",
			ReadOnly: true,
		},
	}

	assert.Len(t, removeDuplicatedVolumeMounts(volumeMounts), 2)
	assert.Equal(t, "data1", volumeMounts[0].Name)
	assert.Equal(t, false, volumeMounts[0].ReadOnly)
	assert.Equal(t, "data2", volumeMounts[1].Name)
}

func TestMergeAnnotations(t *testing.T) {
	generalSpec := v1alpha1.JaegerCommonSpec{
		Annotations: map[string]string{
			"name":  "operator",
			"hello": "jaeger",
		},
	}
	specificSpec := v1alpha1.JaegerCommonSpec{
		Annotations: map[string]string{
			"hello":                "world", // Override general annotation
			"prometheus.io/scrape": "false",
		},
	}

	merged := Merge([]v1alpha1.JaegerCommonSpec{specificSpec, generalSpec})

	assert.Equal(t, "operator", merged.Annotations["name"])
	assert.Equal(t, "world", merged.Annotations["hello"])
	assert.Equal(t, "false", merged.Annotations["prometheus.io/scrape"])
}

func TestMergeMountVolumes(t *testing.T) {
	generalSpec := v1alpha1.JaegerCommonSpec{
		VolumeMounts: []v1.VolumeMount{
			v1.VolumeMount{
				Name:     "data1",
				ReadOnly: true,
			},
		},
	}
	specificSpec := v1alpha1.JaegerCommonSpec{
		VolumeMounts: []v1.VolumeMount{
			v1.VolumeMount{
				Name:     "data1",
				ReadOnly: false,
			},
			v1.VolumeMount{
				Name:     "data2",
				ReadOnly: false,
			},
		},
	}

	merged := Merge([]v1alpha1.JaegerCommonSpec{specificSpec, generalSpec})

	assert.Equal(t, "data1", merged.VolumeMounts[0].Name)
	assert.Equal(t, false, merged.VolumeMounts[0].ReadOnly)
	assert.Equal(t, "data2", merged.VolumeMounts[1].Name)
}

func TestMergeVolumes(t *testing.T) {
	generalSpec := v1alpha1.JaegerCommonSpec{
		Volumes: []v1.Volume{
			v1.Volume{
				Name:         "volume1",
				VolumeSource: v1.VolumeSource{HostPath: &v1.HostPathVolumeSource{Path: "/data3"}},
			},
		},
	}
	specificSpec := v1alpha1.JaegerCommonSpec{
		Volumes: []v1.Volume{
			v1.Volume{
				Name:         "volume1",
				VolumeSource: v1.VolumeSource{HostPath: &v1.HostPathVolumeSource{Path: "/data1"}},
			},
			v1.Volume{
				Name:         "volume2",
				VolumeSource: v1.VolumeSource{HostPath: &v1.HostPathVolumeSource{Path: "/data2"}},
			},
		},
	}

	merged := Merge([]v1alpha1.JaegerCommonSpec{specificSpec, generalSpec})

	assert.Equal(t, "volume1", merged.Volumes[0].Name)
	assert.Equal(t, "/data1", merged.Volumes[0].VolumeSource.HostPath.Path)
	assert.Equal(t, "volume2", merged.Volumes[1].Name)
}

func TestMergeResourceLimits(t *testing.T) {
	generalSpec := v1alpha1.JaegerCommonSpec{
		Resources: v1.ResourceRequirements{
			Limits: v1.ResourceList{
				v1.ResourceLimitsCPU:              *resource.NewQuantity(1024, resource.BinarySI),
				v1.ResourceLimitsEphemeralStorage: *resource.NewQuantity(123, resource.DecimalSI),
			},
		},
	}
	specificSpec := v1alpha1.JaegerCommonSpec{
		Resources: v1.ResourceRequirements{
			Limits: v1.ResourceList{
				v1.ResourceLimitsCPU:    *resource.NewQuantity(2048, resource.BinarySI),
				v1.ResourceLimitsMemory: *resource.NewQuantity(1024, resource.BinarySI),
			},
		},
	}

	merged := Merge([]v1alpha1.JaegerCommonSpec{specificSpec, generalSpec})

	assert.Equal(t, *resource.NewQuantity(2048, resource.BinarySI), merged.Resources.Limits[v1.ResourceLimitsCPU])
	assert.Equal(t, *resource.NewQuantity(1024, resource.BinarySI), merged.Resources.Limits[v1.ResourceLimitsMemory])
	assert.Equal(t, *resource.NewQuantity(123, resource.DecimalSI), merged.Resources.Limits[v1.ResourceLimitsEphemeralStorage])
}

func TestMergeResourceRequests(t *testing.T) {
	generalSpec := v1alpha1.JaegerCommonSpec{
		Resources: v1.ResourceRequirements{
			Requests: v1.ResourceList{
				v1.ResourceRequestsCPU:              *resource.NewQuantity(1024, resource.BinarySI),
				v1.ResourceRequestsEphemeralStorage: *resource.NewQuantity(123, resource.DecimalSI),
			},
		},
	}
	specificSpec := v1alpha1.JaegerCommonSpec{
		Resources: v1.ResourceRequirements{
			Requests: v1.ResourceList{
				v1.ResourceRequestsCPU:    *resource.NewQuantity(2048, resource.BinarySI),
				v1.ResourceRequestsMemory: *resource.NewQuantity(1024, resource.BinarySI),
			},
		},
	}

	merged := Merge([]v1alpha1.JaegerCommonSpec{specificSpec, generalSpec})

	assert.Equal(t, *resource.NewQuantity(2048, resource.BinarySI), merged.Resources.Requests[v1.ResourceRequestsCPU])
	assert.Equal(t, *resource.NewQuantity(1024, resource.BinarySI), merged.Resources.Requests[v1.ResourceRequestsMemory])
	assert.Equal(t, *resource.NewQuantity(123, resource.DecimalSI), merged.Resources.Requests[v1.ResourceRequestsEphemeralStorage])
}
