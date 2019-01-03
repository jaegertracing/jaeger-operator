package deployment

import (
	"testing"

	"github.com/spf13/viper"
	"github.com/stretchr/testify/assert"
	"k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"

	"github.com/jaegertracing/jaeger-operator/pkg/apis/io/v1alpha1"
)

func init() {
	viper.SetDefault("jaeger-version", "1.6")
	viper.SetDefault("jaeger-all-in-one-image", "jaegertracing/all-in-one")
}

func TestDefaultAllInOneImage(t *testing.T) {
	viper.Set("jaeger-all-in-one-image", "org/custom-all-in-one-image")
	viper.Set("jaeger-version", "123")
	defer viper.Reset()

	d := NewAllInOne(v1alpha1.NewJaeger("TestAllInOneDefaultImage")).Get()

	assert.Len(t, d.Spec.Template.Spec.Containers, 1)
	assert.Equal(t, "org/custom-all-in-one-image:123", d.Spec.Template.Spec.Containers[0].Image)

	envvars := []v1.EnvVar{
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
	jaeger := v1alpha1.NewJaeger("TestAllInOneAnnotations")
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
}

func TestAllInOneHasOwner(t *testing.T) {
	name := "TestAllInOneHasOwner"
	a := NewAllInOne(v1alpha1.NewJaeger(name))
	assert.Equal(t, name, a.Get().ObjectMeta.Name)
}

func TestAllInOneNumberOfServices(t *testing.T) {
	name := "TestNumberOfServices"
	services := NewAllInOne(v1alpha1.NewJaeger(name)).Services()
	assert.Len(t, services, 3) // collector, query, agent

	for _, svc := range services {
		owners := svc.ObjectMeta.OwnerReferences
		assert.Equal(t, name, owners[0].Name)
	}
}

func TestAllInOneVolumeMountsWithVolumes(t *testing.T) {
	name := "TestAllInOneVolumeMountsWithVolumes"

	globalVolumes := []v1.Volume{
		{
			Name:         "globalVolume",
			VolumeSource: v1.VolumeSource{},
		},
	}

	globalVolumeMounts := []v1.VolumeMount{
		{
			Name: "globalVolume",
		},
	}

	allInOneVolumes := []v1.Volume{
		{
			Name:         "allInOneVolume",
			VolumeSource: v1.VolumeSource{},
		},
	}

	allInOneVolumeMounts := []v1.VolumeMount{
		{
			Name: "allInOneVolume",
		},
	}

	jaeger := v1alpha1.NewJaeger(name)
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
	jaeger := v1alpha1.NewJaeger("TestAllInOneSecrets")
	secret := "mysecret"
	jaeger.Spec.Storage.SecretName = secret

	allInOne := NewAllInOne(jaeger)
	dep := allInOne.Get()

	assert.Equal(t, "mysecret", dep.Spec.Template.Spec.Containers[0].EnvFrom[0].SecretRef.LocalObjectReference.Name)
}

func TestAllInOneMountGlobalVolumes(t *testing.T) {
	name := "TestAllInOneMountGlobalVolumes"

	globalVolumes := []v1.Volume{
		{
			Name:         "globalVolume",
			VolumeSource: v1.VolumeSource{},
		},
	}

	allInOneVolumeMounts := []v1.VolumeMount{
		{
			Name:     "globalVolume",
			ReadOnly: true,
		},
	}

	jaeger := v1alpha1.NewJaeger(name)
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

	globalVolumeMounts := []v1.VolumeMount{
		{
			Name:     "data",
			ReadOnly: true,
		},
	}

	allInOneVolumeMounts := []v1.VolumeMount{
		{
			Name:     "data",
			ReadOnly: false,
		},
	}

	jaeger := v1alpha1.NewJaeger(name)
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

	globalVolumes := []v1.Volume{
		{
			Name:         "data",
			VolumeSource: v1.VolumeSource{HostPath: &v1.HostPathVolumeSource{Path: "/data1"}},
		},
	}

	allInOneVolumes := []v1.Volume{
		{
			Name:         "data",
			VolumeSource: v1.VolumeSource{HostPath: &v1.HostPathVolumeSource{Path: "/data2"}},
		},
	}

	jaeger := v1alpha1.NewJaeger(name)
	jaeger.Spec.Volumes = globalVolumes
	jaeger.Spec.AllInOne.Volumes = allInOneVolumes
	podSpec := NewAllInOne(jaeger).Get().Spec.Template.Spec

	// Count includes the sampling configmap
	assert.Len(t, podSpec.Volumes, 2)
	// allInOne volume is mounted
	assert.Equal(t, podSpec.Volumes[0].VolumeSource.HostPath.Path, "/data2")
}

func TestAllInOneResources(t *testing.T) {
	jaeger := v1alpha1.NewJaeger("TestAllInOneResources")
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
	jaeger.Spec.AllInOne.Resources = v1.ResourceRequirements{
		Limits: v1.ResourceList{
			v1.ResourceLimitsCPU:    *resource.NewQuantity(2048, resource.BinarySI),
			v1.ResourceLimitsMemory: *resource.NewQuantity(123, resource.DecimalSI),
		},
		Requests: v1.ResourceList{
			v1.ResourceRequestsCPU:    *resource.NewQuantity(2048, resource.BinarySI),
			v1.ResourceRequestsMemory: *resource.NewQuantity(123, resource.DecimalSI),
		},
	}

	allinone := NewAllInOne(jaeger)
	dep := allinone.Get()

	assert.Equal(t, *resource.NewQuantity(2048, resource.BinarySI), dep.Spec.Template.Spec.Containers[0].Resources.Limits[v1.ResourceLimitsCPU])
	assert.Equal(t, *resource.NewQuantity(2048, resource.BinarySI), dep.Spec.Template.Spec.Containers[0].Resources.Requests[v1.ResourceRequestsCPU])
	assert.Equal(t, *resource.NewQuantity(123, resource.DecimalSI), dep.Spec.Template.Spec.Containers[0].Resources.Limits[v1.ResourceLimitsMemory])
	assert.Equal(t, *resource.NewQuantity(123, resource.DecimalSI), dep.Spec.Template.Spec.Containers[0].Resources.Requests[v1.ResourceRequestsMemory])
	assert.Equal(t, *resource.NewQuantity(512, resource.DecimalSI), dep.Spec.Template.Spec.Containers[0].Resources.Limits[v1.ResourceLimitsEphemeralStorage])
	assert.Equal(t, *resource.NewQuantity(512, resource.DecimalSI), dep.Spec.Template.Spec.Containers[0].Resources.Requests[v1.ResourceRequestsEphemeralStorage])
}
