package deployment

import (
	"fmt"
	"strings"
	"testing"

	"github.com/spf13/viper"
	"github.com/stretchr/testify/assert"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"

	v1 "github.com/jaegertracing/jaeger-operator/pkg/apis/jaegertracing/v1"
)

func init() {
	viper.SetDefault("jaeger-version", "1.6")
	viper.SetDefault("jaeger-ingester-image", "jaegertracing/jaeger-ingester")
}

func TestIngesterNotDefined(t *testing.T) {
	jaeger := v1.NewJaeger(types.NamespacedName{Name: "TestIngesterNotDefined"})

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

func TestIngesterNegativeReplicas(t *testing.T) {
	size := int32(-1)
	jaeger := newIngesterJaeger("TestIngesterNegativeReplicas")
	jaeger.Spec.Ingester.Replicas = &size

	ingester := NewIngester(jaeger)
	dep := ingester.Get()
	assert.Equal(t, int32(1), *dep.Spec.Replicas)
}

func TestIngesterDefaultSize(t *testing.T) {
	jaeger := newIngesterJaeger("TestIngesterDefaultSize")

	ingester := NewIngester(jaeger)
	dep := ingester.Get()
	assert.Equal(t, int32(1), *dep.Spec.Replicas)
}

func TestIngesterReplicaSize(t *testing.T) {
	size := int32(0)
	jaeger := newIngesterJaeger("TestIngesterReplicaSize")
	jaeger.Spec.Ingester.Replicas = &size

	ingester := NewIngester(jaeger)
	dep := ingester.Get()
	assert.Equal(t, int32(0), *dep.Spec.Replicas)
}

func TestIngesterSize(t *testing.T) {
	jaeger := newIngesterJaeger("TestIngesterSize")
	jaeger.Spec.Ingester.Size = 2

	ingester := NewIngester(jaeger)
	dep := ingester.Get()
	assert.Equal(t, int32(2), *dep.Spec.Replicas)
}

func TestIngesterReplicaWinsOverSize(t *testing.T) {
	size := int32(3)
	jaeger := newIngesterJaeger("TestIngesterReplicaWinsOverSize")
	jaeger.Spec.Ingester.Size = 2
	jaeger.Spec.Ingester.Replicas = &size

	ingester := NewIngester(jaeger)
	dep := ingester.Get()
	assert.Equal(t, int32(3), *dep.Spec.Replicas)
}

func TestIngesterName(t *testing.T) {
	jaeger := newIngesterJaeger("TestIngesterName")
	ingester := NewIngester(jaeger)
	dep := ingester.Get()
	assert.Equal(t, "TestIngesterName-ingester", dep.ObjectMeta.Name)
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

	envvars := []corev1.EnvVar{
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
	assert.Equal(t, "disabled", dep.Spec.Template.Annotations["linkerd.io/inject"])
}

func TestIngesterLabels(t *testing.T) {
	jaeger := newIngesterJaeger("TestIngesterLabels")
	jaeger.Spec.Labels = map[string]string{
		"name":  "operator",
		"hello": "jaeger",
	}
	jaeger.Spec.Ingester.Labels = map[string]string{
		"hello":   "world", // Override top level annotation
		"another": "false",
	}

	ingester := NewIngester(jaeger)
	dep := ingester.Get()

	assert.Equal(t, "operator", dep.Spec.Template.Labels["name"])
	assert.Equal(t, "world", dep.Spec.Template.Labels["hello"])
	assert.Equal(t, "false", dep.Spec.Template.Labels["another"])
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

	ingesterVolumes := []corev1.Volume{
		{
			Name:         "ingesterVolume",
			VolumeSource: corev1.VolumeSource{},
		},
	}

	ingesterVolumeMounts := []corev1.VolumeMount{
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

	globalVolumes := []corev1.Volume{
		{
			Name:         "globalVolume",
			VolumeSource: corev1.VolumeSource{},
		},
	}

	ingesterVolumeMounts := []corev1.VolumeMount{
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

	globalVolumeMounts := []corev1.VolumeMount{
		{
			Name:     "data",
			ReadOnly: true,
		},
	}

	ingesterVolumeMounts := []corev1.VolumeMount{
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

	globalVolumes := []corev1.Volume{
		{
			Name:         "data",
			VolumeSource: corev1.VolumeSource{HostPath: &corev1.HostPathVolumeSource{Path: "/data1"}},
		},
	}

	ingesterVolumes := []corev1.Volume{
		{
			Name:         "data",
			VolumeSource: corev1.VolumeSource{HostPath: &corev1.HostPathVolumeSource{Path: "/data2"}},
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
	jaeger.Spec.Ingester.Resources = corev1.ResourceRequirements{
		Limits: corev1.ResourceList{
			corev1.ResourceLimitsCPU:    *resource.NewQuantity(2048, resource.BinarySI),
			corev1.ResourceLimitsMemory: *resource.NewQuantity(123, resource.DecimalSI),
		},
		Requests: corev1.ResourceList{
			corev1.ResourceRequestsCPU:    *resource.NewQuantity(2048, resource.BinarySI),
			corev1.ResourceRequestsMemory: *resource.NewQuantity(123, resource.DecimalSI),
		},
	}

	ingester := NewIngester(jaeger)
	dep := ingester.Get()

	assert.Equal(t, *resource.NewQuantity(2048, resource.BinarySI), dep.Spec.Template.Spec.Containers[0].Resources.Limits[corev1.ResourceLimitsCPU])
	assert.Equal(t, *resource.NewQuantity(2048, resource.BinarySI), dep.Spec.Template.Spec.Containers[0].Resources.Requests[corev1.ResourceRequestsCPU])
	assert.Equal(t, *resource.NewQuantity(123, resource.DecimalSI), dep.Spec.Template.Spec.Containers[0].Resources.Limits[corev1.ResourceLimitsMemory])
	assert.Equal(t, *resource.NewQuantity(123, resource.DecimalSI), dep.Spec.Template.Spec.Containers[0].Resources.Requests[corev1.ResourceRequestsMemory])
	assert.Equal(t, *resource.NewQuantity(512, resource.DecimalSI), dep.Spec.Template.Spec.Containers[0].Resources.Limits[corev1.ResourceLimitsEphemeralStorage])
	assert.Equal(t, *resource.NewQuantity(512, resource.DecimalSI), dep.Spec.Template.Spec.Containers[0].Resources.Requests[corev1.ResourceRequestsEphemeralStorage])
}

func TestIngesterWithStorageType(t *testing.T) {
	jaeger := &v1.Jaeger{
		ObjectMeta: metav1.ObjectMeta{
			Name: "TestIngesterStorageType",
		},
		Spec: v1.JaegerSpec{
			Strategy: "streaming",
			Ingester: v1.JaegerIngesterSpec{
				Options: v1.NewOptions(map[string]interface{}{
					"kafka.consumer.topic":   "mytopic",
					"kafka.consumer.brokers": "http://brokers",
				}),
			},
			Storage: v1.JaegerStorageSpec{
				Type: "elasticsearch",
				Options: v1.NewOptions(map[string]interface{}{
					"es.server-urls": "http://somewhere",
				}),
			},
		},
	}
	ingester := NewIngester(jaeger)
	dep := ingester.Get()

	envvars := []corev1.EnvVar{
		{
			Name:  "SPAN_STORAGE_TYPE",
			Value: "elasticsearch",
		},
	}
	assert.Equal(t, envvars, dep.Spec.Template.Spec.Containers[0].Env)
	assert.Len(t, dep.Spec.Template.Spec.Containers[0].Args, 3)
	assert.Equal(t, "--es.server-urls=http://somewhere", dep.Spec.Template.Spec.Containers[0].Args[0])
	assert.Equal(t, "--kafka.consumer.brokers=http://brokers", dep.Spec.Template.Spec.Containers[0].Args[1])
	assert.Equal(t, "--kafka.consumer.topic=mytopic", dep.Spec.Template.Spec.Containers[0].Args[2])
}

func TestIngesterStandardLabels(t *testing.T) {
	ingester := NewIngester(newIngesterJaeger("TestIngesterStandardLabels"))
	dep := ingester.Get()
	assert.Equal(t, "jaeger-operator", dep.Spec.Template.Labels["app.kubernetes.io/managed-by"])
	assert.Equal(t, "ingester", dep.Spec.Template.Labels["app.kubernetes.io/component"])
	assert.Equal(t, ingester.jaeger.Name, dep.Spec.Template.Labels["app.kubernetes.io/instance"])
	assert.Equal(t, fmt.Sprintf("%s-ingester", ingester.jaeger.Name), dep.Spec.Template.Labels["app.kubernetes.io/name"])
}

func TestIngesterOrderOfArguments(t *testing.T) {
	jaeger := newIngesterJaeger("TestIngesterOrderOfArguments")
	jaeger.Spec.Ingester.Options = v1.NewOptions(map[string]interface{}{
		"b-option": "b-value",
		"a-option": "a-value",
		"c-option": "c-value",
	})

	a := NewIngester(jaeger)
	dep := a.Get()

	assert.Len(t, dep.Spec.Template.Spec.Containers, 1)
	assert.Len(t, dep.Spec.Template.Spec.Containers[0].Args, 3)
	assert.True(t, strings.HasPrefix(dep.Spec.Template.Spec.Containers[0].Args[0], "--a-option"))
	assert.True(t, strings.HasPrefix(dep.Spec.Template.Spec.Containers[0].Args[1], "--b-option"))
	assert.True(t, strings.HasPrefix(dep.Spec.Template.Spec.Containers[0].Args[2], "--c-option"))
}

func newIngesterJaeger(name string) *v1.Jaeger {
	return &v1.Jaeger{
		ObjectMeta: metav1.ObjectMeta{
			Name: name,
		},
		Spec: v1.JaegerSpec{
			Strategy: "streaming",
			Ingester: v1.JaegerIngesterSpec{
				Options: v1.NewOptions(map[string]interface{}{
					"any": "option",
				}),
			},
		},
	}
}
