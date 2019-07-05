package deployment

import (
	"fmt"
	"strings"
	"testing"

	"github.com/spf13/viper"
	"github.com/stretchr/testify/assert"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	"k8s.io/apimachinery/pkg/types"

	v1 "github.com/jaegertracing/jaeger-operator/pkg/apis/jaegertracing/v1"
)

func init() {
	viper.SetDefault("jaeger-version", "1.6")
	viper.SetDefault("jaeger-query-image", "jaegertracing/all-in-one")
}

func TestQueryNegativeSize(t *testing.T) {
	jaeger := v1.NewJaeger(types.NamespacedName{Name: "TestQueryNegativeSize"})
	jaeger.Spec.Query.Size = -1

	query := NewQuery(jaeger)
	dep := query.Get()
	assert.Equal(t, int32(1), *dep.Spec.Replicas)
}

func TestQueryNegativeReplicas(t *testing.T) {
	size := int32(-1)
	jaeger := v1.NewJaeger(types.NamespacedName{Name: "TestQueryNegativeReplicas"})
	jaeger.Spec.Query.Replicas = &size

	query := NewQuery(jaeger)
	dep := query.Get()
	assert.Equal(t, int32(1), *dep.Spec.Replicas)
}

func TestQueryDefaultSize(t *testing.T) {
	jaeger := v1.NewJaeger(types.NamespacedName{Name: "TestQueryDefaultSize"})

	query := NewQuery(jaeger)
	dep := query.Get()
	assert.Equal(t, int32(1), *dep.Spec.Replicas)
}

func TestQueryReplicaSize(t *testing.T) {
	size := int32(0)
	jaeger := v1.NewJaeger(types.NamespacedName{Name: "TestQueryReplicaSize"})
	jaeger.Spec.Query.Replicas = &size

	ingester := NewQuery(jaeger)
	dep := ingester.Get()
	assert.Equal(t, int32(0), *dep.Spec.Replicas)
}

func TestQuerySize(t *testing.T) {
	jaeger := v1.NewJaeger(types.NamespacedName{Name: "TestQuerySize"})
	jaeger.Spec.Query.Size = 2

	query := NewQuery(jaeger)
	dep := query.Get()
	assert.Equal(t, int32(2), *dep.Spec.Replicas)
}

func TestQueryReplicaWinsOverSize(t *testing.T) {
	size := int32(3)
	jaeger := v1.NewJaeger(types.NamespacedName{Name: "TestQueryReplicaWinsOverSize"})
	jaeger.Spec.Query.Size = 2
	jaeger.Spec.Query.Replicas = &size

	query := NewQuery(jaeger)
	dep := query.Get()
	assert.Equal(t, int32(3), *dep.Spec.Replicas)
}

func TestDefaultQueryImage(t *testing.T) {
	viper.Set("jaeger-query-image", "org/custom-query-image")
	viper.Set("jaeger-version", "123")
	defer viper.Reset()

	query := NewQuery(v1.NewJaeger(types.NamespacedName{Name: "TestQueryImage"}))
	dep := query.Get()
	containers := dep.Spec.Template.Spec.Containers

	assert.Len(t, containers, 1)
	assert.Equal(t, "org/custom-query-image:123", containers[0].Image)
}

func TestQueryAnnotations(t *testing.T) {
	jaeger := v1.NewJaeger(types.NamespacedName{Name: "TestQueryAnnotations"})
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
	assert.Equal(t, "disabled", dep.Spec.Template.Annotations["linkerd.io/inject"])
}

func TestQueryLabels(t *testing.T) {
	jaeger := v1.NewJaeger(types.NamespacedName{Name: "TestQueryLabels"})
	jaeger.Spec.Labels = map[string]string{
		"name":  "operator",
		"hello": "jaeger",
	}
	jaeger.Spec.Query.Labels = map[string]string{
		"hello":   "world", // Override top level annotation
		"another": "false",
	}

	query := NewQuery(jaeger)
	dep := query.Get()

	assert.Equal(t, "operator", dep.Spec.Template.Labels["name"])
	assert.Equal(t, "world", dep.Spec.Template.Labels["hello"])
	assert.Equal(t, "false", dep.Spec.Template.Labels["another"])
}

func TestQuerySecrets(t *testing.T) {
	jaeger := v1.NewJaeger(types.NamespacedName{Name: "TestQuerySecrets"})
	secret := "mysecret"
	jaeger.Spec.Storage.SecretName = secret

	query := NewQuery(jaeger)
	dep := query.Get()

	assert.Equal(t, "mysecret", dep.Spec.Template.Spec.Containers[0].EnvFrom[0].SecretRef.LocalObjectReference.Name)
}

func TestQueryPodName(t *testing.T) {
	name := "TestQueryPodName"
	query := NewQuery(v1.NewJaeger(types.NamespacedName{Name: name}))
	dep := query.Get()

	assert.Contains(t, dep.ObjectMeta.Name, fmt.Sprintf("%s-query", name))
}

func TestQueryServices(t *testing.T) {
	query := NewQuery(v1.NewJaeger(types.NamespacedName{Name: "TestQueryServices"}))
	svcs := query.Services()

	assert.Len(t, svcs, 1)
}

func TestQueryVolumeMountsWithVolumes(t *testing.T) {
	name := "TestQueryVolumeMountsWithVolumes"

	globalVolumes := []corev1.Volume{
		{
			Name:         "globalVolume",
			VolumeSource: corev1.VolumeSource{},
		},
	}

	globalVolumeMounts := []corev1.VolumeMount{
		{
			Name: "globalVolume",
		},
	}

	queryVolumes := []corev1.Volume{
		{
			Name:         "queryVolume",
			VolumeSource: corev1.VolumeSource{},
		},
	}

	queryVolumeMounts := []corev1.VolumeMount{
		{
			Name: "queryVolume",
		},
	}

	jaeger := v1.NewJaeger(types.NamespacedName{Name: name})
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

	globalVolumes := []corev1.Volume{
		{
			Name:         "globalVolume",
			VolumeSource: corev1.VolumeSource{},
		},
	}

	queryVolumeMounts := []corev1.VolumeMount{
		{
			Name:     "globalVolume",
			ReadOnly: true,
		},
	}

	jaeger := v1.NewJaeger(types.NamespacedName{Name: name})
	jaeger.Spec.Volumes = globalVolumes
	jaeger.Spec.Query.VolumeMounts = queryVolumeMounts
	podSpec := NewQuery(jaeger).Get().Spec.Template.Spec

	assert.Len(t, podSpec.Containers[0].VolumeMounts, 1)
	// query volume is mounted
	assert.Equal(t, podSpec.Containers[0].VolumeMounts[0].Name, "globalVolume")
}

func TestQueryVolumeMountsWithSameName(t *testing.T) {
	name := "TestQueryVolumeMountsWithSameName"

	globalVolumeMounts := []corev1.VolumeMount{
		{
			Name:     "data",
			ReadOnly: true,
		},
	}

	queryVolumeMounts := []corev1.VolumeMount{
		{
			Name:     "data",
			ReadOnly: false,
		},
	}

	jaeger := v1.NewJaeger(types.NamespacedName{Name: name})
	jaeger.Spec.VolumeMounts = globalVolumeMounts
	jaeger.Spec.Query.VolumeMounts = queryVolumeMounts
	podSpec := NewQuery(jaeger).Get().Spec.Template.Spec

	assert.Len(t, podSpec.Containers[0].VolumeMounts, 1)
	// query volume is mounted
	assert.Equal(t, podSpec.Containers[0].VolumeMounts[0].ReadOnly, false)
}

func TestQueryVolumeWithSameName(t *testing.T) {
	name := "TestQueryVolumeWithSameName"

	globalVolumes := []corev1.Volume{
		{
			Name:         "data",
			VolumeSource: corev1.VolumeSource{HostPath: &corev1.HostPathVolumeSource{Path: "/data1"}},
		},
	}

	queryVolumes := []corev1.Volume{
		{
			Name:         "data",
			VolumeSource: corev1.VolumeSource{HostPath: &corev1.HostPathVolumeSource{Path: "/data2"}},
		},
	}

	jaeger := v1.NewJaeger(types.NamespacedName{Name: name})
	jaeger.Spec.Volumes = globalVolumes
	jaeger.Spec.Query.Volumes = queryVolumes
	podSpec := NewQuery(jaeger).Get().Spec.Template.Spec

	assert.Len(t, podSpec.Volumes, 1)
	// query volume is mounted
	assert.Equal(t, podSpec.Volumes[0].VolumeSource.HostPath.Path, "/data2")
}

func TestQueryResources(t *testing.T) {
	jaeger := v1.NewJaeger(types.NamespacedName{Name: "TestQueryResources"})
	jaeger.Spec.Resources = corev1.ResourceRequirements{
		Limits: corev1.ResourceList{
			corev1.ResourceLimitsCPU:              *resource.NewQuantity(1024, resource.BinarySI),
			corev1.ResourceLimitsEphemeralStorage: *resource.NewQuantity(512, resource.DecimalSI),
		},
		Requests: corev1.ResourceList{
			corev1.ResourceRequestsCPU:              *resource.NewQuantity(1024, resource.BinarySI),
			corev1.ResourceRequestsEphemeralStorage: *resource.NewQuantity(512, resource.DecimalSI),
		},
	}
	jaeger.Spec.Query.Resources = corev1.ResourceRequirements{
		Limits: corev1.ResourceList{
			corev1.ResourceLimitsCPU:    *resource.NewQuantity(2048, resource.BinarySI),
			corev1.ResourceLimitsMemory: *resource.NewQuantity(123, resource.DecimalSI),
		},
		Requests: corev1.ResourceList{
			corev1.ResourceRequestsCPU:    *resource.NewQuantity(2048, resource.BinarySI),
			corev1.ResourceRequestsMemory: *resource.NewQuantity(123, resource.DecimalSI),
		},
	}

	query := NewQuery(jaeger)
	dep := query.Get()

	assert.Equal(t, *resource.NewQuantity(2048, resource.BinarySI), dep.Spec.Template.Spec.Containers[0].Resources.Limits[corev1.ResourceLimitsCPU])
	assert.Equal(t, *resource.NewQuantity(2048, resource.BinarySI), dep.Spec.Template.Spec.Containers[0].Resources.Requests[corev1.ResourceRequestsCPU])
	assert.Equal(t, *resource.NewQuantity(123, resource.DecimalSI), dep.Spec.Template.Spec.Containers[0].Resources.Limits[corev1.ResourceLimitsMemory])
	assert.Equal(t, *resource.NewQuantity(123, resource.DecimalSI), dep.Spec.Template.Spec.Containers[0].Resources.Requests[corev1.ResourceRequestsMemory])
	assert.Equal(t, *resource.NewQuantity(512, resource.DecimalSI), dep.Spec.Template.Spec.Containers[0].Resources.Limits[corev1.ResourceLimitsEphemeralStorage])
	assert.Equal(t, *resource.NewQuantity(512, resource.DecimalSI), dep.Spec.Template.Spec.Containers[0].Resources.Requests[corev1.ResourceRequestsEphemeralStorage])
}

func TestQueryStandardLabels(t *testing.T) {
	query := NewQuery(v1.NewJaeger(types.NamespacedName{Name: "TestQueryStandardLabels"}))
	dep := query.Get()
	assert.Equal(t, "jaeger-operator", dep.Spec.Template.Labels["app.kubernetes.io/managed-by"])
	assert.Equal(t, "query", dep.Spec.Template.Labels["app.kubernetes.io/component"])
	assert.Equal(t, query.jaeger.Name, dep.Spec.Template.Labels["app.kubernetes.io/instance"])
	assert.Equal(t, fmt.Sprintf("%s-query", query.jaeger.Name), dep.Spec.Template.Labels["app.kubernetes.io/name"])
}

func TestQueryOrderOfArguments(t *testing.T) {
	jaeger := v1.NewJaeger(types.NamespacedName{Name: "TestQueryOrderOfArguments"})
	jaeger.Spec.Query.Options = v1.NewOptions(map[string]interface{}{
		"b-option": "b-value",
		"a-option": "a-value",
		"c-option": "c-value",
	})

	a := NewQuery(jaeger)
	dep := a.Get()

	assert.Len(t, dep.Spec.Template.Spec.Containers, 1)
	assert.Len(t, dep.Spec.Template.Spec.Containers[0].Args, 3)
	assert.True(t, strings.HasPrefix(dep.Spec.Template.Spec.Containers[0].Args[0], "--a-option"))
	assert.True(t, strings.HasPrefix(dep.Spec.Template.Spec.Containers[0].Args[1], "--b-option"))
	assert.True(t, strings.HasPrefix(dep.Spec.Template.Spec.Containers[0].Args[2], "--c-option"))
}
