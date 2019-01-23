package deployment

import (
	"fmt"
	"testing"

	"github.com/spf13/viper"
	"github.com/stretchr/testify/assert"
	"k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"

	"github.com/jaegertracing/jaeger-operator/pkg/apis/io/v1alpha1"
)

func init() {
	viper.SetDefault("jaeger-version", "1.6")
	viper.SetDefault("jaeger-collector-image", "jaegertracing/all-in-one")
}

func TestNegativeSize(t *testing.T) {
	jaeger := v1alpha1.NewJaeger("TestNegativeSize")
	jaeger.Spec.Collector.Size = -1

	collector := NewCollector(jaeger)
	dep := collector.Get()
	assert.Equal(t, int32(1), *dep.Spec.Replicas)
}

func TestDefaultSize(t *testing.T) {
	jaeger := v1alpha1.NewJaeger("TestDefaultSize")
	jaeger.Spec.Collector.Size = 0

	collector := NewCollector(jaeger)
	dep := collector.Get()
	assert.Equal(t, int32(1), *dep.Spec.Replicas)
}

func TestName(t *testing.T) {
	collector := NewCollector(v1alpha1.NewJaeger("TestName"))
	dep := collector.Get()
	assert.Equal(t, "TestName-collector", dep.ObjectMeta.Name)
}

func TestCollectorServices(t *testing.T) {
	collector := NewCollector(v1alpha1.NewJaeger("TestName"))
	svcs := collector.Services()
	assert.Len(t, svcs, 1)
}

func TestDefaultCollectorImage(t *testing.T) {
	viper.Set("jaeger-collector-image", "org/custom-collector-image")
	defer viper.Reset()

	jaeger := v1alpha1.NewJaeger("TestCollectorImage")
	jaeger.Spec.Version = "123"
	collector := NewCollector(jaeger)
	dep := collector.Get()

	containers := dep.Spec.Template.Spec.Containers
	assert.Len(t, containers, 1)
	assert.Equal(t, "org/custom-collector-image:123", containers[0].Image)

	envvars := []v1.EnvVar{
		v1.EnvVar{
			Name:  "SPAN_STORAGE_TYPE",
			Value: "",
		},
		v1.EnvVar{
			Name:  "COLLECTOR_ZIPKIN_HTTP_PORT",
			Value: "9411",
		},
	}
	assert.Equal(t, envvars, containers[0].Env)
}

func TestCollectorAnnotations(t *testing.T) {
	jaeger := v1alpha1.NewJaeger("TestCollectorAnnotations")
	jaeger.Spec.Annotations = map[string]string{
		"name":  "operator",
		"hello": "jaeger",
	}
	jaeger.Spec.Collector.Annotations = map[string]string{
		"hello":                "world", // Override top level annotation
		"prometheus.io/scrape": "false", // Override implicit value
	}

	collector := NewCollector(jaeger)
	dep := collector.Get()

	assert.Equal(t, "operator", dep.Spec.Template.Annotations["name"])
	assert.Equal(t, "false", dep.Spec.Template.Annotations["sidecar.istio.io/inject"])
	assert.Equal(t, "world", dep.Spec.Template.Annotations["hello"])
	assert.Equal(t, "false", dep.Spec.Template.Annotations["prometheus.io/scrape"])
}

func TestCollectorSecrets(t *testing.T) {
	jaeger := v1alpha1.NewJaeger("TestCollectorSecrets")
	secret := "mysecret"
	jaeger.Spec.Storage.SecretName = secret

	collector := NewCollector(jaeger)
	dep := collector.Get()

	assert.Equal(t, "mysecret", dep.Spec.Template.Spec.Containers[0].EnvFrom[0].SecretRef.LocalObjectReference.Name)
}

func TestCollectorVolumeMountsWithVolumes(t *testing.T) {
	name := "TestCollectorVolumeMountsWithVolumes"

	globalVolumes := []v1.Volume{
		v1.Volume{
			Name:         "globalVolume",
			VolumeSource: v1.VolumeSource{},
		},
	}

	globalVolumeMounts := []v1.VolumeMount{
		v1.VolumeMount{
			Name: "globalVolume",
		},
	}

	collectorVolumes := []v1.Volume{
		v1.Volume{
			Name:         "collectorVolume",
			VolumeSource: v1.VolumeSource{},
		},
	}

	collectorVolumeMounts := []v1.VolumeMount{
		v1.VolumeMount{
			Name: "collectorVolume",
		},
	}

	jaeger := v1alpha1.NewJaeger(name)
	jaeger.Spec.Volumes = globalVolumes
	jaeger.Spec.VolumeMounts = globalVolumeMounts
	jaeger.Spec.Collector.Volumes = collectorVolumes
	jaeger.Spec.Collector.VolumeMounts = collectorVolumeMounts
	podSpec := NewCollector(jaeger).Get().Spec.Template.Spec

	// Additional 1 is sampling configmap
	assert.Len(t, podSpec.Volumes, len(append(collectorVolumes, globalVolumes...))+1)
	assert.Len(t, podSpec.Containers[0].VolumeMounts, len(append(collectorVolumeMounts, globalVolumeMounts...))+1)

	// collector is first while global is second
	assert.Equal(t, "collectorVolume", podSpec.Volumes[0].Name)
	assert.Equal(t, "globalVolume", podSpec.Volumes[1].Name)
	assert.Equal(t, "collectorVolume", podSpec.Containers[0].VolumeMounts[0].Name)
	assert.Equal(t, "globalVolume", podSpec.Containers[0].VolumeMounts[1].Name)
}

func TestCollectorMountGlobalVolumes(t *testing.T) {
	name := "TestCollectorMountGlobalVolumes"

	globalVolumes := []v1.Volume{
		v1.Volume{
			Name:         "globalVolume",
			VolumeSource: v1.VolumeSource{},
		},
	}

	collectorVolumeMounts := []v1.VolumeMount{
		v1.VolumeMount{
			Name:     "globalVolume",
			ReadOnly: true,
		},
	}

	jaeger := v1alpha1.NewJaeger(name)
	jaeger.Spec.Volumes = globalVolumes
	jaeger.Spec.Collector.VolumeMounts = collectorVolumeMounts
	podSpec := NewCollector(jaeger).Get().Spec.Template.Spec

	// Count includes the sampling configmap
	assert.Len(t, podSpec.Containers[0].VolumeMounts, 2)
	// collector volume is mounted
	assert.Equal(t, podSpec.Containers[0].VolumeMounts[0].Name, "globalVolume")
}

func TestCollectorVolumeMountsWithSameName(t *testing.T) {
	name := "TestCollectorVolumeMountsWithSameName"

	globalVolumeMounts := []v1.VolumeMount{
		v1.VolumeMount{
			Name:     "data",
			ReadOnly: true,
		},
	}

	collectorVolumeMounts := []v1.VolumeMount{
		v1.VolumeMount{
			Name:     "data",
			ReadOnly: false,
		},
	}

	jaeger := v1alpha1.NewJaeger(name)
	jaeger.Spec.VolumeMounts = globalVolumeMounts
	jaeger.Spec.Collector.VolumeMounts = collectorVolumeMounts
	podSpec := NewCollector(jaeger).Get().Spec.Template.Spec

	// Count includes the sampling configmap
	assert.Len(t, podSpec.Containers[0].VolumeMounts, 2)
	// collector volume is mounted
	assert.Equal(t, podSpec.Containers[0].VolumeMounts[0].ReadOnly, false)
}

func TestCollectorVolumeWithSameName(t *testing.T) {
	name := "TestCollectorVolumeWithSameName"

	globalVolumes := []v1.Volume{
		v1.Volume{
			Name:         "data",
			VolumeSource: v1.VolumeSource{HostPath: &v1.HostPathVolumeSource{Path: "/data1"}},
		},
	}

	collectorVolumes := []v1.Volume{
		v1.Volume{
			Name:         "data",
			VolumeSource: v1.VolumeSource{HostPath: &v1.HostPathVolumeSource{Path: "/data2"}},
		},
	}

	jaeger := v1alpha1.NewJaeger(name)
	jaeger.Spec.Volumes = globalVolumes
	jaeger.Spec.Collector.Volumes = collectorVolumes
	podSpec := NewCollector(jaeger).Get().Spec.Template.Spec

	// Count includes the sampling configmap
	assert.Len(t, podSpec.Volumes, 2)
	// collector volume is mounted
	assert.Equal(t, podSpec.Volumes[0].VolumeSource.HostPath.Path, "/data2")
}

func TestCollectorResources(t *testing.T) {
	jaeger := v1alpha1.NewJaeger("TestCollectorResources")
	jaeger.Spec.Resources = v1.ResourceRequirements{
		Limits: v1.ResourceList{
			v1.ResourceLimitsCPU:              *resource.NewQuantity(1024, resource.BinarySI),
			v1.ResourceLimitsEphemeralStorage: *resource.NewQuantity(512, resource.DecimalSI),
		},
		Requests: v1.ResourceList{
			v1.ResourceRequestsCPU:              *resource.NewQuantity(1024, resource.BinarySI),
			v1.ResourceRequestsEphemeralStorage: *resource.NewQuantity(512, resource.DecimalSI),
		},
	}
	jaeger.Spec.Collector.Resources = v1.ResourceRequirements{
		Limits: v1.ResourceList{
			v1.ResourceLimitsCPU:    *resource.NewQuantity(2048, resource.BinarySI),
			v1.ResourceLimitsMemory: *resource.NewQuantity(123, resource.DecimalSI),
		},
		Requests: v1.ResourceList{
			v1.ResourceRequestsCPU:    *resource.NewQuantity(2048, resource.BinarySI),
			v1.ResourceRequestsMemory: *resource.NewQuantity(123, resource.DecimalSI),
		},
	}

	collector := NewCollector(jaeger)
	dep := collector.Get()

	assert.Equal(t, *resource.NewQuantity(2048, resource.BinarySI), dep.Spec.Template.Spec.Containers[0].Resources.Limits[v1.ResourceLimitsCPU])
	assert.Equal(t, *resource.NewQuantity(2048, resource.BinarySI), dep.Spec.Template.Spec.Containers[0].Resources.Requests[v1.ResourceRequestsCPU])
	assert.Equal(t, *resource.NewQuantity(123, resource.DecimalSI), dep.Spec.Template.Spec.Containers[0].Resources.Limits[v1.ResourceLimitsMemory])
	assert.Equal(t, *resource.NewQuantity(123, resource.DecimalSI), dep.Spec.Template.Spec.Containers[0].Resources.Requests[v1.ResourceRequestsMemory])
	assert.Equal(t, *resource.NewQuantity(512, resource.DecimalSI), dep.Spec.Template.Spec.Containers[0].Resources.Limits[v1.ResourceLimitsEphemeralStorage])
	assert.Equal(t, *resource.NewQuantity(512, resource.DecimalSI), dep.Spec.Template.Spec.Containers[0].Resources.Requests[v1.ResourceRequestsEphemeralStorage])
}

func TestCollectorLabels(t *testing.T) {
	c := NewCollector(v1alpha1.NewJaeger("TestCollectorLabels"))
	dep := c.Get()
	assert.Equal(t, "jaeger-operator", dep.Spec.Template.Labels["app.kubernetes.io/managed-by"])
	assert.Equal(t, "collector", dep.Spec.Template.Labels["app.kubernetes.io/component"])
	assert.Equal(t, c.jaeger.Name, dep.Spec.Template.Labels["app.kubernetes.io/instance"])
	assert.Equal(t, fmt.Sprintf("%s-collector", c.jaeger.Name), dep.Spec.Template.Labels["app.kubernetes.io/name"])
}
