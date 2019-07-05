package deployment

import (
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
	viper.SetDefault("jaeger-all-in-one-image", "jaegertracing/all-in-one")
}

func TestDefaultAllInOneImage(t *testing.T) {
	viper.Set("jaeger-all-in-one-image", "org/custom-all-in-one-image")
	viper.Set("jaeger-version", "123")
	defer viper.Reset()

	d := NewAllInOne(v1.NewJaeger(types.NamespacedName{Name: "TestAllInOneDefaultImage"})).Get()

	assert.Len(t, d.Spec.Template.Spec.Containers, 1)
	assert.Equal(t, "org/custom-all-in-one-image:123", d.Spec.Template.Spec.Containers[0].Image)

	envvars := []corev1.EnvVar{
		{
			Name:  "SPAN_STORAGE_TYPE",
			Value: "",
		},
		{
			Name:  "COLLECTOR_ZIPKIN_HTTP_PORT",
			Value: "9411",
		},
	}
	assert.Equal(t, envvars, d.Spec.Template.Spec.Containers[0].Env)
}

func TestAllInOneAnnotations(t *testing.T) {
	jaeger := v1.NewJaeger(types.NamespacedName{Name: "TestAllInOneAnnotations"})
	jaeger.Spec.Annotations = map[string]string{
		"name":  "operator",
		"hello": "jaeger",
	}
	jaeger.Spec.AllInOne.Annotations = map[string]string{
		"hello":                "world", // Override top level annotation
		"prometheus.io/scrape": "false", // Override implicit value
	}

	allinone := NewAllInOne(jaeger)
	dep := allinone.Get()

	assert.Equal(t, "operator", dep.Spec.Template.Annotations["name"])
	assert.Equal(t, "false", dep.Spec.Template.Annotations["sidecar.istio.io/inject"])
	assert.Equal(t, "world", dep.Spec.Template.Annotations["hello"])
	assert.Equal(t, "false", dep.Spec.Template.Annotations["prometheus.io/scrape"])
	assert.Equal(t, "disabled", dep.Spec.Template.Annotations["linkerd.io/inject"])
}

func TestAllInOneLabels(t *testing.T) {
	jaeger := v1.NewJaeger(types.NamespacedName{Name: "TestAllInOneLabels"})
	jaeger.Spec.Labels = map[string]string{
		"name":  "operator",
		"hello": "jaeger",
	}
	jaeger.Spec.AllInOne.Labels = map[string]string{
		"hello":   "world", // Override top level annotation
		"another": "false",
	}

	allinone := NewAllInOne(jaeger)
	dep := allinone.Get()

	assert.Equal(t, "operator", dep.Spec.Template.Labels["name"])
	assert.Equal(t, "world", dep.Spec.Template.Labels["hello"])
	assert.Equal(t, "false", dep.Spec.Template.Labels["another"])
}

func TestAllInOneHasOwner(t *testing.T) {
	name := "TestAllInOneHasOwner"
	a := NewAllInOne(v1.NewJaeger(types.NamespacedName{Name: name}))
	assert.Equal(t, name, a.Get().ObjectMeta.Name)
}

func TestAllInOneNumberOfServices(t *testing.T) {
	name := "TestNumberOfServices"
	services := NewAllInOne(v1.NewJaeger(types.NamespacedName{Name: name})).Services()
	assert.Len(t, services, 4) // collector (headless and cluster IP), query, agent

	for _, svc := range services {
		owners := svc.ObjectMeta.OwnerReferences
		assert.Equal(t, name, owners[0].Name)
	}
}

func TestAllInOneVolumeMountsWithVolumes(t *testing.T) {
	name := "TestAllInOneVolumeMountsWithVolumes"

	globalVolumes := []corev1.Volume{{
		Name:         "globalVolume",
		VolumeSource: corev1.VolumeSource{},
	}}

	globalVolumeMounts := []corev1.VolumeMount{{
		Name: "globalVolume",
	}}

	allInOneVolumes := []corev1.Volume{{
		Name:         "allInOneVolume",
		VolumeSource: corev1.VolumeSource{},
	}}

	allInOneVolumeMounts := []corev1.VolumeMount{{
		Name: "allInOneVolume",
	}}

	jaeger := v1.NewJaeger(types.NamespacedName{Name: name})
	jaeger.Spec.Volumes = globalVolumes
	jaeger.Spec.VolumeMounts = globalVolumeMounts
	jaeger.Spec.AllInOne.Volumes = allInOneVolumes
	jaeger.Spec.AllInOne.VolumeMounts = allInOneVolumeMounts
	podSpec := NewAllInOne(jaeger).Get().Spec.Template.Spec

	// Additional 1 is sampling configmap
	assert.Len(t, podSpec.Volumes, len(append(allInOneVolumes, globalVolumes...))+1)
	assert.Len(t, podSpec.Containers[0].VolumeMounts, len(append(allInOneVolumeMounts, globalVolumeMounts...))+1)

	// AllInOne is first while global is second
	assert.Equal(t, "allInOneVolume", podSpec.Volumes[0].Name)
	assert.Equal(t, "globalVolume", podSpec.Volumes[1].Name)
	assert.Equal(t, "allInOneVolume", podSpec.Containers[0].VolumeMounts[0].Name)
	assert.Equal(t, "globalVolume", podSpec.Containers[0].VolumeMounts[1].Name)
}

func TestAllInOneSecrets(t *testing.T) {
	jaeger := v1.NewJaeger(types.NamespacedName{Name: "TestAllInOneSecrets"})
	secret := "mysecret"
	jaeger.Spec.Storage.SecretName = secret

	allInOne := NewAllInOne(jaeger)
	dep := allInOne.Get()

	assert.Equal(t, "mysecret", dep.Spec.Template.Spec.Containers[0].EnvFrom[0].SecretRef.LocalObjectReference.Name)
}

func TestAllInOneMountGlobalVolumes(t *testing.T) {
	name := "TestAllInOneMountGlobalVolumes"

	globalVolumes := []corev1.Volume{{
		Name:         "globalVolume",
		VolumeSource: corev1.VolumeSource{},
	}}

	allInOneVolumeMounts := []corev1.VolumeMount{{
		Name:     "globalVolume",
		ReadOnly: true,
	}}

	jaeger := v1.NewJaeger(types.NamespacedName{Name: name})
	jaeger.Spec.Volumes = globalVolumes
	jaeger.Spec.AllInOne.VolumeMounts = allInOneVolumeMounts
	podSpec := NewAllInOne(jaeger).Get().Spec.Template.Spec

	// Count includes the sampling configmap
	assert.Len(t, podSpec.Containers[0].VolumeMounts, 2)
	// allInOne volume is mounted
	assert.Equal(t, podSpec.Containers[0].VolumeMounts[0].Name, "globalVolume")
}

func TestAllInOneVolumeMountsWithSameName(t *testing.T) {
	name := "TestAllInOneVolumeMountsWithSameName"

	globalVolumeMounts := []corev1.VolumeMount{{
		Name:     "data",
		ReadOnly: true,
	}}

	allInOneVolumeMounts := []corev1.VolumeMount{{
		Name:     "data",
		ReadOnly: false,
	}}

	jaeger := v1.NewJaeger(types.NamespacedName{Name: name})
	jaeger.Spec.VolumeMounts = globalVolumeMounts
	jaeger.Spec.AllInOne.VolumeMounts = allInOneVolumeMounts
	podSpec := NewAllInOne(jaeger).Get().Spec.Template.Spec

	// Count includes the sampling configmap
	assert.Len(t, podSpec.Containers[0].VolumeMounts, 2)
	// allInOne volume is mounted
	assert.Equal(t, podSpec.Containers[0].VolumeMounts[0].ReadOnly, false)
}

func TestAllInOneVolumeWithSameName(t *testing.T) {
	name := "TestAllInOneVolumeWithSameName"

	globalVolumes := []corev1.Volume{{
		Name:         "data",
		VolumeSource: corev1.VolumeSource{HostPath: &corev1.HostPathVolumeSource{Path: "/data1"}},
	}}

	allInOneVolumes := []corev1.Volume{{
		Name:         "data",
		VolumeSource: corev1.VolumeSource{HostPath: &corev1.HostPathVolumeSource{Path: "/data2"}},
	}}

	jaeger := v1.NewJaeger(types.NamespacedName{Name: name})
	jaeger.Spec.Volumes = globalVolumes
	jaeger.Spec.AllInOne.Volumes = allInOneVolumes
	podSpec := NewAllInOne(jaeger).Get().Spec.Template.Spec

	// Count includes the sampling configmap
	assert.Len(t, podSpec.Volumes, 2)
	// allInOne volume is mounted
	assert.Equal(t, podSpec.Volumes[0].VolumeSource.HostPath.Path, "/data2")
}

func TestAllInOneResources(t *testing.T) {
	jaeger := v1.NewJaeger(types.NamespacedName{Name: "TestAllInOneResources"})
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
	jaeger.Spec.AllInOne.Resources = corev1.ResourceRequirements{
		Limits: corev1.ResourceList{
			corev1.ResourceLimitsCPU:    *resource.NewQuantity(2048, resource.BinarySI),
			corev1.ResourceLimitsMemory: *resource.NewQuantity(123, resource.DecimalSI),
		},
		Requests: corev1.ResourceList{
			corev1.ResourceRequestsCPU:    *resource.NewQuantity(2048, resource.BinarySI),
			corev1.ResourceRequestsMemory: *resource.NewQuantity(123, resource.DecimalSI),
		},
	}

	allinone := NewAllInOne(jaeger)
	dep := allinone.Get()

	assert.Equal(t, *resource.NewQuantity(2048, resource.BinarySI), dep.Spec.Template.Spec.Containers[0].Resources.Limits[corev1.ResourceLimitsCPU])
	assert.Equal(t, *resource.NewQuantity(2048, resource.BinarySI), dep.Spec.Template.Spec.Containers[0].Resources.Requests[corev1.ResourceRequestsCPU])
	assert.Equal(t, *resource.NewQuantity(123, resource.DecimalSI), dep.Spec.Template.Spec.Containers[0].Resources.Limits[corev1.ResourceLimitsMemory])
	assert.Equal(t, *resource.NewQuantity(123, resource.DecimalSI), dep.Spec.Template.Spec.Containers[0].Resources.Requests[corev1.ResourceRequestsMemory])
	assert.Equal(t, *resource.NewQuantity(512, resource.DecimalSI), dep.Spec.Template.Spec.Containers[0].Resources.Limits[corev1.ResourceLimitsEphemeralStorage])
	assert.Equal(t, *resource.NewQuantity(512, resource.DecimalSI), dep.Spec.Template.Spec.Containers[0].Resources.Requests[corev1.ResourceRequestsEphemeralStorage])
}

func TestAllInOneStandardLabels(t *testing.T) {
	a := NewAllInOne(v1.NewJaeger(types.NamespacedName{Name: "TestAllInOneStandardLabels"}))
	dep := a.Get()
	assert.Equal(t, "jaeger-operator", dep.Spec.Template.Labels["app.kubernetes.io/managed-by"])
	assert.Equal(t, "all-in-one", dep.Spec.Template.Labels["app.kubernetes.io/component"])
	assert.Equal(t, a.jaeger.Name, dep.Spec.Template.Labels["app.kubernetes.io/instance"])
	assert.Equal(t, a.jaeger.Name, dep.Spec.Template.Labels["app.kubernetes.io/name"])
}

func TestAllInOneOrderOfArguments(t *testing.T) {
	jaeger := v1.NewJaeger(types.NamespacedName{Name: "TestAllInOneOrderOfArguments"})
	jaeger.Spec.AllInOne.Options = v1.NewOptions(map[string]interface{}{
		"b-option": "b-value",
		"a-option": "a-value",
		"c-option": "c-value",
	})

	a := NewAllInOne(jaeger)
	dep := a.Get()

	assert.Len(t, dep.Spec.Template.Spec.Containers, 1)
	assert.Len(t, dep.Spec.Template.Spec.Containers[0].Args, 4)
	assert.True(t, strings.HasPrefix(dep.Spec.Template.Spec.Containers[0].Args[0], "--a-option"))
	assert.True(t, strings.HasPrefix(dep.Spec.Template.Spec.Containers[0].Args[1], "--b-option"))
	assert.True(t, strings.HasPrefix(dep.Spec.Template.Spec.Containers[0].Args[2], "--c-option"))

	// the following are added automatically
	assert.True(t, strings.HasPrefix(dep.Spec.Template.Spec.Containers[0].Args[3], "--sampling.strategies-file"))
}
