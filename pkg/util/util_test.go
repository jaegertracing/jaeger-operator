package util

import (
	"testing"

	"github.com/stretchr/testify/assert"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/jaegertracing/jaeger-operator/pkg/apis/jaegertracing/v1"
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

	assert.Len(t, removeDuplicatedVolumes(volumes), 2)
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

	assert.Len(t, removeDuplicatedVolumeMounts(volumeMounts), 2)
	assert.Equal(t, "data1", volumeMounts[0].Name)
	assert.Equal(t, false, volumeMounts[0].ReadOnly)
	assert.Equal(t, "data2", volumeMounts[1].Name)
}

func TestMergeAnnotations(t *testing.T) {
	generalSpec := v1.JaegerCommonSpec{
		Annotations: map[string]string{
			"name":  "operator",
			"hello": "jaeger",
		},
	}
	specificSpec := v1.JaegerCommonSpec{
		Annotations: map[string]string{
			"hello":                "world", // Override general annotation
			"prometheus.io/scrape": "false",
		},
	}

	merged := Merge([]v1.JaegerCommonSpec{specificSpec, generalSpec})

	assert.Equal(t, "operator", merged.Annotations["name"])
	assert.Equal(t, "world", merged.Annotations["hello"])
	assert.Equal(t, "false", merged.Annotations["prometheus.io/scrape"])
}

func TestMergeMountVolumes(t *testing.T) {
	generalSpec := v1.JaegerCommonSpec{
		VolumeMounts: []corev1.VolumeMount{{
			Name:     "data1",
			ReadOnly: true,
		}},
	}
	specificSpec := v1.JaegerCommonSpec{
		VolumeMounts: []corev1.VolumeMount{{
			Name:     "data1",
			ReadOnly: false,
		}, {
			Name:     "data2",
			ReadOnly: false,
		}},
	}

	merged := Merge([]v1.JaegerCommonSpec{specificSpec, generalSpec})

	assert.Equal(t, "data1", merged.VolumeMounts[0].Name)
	assert.Equal(t, false, merged.VolumeMounts[0].ReadOnly)
	assert.Equal(t, "data2", merged.VolumeMounts[1].Name)
}

func TestMergeVolumes(t *testing.T) {
	generalSpec := v1.JaegerCommonSpec{
		Volumes: []corev1.Volume{{
			Name:         "volume1",
			VolumeSource: corev1.VolumeSource{HostPath: &corev1.HostPathVolumeSource{Path: "/data3"}},
		}},
	}
	specificSpec := v1.JaegerCommonSpec{
		Volumes: []corev1.Volume{{
			Name:         "volume1",
			VolumeSource: corev1.VolumeSource{HostPath: &corev1.HostPathVolumeSource{Path: "/data1"}},
		}, {
			Name:         "volume2",
			VolumeSource: corev1.VolumeSource{HostPath: &corev1.HostPathVolumeSource{Path: "/data2"}},
		}},
	}

	merged := Merge([]v1.JaegerCommonSpec{specificSpec, generalSpec})

	assert.Equal(t, "volume1", merged.Volumes[0].Name)
	assert.Equal(t, "/data1", merged.Volumes[0].VolumeSource.HostPath.Path)
	assert.Equal(t, "volume2", merged.Volumes[1].Name)
}

func TestMergeResourceLimits(t *testing.T) {
	generalSpec := v1.JaegerCommonSpec{
		Resources: corev1.ResourceRequirements{
			Limits: corev1.ResourceList{
				corev1.ResourceLimitsCPU:              *resource.NewQuantity(1024, resource.BinarySI),
				corev1.ResourceLimitsEphemeralStorage: *resource.NewQuantity(123, resource.DecimalSI),
			},
		},
	}
	specificSpec := v1.JaegerCommonSpec{
		Resources: corev1.ResourceRequirements{
			Limits: corev1.ResourceList{
				corev1.ResourceLimitsCPU:    *resource.NewQuantity(2048, resource.BinarySI),
				corev1.ResourceLimitsMemory: *resource.NewQuantity(1024, resource.BinarySI),
			},
		},
	}

	merged := Merge([]v1.JaegerCommonSpec{specificSpec, generalSpec})

	assert.Equal(t, *resource.NewQuantity(2048, resource.BinarySI), merged.Resources.Limits[corev1.ResourceLimitsCPU])
	assert.Equal(t, *resource.NewQuantity(1024, resource.BinarySI), merged.Resources.Limits[corev1.ResourceLimitsMemory])
	assert.Equal(t, *resource.NewQuantity(123, resource.DecimalSI), merged.Resources.Limits[corev1.ResourceLimitsEphemeralStorage])
}

func TestMergeResourceRequests(t *testing.T) {
	generalSpec := v1.JaegerCommonSpec{
		Resources: corev1.ResourceRequirements{
			Requests: corev1.ResourceList{
				corev1.ResourceRequestsCPU:              *resource.NewQuantity(1024, resource.BinarySI),
				corev1.ResourceRequestsEphemeralStorage: *resource.NewQuantity(123, resource.DecimalSI),
			},
		},
	}
	specificSpec := v1.JaegerCommonSpec{
		Resources: corev1.ResourceRequirements{
			Requests: corev1.ResourceList{
				corev1.ResourceRequestsCPU:    *resource.NewQuantity(2048, resource.BinarySI),
				corev1.ResourceRequestsMemory: *resource.NewQuantity(1024, resource.BinarySI),
			},
		},
	}

	merged := Merge([]v1.JaegerCommonSpec{specificSpec, generalSpec})

	assert.Equal(t, *resource.NewQuantity(2048, resource.BinarySI), merged.Resources.Requests[corev1.ResourceRequestsCPU])
	assert.Equal(t, *resource.NewQuantity(1024, resource.BinarySI), merged.Resources.Requests[corev1.ResourceRequestsMemory])
	assert.Equal(t, *resource.NewQuantity(123, resource.DecimalSI), merged.Resources.Requests[corev1.ResourceRequestsEphemeralStorage])
}

func TestGetEsHostname(t *testing.T) {
	tests := []struct {
		underTest map[string]string
		hostname  string
	}{
		{hostname: ""},
		{underTest: map[string]string{"": ""}, hostname: ""},
		{underTest: map[string]string{"es.server-urls": ""}, hostname: ""},
		{underTest: map[string]string{"es.server-urls": "goo:tar"}, hostname: "goo:tar"},
		{underTest: map[string]string{"es.server-urls": "http://es:9000,https://es2:9200"}, hostname: "http://es:9000"},
	}
	for _, test := range tests {
		assert.Equal(t, test.hostname, GetEsHostname(test.underTest))
	}
}

func TestAsOwner(t *testing.T) {
	j := v1.NewJaeger("joe")
	j.Kind = "human"
	j.APIVersion = "homosapiens"
	j.UID = "boom!"
	ow := AsOwner(j)
	trueVar := true
	assert.Equal(t, metav1.OwnerReference{Name: "joe", Kind: "human", APIVersion: "homosapiens", UID: "boom!", Controller: &trueVar}, ow)
}

func TestLabels(t *testing.T) {
	assert.Equal(t, map[string]string{
		"app":                          "jaeger",
		"app.kubernetes.io/name":       "joe",
		"app.kubernetes.io/instance":   "thatone",
		"app.kubernetes.io/component":  "leg",
		"app.kubernetes.io/part-of":    "jaeger",
		"app.kubernetes.io/managed-by": "jaeger-operator",
	}, Labels("joe", "leg", *v1.NewJaeger("thatone")))
}

func TestFindItem(t *testing.T) {
	opts := v1.NewOptions(map[string]interface{}{
		"reporter.type":             "thrift",
		"reporter.thrift.host-port": "collector:14267",
	})

	args := opts.ToArgs()

	assert.Equal(t, "--reporter.type=thrift", FindItem("--reporter.type=", args))
	assert.Len(t, FindItem("--c-option", args), 0)
}

func TestGetPortDefault(t *testing.T) {
	opts := v1.NewOptions(map[string]interface{}{})

	args := opts.ToArgs()

	assert.Equal(t, int32(1234), GetPort("--processor.jaeger-compact.server-host-port=", args, 1234))
}

func TestGetPortSpecified(t *testing.T) {
	opts := v1.NewOptions(map[string]interface{}{
		"processor.jaeger-compact.server-host-port": ":6831",
	})

	args := opts.ToArgs()

	assert.Equal(t, int32(6831), GetPort("--processor.jaeger-compact.server-host-port=", args, 1234))
}
