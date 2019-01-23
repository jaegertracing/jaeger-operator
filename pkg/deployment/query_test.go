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
	viper.SetDefault("jaeger-query-image", "jaegertracing/all-in-one")
}

func TestQueryNegativeSize(t *testing.T) {
	jaeger := v1alpha1.NewJaeger("TestQueryNegativeSize")
	jaeger.Spec.Query.Size = -1

	query := NewQuery(jaeger)
	dep := query.Get()
	assert.Equal(t, int32(1), *dep.Spec.Replicas)
}

func TestQueryDefaultSize(t *testing.T) {
	jaeger := v1alpha1.NewJaeger("TestQueryDefaultSize")
	jaeger.Spec.Query.Size = 0

	query := NewQuery(jaeger)
	dep := query.Get()
	assert.Equal(t, int32(1), *dep.Spec.Replicas)
}

func TestDefaultQueryImage(t *testing.T) {
	viper.Set("jaeger-query-image", "org/custom-query-image")
	defer viper.Reset()

	jaeger := v1alpha1.NewJaeger("TestQueryImage")
	jaeger.Spec.Version = "123"
	query := NewQuery(jaeger)
	dep := query.Get()
	containers := dep.Spec.Template.Spec.Containers

	assert.Len(t, containers, 1)
	assert.Equal(t, "org/custom-query-image:123", containers[0].Image)
}

func TestQueryAnnotations(t *testing.T) {
	jaeger := v1alpha1.NewJaeger("TestQueryAnnotations")
	jaeger.Spec.Annotations = map[string]string{
		"name":  "operator",
		"hello": "jaeger",
	}
	jaeger.Spec.Query.Annotations = map[string]string{
		"hello":                "world", // Override top level annotation
		"prometheus.io/scrape": "false", // Override implicit value
	}

	query := NewQuery(jaeger)
	dep := query.Get()

	assert.Equal(t, "operator", dep.Spec.Template.Annotations["name"])
	assert.Equal(t, "false", dep.Spec.Template.Annotations["sidecar.istio.io/inject"])
	assert.Equal(t, "world", dep.Spec.Template.Annotations["hello"])
	assert.Equal(t, "false", dep.Spec.Template.Annotations["prometheus.io/scrape"])
}

func TestQuerySecrets(t *testing.T) {
	jaeger := v1alpha1.NewJaeger("TestQuerySecrets")
	secret := "mysecret"
	jaeger.Spec.Storage.SecretName = secret

	query := NewQuery(jaeger)
	dep := query.Get()

	assert.Equal(t, "mysecret", dep.Spec.Template.Spec.Containers[0].EnvFrom[0].SecretRef.LocalObjectReference.Name)
}

func TestQueryPodName(t *testing.T) {
	name := "TestQueryPodName"
	query := NewQuery(v1alpha1.NewJaeger(name))
	dep := query.Get()

	assert.Contains(t, dep.ObjectMeta.Name, fmt.Sprintf("%s-query", name))
}

func TestQueryServices(t *testing.T) {
	query := NewQuery(v1alpha1.NewJaeger("TestQueryServices"))
	svcs := query.Services()

	assert.Len(t, svcs, 1)
}

func TestQueryVolumeMountsWithVolumes(t *testing.T) {
	name := "TestQueryVolumeMountsWithVolumes"

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

	queryVolumes := []v1.Volume{
		v1.Volume{
			Name:         "queryVolume",
			VolumeSource: v1.VolumeSource{},
		},
	}

	queryVolumeMounts := []v1.VolumeMount{
		v1.VolumeMount{
			Name: "queryVolume",
		},
	}

	jaeger := v1alpha1.NewJaeger(name)
	jaeger.Spec.Volumes = globalVolumes
	jaeger.Spec.VolumeMounts = globalVolumeMounts
	jaeger.Spec.Query.Volumes = queryVolumes
	jaeger.Spec.Query.VolumeMounts = queryVolumeMounts
	podSpec := NewQuery(jaeger).Get().Spec.Template.Spec

	assert.Len(t, podSpec.Volumes, len(append(queryVolumes, globalVolumes...)))
	assert.Len(t, podSpec.Containers[0].VolumeMounts, len(append(queryVolumeMounts, globalVolumeMounts...)))

	// query is first while global is second
	assert.Equal(t, "queryVolume", podSpec.Volumes[0].Name)
	assert.Equal(t, "globalVolume", podSpec.Volumes[1].Name)
	assert.Equal(t, "queryVolume", podSpec.Containers[0].VolumeMounts[0].Name)
	assert.Equal(t, "globalVolume", podSpec.Containers[0].VolumeMounts[1].Name)
}

func TestQueryMountGlobalVolumes(t *testing.T) {
	name := "TestQueryMountGlobalVolumes"

	globalVolumes := []v1.Volume{
		v1.Volume{
			Name:         "globalVolume",
			VolumeSource: v1.VolumeSource{},
		},
	}

	queryVolumeMounts := []v1.VolumeMount{
		v1.VolumeMount{
			Name:     "globalVolume",
			ReadOnly: true,
		},
	}

	jaeger := v1alpha1.NewJaeger(name)
	jaeger.Spec.Volumes = globalVolumes
	jaeger.Spec.Query.VolumeMounts = queryVolumeMounts
	podSpec := NewQuery(jaeger).Get().Spec.Template.Spec

	assert.Len(t, podSpec.Containers[0].VolumeMounts, 1)
	// query volume is mounted
	assert.Equal(t, podSpec.Containers[0].VolumeMounts[0].Name, "globalVolume")
}

func TestQueryVolumeMountsWithSameName(t *testing.T) {
	name := "TestQueryVolumeMountsWithSameName"

	globalVolumeMounts := []v1.VolumeMount{
		v1.VolumeMount{
			Name:     "data",
			ReadOnly: true,
		},
	}

	queryVolumeMounts := []v1.VolumeMount{
		v1.VolumeMount{
			Name:     "data",
			ReadOnly: false,
		},
	}

	jaeger := v1alpha1.NewJaeger(name)
	jaeger.Spec.VolumeMounts = globalVolumeMounts
	jaeger.Spec.Query.VolumeMounts = queryVolumeMounts
	podSpec := NewQuery(jaeger).Get().Spec.Template.Spec

	assert.Len(t, podSpec.Containers[0].VolumeMounts, 1)
	// query volume is mounted
	assert.Equal(t, podSpec.Containers[0].VolumeMounts[0].ReadOnly, false)
}

func TestQueryVolumeWithSameName(t *testing.T) {
	name := "TestQueryVolumeWithSameName"

	globalVolumes := []v1.Volume{
		v1.Volume{
			Name:         "data",
			VolumeSource: v1.VolumeSource{HostPath: &v1.HostPathVolumeSource{Path: "/data1"}},
		},
	}

	queryVolumes := []v1.Volume{
		v1.Volume{
			Name:         "data",
			VolumeSource: v1.VolumeSource{HostPath: &v1.HostPathVolumeSource{Path: "/data2"}},
		},
	}

	jaeger := v1alpha1.NewJaeger(name)
	jaeger.Spec.Volumes = globalVolumes
	jaeger.Spec.Query.Volumes = queryVolumes
	podSpec := NewQuery(jaeger).Get().Spec.Template.Spec

	assert.Len(t, podSpec.Volumes, 1)
	// query volume is mounted
	assert.Equal(t, podSpec.Volumes[0].VolumeSource.HostPath.Path, "/data2")
}

func TestQueryResources(t *testing.T) {
	jaeger := v1alpha1.NewJaeger("TestQueryResources")
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
	jaeger.Spec.Query.Resources = v1.ResourceRequirements{
		Limits: v1.ResourceList{
			v1.ResourceLimitsCPU:    *resource.NewQuantity(2048, resource.BinarySI),
			v1.ResourceLimitsMemory: *resource.NewQuantity(123, resource.DecimalSI),
		},
		Requests: v1.ResourceList{
			v1.ResourceRequestsCPU:    *resource.NewQuantity(2048, resource.BinarySI),
			v1.ResourceRequestsMemory: *resource.NewQuantity(123, resource.DecimalSI),
		},
	}

	query := NewQuery(jaeger)
	dep := query.Get()

	assert.Equal(t, *resource.NewQuantity(2048, resource.BinarySI), dep.Spec.Template.Spec.Containers[0].Resources.Limits[v1.ResourceLimitsCPU])
	assert.Equal(t, *resource.NewQuantity(2048, resource.BinarySI), dep.Spec.Template.Spec.Containers[0].Resources.Requests[v1.ResourceRequestsCPU])
	assert.Equal(t, *resource.NewQuantity(123, resource.DecimalSI), dep.Spec.Template.Spec.Containers[0].Resources.Limits[v1.ResourceLimitsMemory])
	assert.Equal(t, *resource.NewQuantity(123, resource.DecimalSI), dep.Spec.Template.Spec.Containers[0].Resources.Requests[v1.ResourceRequestsMemory])
	assert.Equal(t, *resource.NewQuantity(512, resource.DecimalSI), dep.Spec.Template.Spec.Containers[0].Resources.Limits[v1.ResourceLimitsEphemeralStorage])
	assert.Equal(t, *resource.NewQuantity(512, resource.DecimalSI), dep.Spec.Template.Spec.Containers[0].Resources.Requests[v1.ResourceRequestsEphemeralStorage])
}

func TestQueryLabels(t *testing.T) {
	query := NewQuery(v1alpha1.NewJaeger("TestQueryLabels"))
	dep := query.Get()
	assert.Equal(t, "jaeger-operator", dep.Spec.Template.Labels["app.kubernetes.io/managed-by"])
	assert.Equal(t, "query", dep.Spec.Template.Labels["app.kubernetes.io/component"])
	assert.Equal(t, query.jaeger.Name, dep.Spec.Template.Labels["app.kubernetes.io/instance"])
	assert.Equal(t, fmt.Sprintf("%s-query", query.jaeger.Name), dep.Spec.Template.Labels["app.kubernetes.io/name"])
}
