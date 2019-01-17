package deployment

import (
	"testing"

	"github.com/spf13/viper"
	"github.com/stretchr/testify/assert"
	"k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/jaegertracing/jaeger-operator/pkg/apis/io/v1alpha1"
)

func init() {
	viper.SetDefault("jaeger-version", "1.6")
	viper.SetDefault("jaeger-ingester-image", "jaegertracing/jaeger-ingester")
}

func TestIngesterNotDefined(t *testing.T) {
	jaeger := v1alpha1.NewJaeger("TestIngesterNotDefined")

	ingester := NewIngester(jaeger)
	assert.Nil(t, ingester.Get())
}

func TestIngesterNegativeSize(t *testing.T) {
	jaeger := newIngesterJaeger("TestIngesterNegativeSize")
	jaeger.Spec.Ingester.Size = -1

	ingester := NewIngester(jaeger)
	dep := ingester.Get()
	assert.Equal(t, int32(1), *dep.Spec.Replicas)
}

func TestIngesterDefaultSize(t *testing.T) {
	jaeger := newIngesterJaeger("TestIngesterDefaultSize")
	jaeger.Spec.Ingester.Size = 0

	ingester := NewIngester(jaeger)
	dep := ingester.Get()
	assert.Equal(t, int32(1), *dep.Spec.Replicas)
}

func TestIngesterName(t *testing.T) {
	jaeger := newIngesterJaeger("TestIngesterName")
	ingester := NewIngester(jaeger)
	dep := ingester.Get()
	assert.Equal(t, "TestIngesterName-ingester", dep.ObjectMeta.Name)
}

func TestIngesterServices(t *testing.T) {
	jaeger := newIngesterJaeger("TestIngesterServices")
	ingester := NewIngester(jaeger)
	svcs := ingester.Services()
	assert.Len(t, svcs, 1)
}

func TestDefaultIngesterImage(t *testing.T) {
	viper.Set("jaeger-ingester-image", "org/custom-ingester-image")
	viper.Set("jaeger-version", "123")
	defer viper.Reset()

	ingester := NewIngester(newIngesterJaeger("TestDefaultIngesterImage"))
	dep := ingester.Get()

	containers := dep.Spec.Template.Spec.Containers
	assert.Len(t, containers, 1)
	assert.Equal(t, "org/custom-ingester-image:123", containers[0].Image)

	envvars := []v1.EnvVar{
		{
			Name:  "SPAN_STORAGE_TYPE",
			Value: "",
		},
	}
	assert.Equal(t, envvars, containers[0].Env)
}

func TestIngesterAnnotations(t *testing.T) {
	jaeger := newIngesterJaeger("TestIngesterAnnotations")
	jaeger.Spec.Annotations = map[string]string{
		"name":  "operator",
		"hello": "jaeger",
	}
	jaeger.Spec.Ingester.Annotations = map[string]string{
		"hello":                "world", // Override top level annotation
		"prometheus.io/scrape": "false", // Override implicit value
	}

	ingester := NewIngester(jaeger)
	dep := ingester.Get()

	assert.Equal(t, "operator", dep.Spec.Template.Annotations["name"])
	assert.Equal(t, "false", dep.Spec.Template.Annotations["sidecar.istio.io/inject"])
	assert.Equal(t, "world", dep.Spec.Template.Annotations["hello"])
	assert.Equal(t, "false", dep.Spec.Template.Annotations["prometheus.io/scrape"])
}

func TestIngesterSecrets(t *testing.T) {
	jaeger := newIngesterJaeger("TestIngesterSecrets")
	secret := "mysecret"
	jaeger.Spec.Storage.SecretName = secret

	ingester := NewIngester(jaeger)
	dep := ingester.Get()

	assert.Equal(t, "mysecret", dep.Spec.Template.Spec.Containers[0].EnvFrom[0].SecretRef.LocalObjectReference.Name)
}

func TestIngesterVolumeMountsWithVolumes(t *testing.T) {
	name := "TestIngesterVolumeMountsWithVolumes"

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

	ingesterVolumes := []v1.Volume{
		{
			Name:         "ingesterVolume",
			VolumeSource: v1.VolumeSource{},
		},
	}

	ingesterVolumeMounts := []v1.VolumeMount{
		{
			Name: "ingesterVolume",
		},
	}

	jaeger := newIngesterJaeger(name)
	jaeger.Spec.Volumes = globalVolumes
	jaeger.Spec.VolumeMounts = globalVolumeMounts
	jaeger.Spec.Ingester.Volumes = ingesterVolumes
	jaeger.Spec.Ingester.VolumeMounts = ingesterVolumeMounts
	podSpec := NewIngester(jaeger).Get().Spec.Template.Spec

	assert.Len(t, podSpec.Volumes, len(append(ingesterVolumes, globalVolumes...)))
	assert.Len(t, podSpec.Containers[0].VolumeMounts, len(append(ingesterVolumeMounts, globalVolumeMounts...)))

	// ingester is first while global is second
	assert.Equal(t, "ingesterVolume", podSpec.Volumes[0].Name)
	assert.Equal(t, "globalVolume", podSpec.Volumes[1].Name)
	assert.Equal(t, "ingesterVolume", podSpec.Containers[0].VolumeMounts[0].Name)
	assert.Equal(t, "globalVolume", podSpec.Containers[0].VolumeMounts[1].Name)
}

func TestIngesterMountGlobalVolumes(t *testing.T) {
	name := "TestIngesterMountGlobalVolumes"

	globalVolumes := []v1.Volume{
		{
			Name:         "globalVolume",
			VolumeSource: v1.VolumeSource{},
		},
	}

	ingesterVolumeMounts := []v1.VolumeMount{
		{
			Name:     "globalVolume",
			ReadOnly: true,
		},
	}

	jaeger := newIngesterJaeger(name)
	jaeger.Spec.Volumes = globalVolumes
	jaeger.Spec.Ingester.VolumeMounts = ingesterVolumeMounts
	podSpec := NewIngester(jaeger).Get().Spec.Template.Spec

	assert.Len(t, podSpec.Containers[0].VolumeMounts, 1)
	// ingester volume is mounted
	assert.Equal(t, podSpec.Containers[0].VolumeMounts[0].Name, "globalVolume")
}

func TestIngesterVolumeMountsWithSameName(t *testing.T) {
	name := "TestIngesterVolumeMountsWithSameName"

	globalVolumeMounts := []v1.VolumeMount{
		{
			Name:     "data",
			ReadOnly: true,
		},
	}

	ingesterVolumeMounts := []v1.VolumeMount{
		{
			Name:     "data",
			ReadOnly: false,
		},
	}

	jaeger := newIngesterJaeger(name)
	jaeger.Spec.VolumeMounts = globalVolumeMounts
	jaeger.Spec.Ingester.VolumeMounts = ingesterVolumeMounts
	podSpec := NewIngester(jaeger).Get().Spec.Template.Spec

	assert.Len(t, podSpec.Containers[0].VolumeMounts, 1)
	// ingester volume is mounted
	assert.Equal(t, podSpec.Containers[0].VolumeMounts[0].ReadOnly, false)
}

func TestIngesterVolumeWithSameName(t *testing.T) {
	name := "TestIngesterVolumeWithSameName"

	globalVolumes := []v1.Volume{
		{
			Name:         "data",
			VolumeSource: v1.VolumeSource{HostPath: &v1.HostPathVolumeSource{Path: "/data1"}},
		},
	}

	ingesterVolumes := []v1.Volume{
		{
			Name:         "data",
			VolumeSource: v1.VolumeSource{HostPath: &v1.HostPathVolumeSource{Path: "/data2"}},
		},
	}

	jaeger := newIngesterJaeger(name)
	jaeger.Spec.Volumes = globalVolumes
	jaeger.Spec.Ingester.Volumes = ingesterVolumes
	podSpec := NewIngester(jaeger).Get().Spec.Template.Spec

	assert.Len(t, podSpec.Volumes, 1)
	// ingester volume is mounted
	assert.Equal(t, podSpec.Volumes[0].VolumeSource.HostPath.Path, "/data2")
}

func TestIngesterResources(t *testing.T) {
	jaeger := newIngesterJaeger("TestIngesterResources")
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
	jaeger.Spec.Ingester.Resources = v1.ResourceRequirements{
		Limits: v1.ResourceList{
			v1.ResourceLimitsCPU:    *resource.NewQuantity(2048, resource.BinarySI),
			v1.ResourceLimitsMemory: *resource.NewQuantity(123, resource.DecimalSI),
		},
		Requests: v1.ResourceList{
			v1.ResourceRequestsCPU:    *resource.NewQuantity(2048, resource.BinarySI),
			v1.ResourceRequestsMemory: *resource.NewQuantity(123, resource.DecimalSI),
		},
	}

	ingester := NewIngester(jaeger)
	dep := ingester.Get()

	assert.Equal(t, *resource.NewQuantity(2048, resource.BinarySI), dep.Spec.Template.Spec.Containers[0].Resources.Limits[v1.ResourceLimitsCPU])
	assert.Equal(t, *resource.NewQuantity(2048, resource.BinarySI), dep.Spec.Template.Spec.Containers[0].Resources.Requests[v1.ResourceRequestsCPU])
	assert.Equal(t, *resource.NewQuantity(123, resource.DecimalSI), dep.Spec.Template.Spec.Containers[0].Resources.Limits[v1.ResourceLimitsMemory])
	assert.Equal(t, *resource.NewQuantity(123, resource.DecimalSI), dep.Spec.Template.Spec.Containers[0].Resources.Requests[v1.ResourceRequestsMemory])
	assert.Equal(t, *resource.NewQuantity(512, resource.DecimalSI), dep.Spec.Template.Spec.Containers[0].Resources.Limits[v1.ResourceLimitsEphemeralStorage])
	assert.Equal(t, *resource.NewQuantity(512, resource.DecimalSI), dep.Spec.Template.Spec.Containers[0].Resources.Requests[v1.ResourceRequestsEphemeralStorage])
}

func TestIngesterWithStorageType(t *testing.T) {
	jaeger := &v1alpha1.Jaeger{
		ObjectMeta: metav1.ObjectMeta{
			Name: "TestIngesterStorageType",
		},
		Spec: v1alpha1.JaegerSpec{
			Strategy: "streaming",
			Ingester: v1alpha1.JaegerIngesterSpec{
				Options: v1alpha1.NewOptions(map[string]interface{}{
					"kafka.topic": "mytopic",
				}),
			},
			Storage: v1alpha1.JaegerStorageSpec{
				Type: "elasticsearch",
				Options: v1alpha1.NewOptions(map[string]interface{}{
					"kafka.brokers":  "http://brokers",
					"es.server-urls": "http://somewhere",
				}),
			},
		},
	}
	ingester := NewIngester(jaeger)
	dep := ingester.Get()

	envvars := []v1.EnvVar{
		{
			Name:  "SPAN_STORAGE_TYPE",
			Value: "elasticsearch",
		},
	}
	assert.Equal(t, envvars, dep.Spec.Template.Spec.Containers[0].Env)
	assert.Len(t, dep.Spec.Template.Spec.Containers[0].Args, 3)
	assert.Equal(t, "--kafka.topic=mytopic", dep.Spec.Template.Spec.Containers[0].Args[0])
	assert.Equal(t, "--es.server-urls=http://somewhere", dep.Spec.Template.Spec.Containers[0].Args[1])
	assert.Equal(t, "--kafka.brokers=http://brokers", dep.Spec.Template.Spec.Containers[0].Args[2])
}

func newIngesterJaeger(name string) *v1alpha1.Jaeger {
	return &v1alpha1.Jaeger{
		ObjectMeta: metav1.ObjectMeta{
			Name: name,
		},
		Spec: v1alpha1.JaegerSpec{
			Strategy: "streaming",
			Ingester: v1alpha1.JaegerIngesterSpec{
				Options: v1alpha1.NewOptions(map[string]interface{}{
					"any": "option",
				}),
			},
		},
	}
}
