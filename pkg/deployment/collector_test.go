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
	viper.SetDefault("jaeger-collector-image", "jaegertracing/all-in-one")
}

func TestNegativeSize(t *testing.T) {
	jaeger := v1.NewJaeger(types.NamespacedName{Name: "TestNegativeSize"})
	jaeger.Spec.Collector.Size = -1

	collector := NewCollector(jaeger)
	dep := collector.Get()
	assert.Equal(t, int32(1), *dep.Spec.Replicas)
}

func TestNegativeReplicas(t *testing.T) {
	size := int32(-1)
	jaeger := v1.NewJaeger(types.NamespacedName{Name: "TestNegativeReplicas"})
	jaeger.Spec.Collector.Replicas = &size

	collector := NewCollector(jaeger)
	dep := collector.Get()
	assert.Equal(t, int32(1), *dep.Spec.Replicas)
}

func TestDefaultSize(t *testing.T) {
	jaeger := v1.NewJaeger(types.NamespacedName{Name: "TestDefaultSize"})

	collector := NewCollector(jaeger)
	dep := collector.Get()
	assert.Equal(t, int32(1), *dep.Spec.Replicas)
}

func TestReplicaSize(t *testing.T) {
	size := int32(0)
	jaeger := v1.NewJaeger(types.NamespacedName{Name: "TestReplicaSize"})
	jaeger.Spec.Collector.Replicas = &size

	collector := NewCollector(jaeger)
	dep := collector.Get()
	assert.Equal(t, int32(0), *dep.Spec.Replicas)
}

func TestSize(t *testing.T) {
	jaeger := v1.NewJaeger(types.NamespacedName{Name: "TestSize"})
	jaeger.Spec.Collector.Size = 2

	collector := NewCollector(jaeger)
	dep := collector.Get()
	assert.Equal(t, int32(2), *dep.Spec.Replicas)
}

func TestReplicaWinsOverSize(t *testing.T) {
	size := int32(3)
	jaeger := v1.NewJaeger(types.NamespacedName{Name: "TestReplicaWinsOverSize"})
	jaeger.Spec.Collector.Size = 2
	jaeger.Spec.Collector.Replicas = &size

	collector := NewCollector(jaeger)
	dep := collector.Get()
	assert.Equal(t, int32(3), *dep.Spec.Replicas)
}

func TestName(t *testing.T) {
	collector := NewCollector(v1.NewJaeger(types.NamespacedName{Name: "TestName"}))
	dep := collector.Get()
	assert.Equal(t, "TestName-collector", dep.ObjectMeta.Name)
}

func TestCollectorServices(t *testing.T) {
	collector := NewCollector(v1.NewJaeger(types.NamespacedName{Name: "TestName"}))
	svcs := collector.Services()
	assert.Len(t, svcs, 2) // headless and cluster IP
}

func TestDefaultCollectorImage(t *testing.T) {
	viper.Set("jaeger-collector-image", "org/custom-collector-image")
	viper.Set("jaeger-version", "123")
	defer viper.Reset()

	collector := NewCollector(v1.NewJaeger(types.NamespacedName{Name: "TestCollectorImage"}))
	dep := collector.Get()

	containers := dep.Spec.Template.Spec.Containers
	assert.Len(t, containers, 1)
	assert.Equal(t, "org/custom-collector-image:123", containers[0].Image)

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
	assert.Equal(t, envvars, containers[0].Env)
}

func TestCollectorAnnotations(t *testing.T) {
	jaeger := v1.NewJaeger(types.NamespacedName{Name: "TestCollectorAnnotations"})
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
	assert.Equal(t, "disabled", dep.Spec.Template.Annotations["linkerd.io/inject"])
}

func TestCollectorLabels(t *testing.T) {
	jaeger := v1.NewJaeger(types.NamespacedName{Name: "TestCollectorLabels"})
	jaeger.Spec.Labels = map[string]string{
		"name":  "operator",
		"hello": "jaeger",
	}
	jaeger.Spec.Collector.Labels = map[string]string{
		"hello":   "world", // Override top level annotation
		"another": "false",
	}

	collector := NewCollector(jaeger)
	dep := collector.Get()

	assert.Equal(t, "operator", dep.Spec.Template.Labels["name"])
	assert.Equal(t, "world", dep.Spec.Template.Labels["hello"])
	assert.Equal(t, "false", dep.Spec.Template.Labels["another"])
}

func TestCollectorSecrets(t *testing.T) {
	jaeger := v1.NewJaeger(types.NamespacedName{Name: "TestCollectorSecrets"})
	secret := "mysecret"
	jaeger.Spec.Storage.SecretName = secret

	collector := NewCollector(jaeger)
	dep := collector.Get()

	assert.Equal(t, "mysecret", dep.Spec.Template.Spec.Containers[0].EnvFrom[0].SecretRef.LocalObjectReference.Name)
}

func TestCollectorVolumeMountsWithVolumes(t *testing.T) {
	name := "TestCollectorVolumeMountsWithVolumes"

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

	collectorVolumes := []corev1.Volume{
		{
			Name:         "collectorVolume",
			VolumeSource: corev1.VolumeSource{},
		},
	}

	collectorVolumeMounts := []corev1.VolumeMount{
		{
			Name: "collectorVolume",
		},
	}

	jaeger := v1.NewJaeger(types.NamespacedName{Name: name})
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

	globalVolumes := []corev1.Volume{
		{
			Name:         "globalVolume",
			VolumeSource: corev1.VolumeSource{},
		},
	}

	collectorVolumeMounts := []corev1.VolumeMount{
		{
			Name:     "globalVolume",
			ReadOnly: true,
		},
	}

	jaeger := v1.NewJaeger(types.NamespacedName{Name: name})
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

	globalVolumeMounts := []corev1.VolumeMount{
		{
			Name:     "data",
			ReadOnly: true,
		},
	}

	collectorVolumeMounts := []corev1.VolumeMount{
		{
			Name:     "data",
			ReadOnly: false,
		},
	}

	jaeger := v1.NewJaeger(types.NamespacedName{Name: name})
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

	globalVolumes := []corev1.Volume{
		{
			Name:         "data",
			VolumeSource: corev1.VolumeSource{HostPath: &corev1.HostPathVolumeSource{Path: "/data1"}},
		},
	}

	collectorVolumes := []corev1.Volume{
		{
			Name:         "data",
			VolumeSource: corev1.VolumeSource{HostPath: &corev1.HostPathVolumeSource{Path: "/data2"}},
		},
	}

	jaeger := v1.NewJaeger(types.NamespacedName{Name: name})
	jaeger.Spec.Volumes = globalVolumes
	jaeger.Spec.Collector.Volumes = collectorVolumes
	podSpec := NewCollector(jaeger).Get().Spec.Template.Spec

	// Count includes the sampling configmap
	assert.Len(t, podSpec.Volumes, 2)
	// collector volume is mounted
	assert.Equal(t, podSpec.Volumes[0].VolumeSource.HostPath.Path, "/data2")
}

func TestCollectorResources(t *testing.T) {
	jaeger := v1.NewJaeger(types.NamespacedName{Name: "TestCollectorResources"})
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
	jaeger.Spec.Collector.Resources = corev1.ResourceRequirements{
		Limits: corev1.ResourceList{
			corev1.ResourceLimitsCPU:    *resource.NewQuantity(2048, resource.BinarySI),
			corev1.ResourceLimitsMemory: *resource.NewQuantity(123, resource.DecimalSI),
		},
		Requests: corev1.ResourceList{
			corev1.ResourceRequestsCPU:    *resource.NewQuantity(2048, resource.BinarySI),
			corev1.ResourceRequestsMemory: *resource.NewQuantity(123, resource.DecimalSI),
		},
	}

	collector := NewCollector(jaeger)
	dep := collector.Get()

	assert.Equal(t, *resource.NewQuantity(2048, resource.BinarySI), dep.Spec.Template.Spec.Containers[0].Resources.Limits[corev1.ResourceLimitsCPU])
	assert.Equal(t, *resource.NewQuantity(2048, resource.BinarySI), dep.Spec.Template.Spec.Containers[0].Resources.Requests[corev1.ResourceRequestsCPU])
	assert.Equal(t, *resource.NewQuantity(123, resource.DecimalSI), dep.Spec.Template.Spec.Containers[0].Resources.Limits[corev1.ResourceLimitsMemory])
	assert.Equal(t, *resource.NewQuantity(123, resource.DecimalSI), dep.Spec.Template.Spec.Containers[0].Resources.Requests[corev1.ResourceRequestsMemory])
	assert.Equal(t, *resource.NewQuantity(512, resource.DecimalSI), dep.Spec.Template.Spec.Containers[0].Resources.Limits[corev1.ResourceLimitsEphemeralStorage])
	assert.Equal(t, *resource.NewQuantity(512, resource.DecimalSI), dep.Spec.Template.Spec.Containers[0].Resources.Requests[corev1.ResourceRequestsEphemeralStorage])
}

func TestCollectorStandardLabels(t *testing.T) {
	c := NewCollector(v1.NewJaeger(types.NamespacedName{Name: "TestCollectorStandardLabels"}))
	dep := c.Get()
	assert.Equal(t, "jaeger-operator", dep.Spec.Template.Labels["app.kubernetes.io/managed-by"])
	assert.Equal(t, "collector", dep.Spec.Template.Labels["app.kubernetes.io/component"])
	assert.Equal(t, c.jaeger.Name, dep.Spec.Template.Labels["app.kubernetes.io/instance"])
	assert.Equal(t, fmt.Sprintf("%s-collector", c.jaeger.Name), dep.Spec.Template.Labels["app.kubernetes.io/name"])
}

func TestCollectorWithDirectStorageType(t *testing.T) {
	jaeger := &v1.Jaeger{
		ObjectMeta: metav1.ObjectMeta{
			Name: "TestCollectorWithDirectStorageType",
		},
		Spec: v1.JaegerSpec{
			Storage: v1.JaegerStorageSpec{
				Type: "elasticsearch",
				Options: v1.NewOptions(map[string]interface{}{
					"es.server-urls": "http://somewhere",
				}),
			},
		},
	}
	collector := NewCollector(jaeger)
	dep := collector.Get()

	envvars := []corev1.EnvVar{
		{
			Name:  "SPAN_STORAGE_TYPE",
			Value: "elasticsearch",
		},
		{
			Name:  "COLLECTOR_ZIPKIN_HTTP_PORT",
			Value: "9411",
		},
	}
	assert.Equal(t, envvars, dep.Spec.Template.Spec.Containers[0].Env)
	assert.Len(t, dep.Spec.Template.Spec.Containers[0].Args, 2)
	assert.Equal(t, "--es.server-urls=http://somewhere", dep.Spec.Template.Spec.Containers[0].Args[0])
}

func TestCollectorWithKafkaStorageType(t *testing.T) {
	jaeger := &v1.Jaeger{
		ObjectMeta: metav1.ObjectMeta{
			Name: "TestCollectorWithIngesterStorageType",
		},
		Spec: v1.JaegerSpec{
			Strategy: "streaming",
			Collector: v1.JaegerCollectorSpec{
				Options: v1.NewOptions(map[string]interface{}{
					"kafka.producer.topic": "mytopic",
				}),
			},
			Storage: v1.JaegerStorageSpec{
				Type: "elasticsearch",
				Options: v1.NewOptions(map[string]interface{}{
					"kafka.producer.brokers": "http://brokers",
					"es.server-urls":         "http://somewhere",
				}),
			},
		},
	}
	collector := NewCollector(jaeger)
	dep := collector.Get()

	envvars := []corev1.EnvVar{
		{
			Name:  "SPAN_STORAGE_TYPE",
			Value: "kafka",
		},
		{
			Name:  "COLLECTOR_ZIPKIN_HTTP_PORT",
			Value: "9411",
		},
	}
	assert.Equal(t, envvars, dep.Spec.Template.Spec.Containers[0].Env)
	assert.Len(t, dep.Spec.Template.Spec.Containers[0].Args, 3)
	assert.Equal(t, "--kafka.producer.brokers=http://brokers", dep.Spec.Template.Spec.Containers[0].Args[0])
	assert.Equal(t, "--kafka.producer.topic=mytopic", dep.Spec.Template.Spec.Containers[0].Args[1])
}

func TestCollectorWithIngesterNoOptionsStorageType(t *testing.T) {
	jaeger := &v1.Jaeger{
		ObjectMeta: metav1.ObjectMeta{
			Name: "TestCollectorWithIngesterNoOptionsStorageType",
		},
		Spec: v1.JaegerSpec{
			Strategy: "streaming",
			Storage: v1.JaegerStorageSpec{
				Type: "elasticsearch",
				Options: v1.NewOptions(map[string]interface{}{
					"kafka.brokers":  "http://brokers",
					"es.server-urls": "http://somewhere",
				}),
			},
		},
	}
	collector := NewCollector(jaeger)
	dep := collector.Get()

	envvars := []corev1.EnvVar{
		{
			Name:  "SPAN_STORAGE_TYPE",
			Value: "kafka",
		},
		{
			Name:  "COLLECTOR_ZIPKIN_HTTP_PORT",
			Value: "9411",
		},
	}
	assert.Equal(t, envvars, dep.Spec.Template.Spec.Containers[0].Env)
	assert.Len(t, dep.Spec.Template.Spec.Containers[0].Args, 2)
	assert.Equal(t, "--kafka.brokers=http://brokers", dep.Spec.Template.Spec.Containers[0].Args[0])
}

func TestCollectorOrderOfArguments(t *testing.T) {
	jaeger := v1.NewJaeger(types.NamespacedName{Name: "TestCollectorOrderOfArguments"})
	jaeger.Spec.Collector.Options = v1.NewOptions(map[string]interface{}{
		"b-option": "b-value",
		"a-option": "a-value",
		"c-option": "c-value",
	})

	a := NewCollector(jaeger)
	dep := a.Get()

	assert.Len(t, dep.Spec.Template.Spec.Containers, 1)
	assert.Len(t, dep.Spec.Template.Spec.Containers[0].Args, 4)
	assert.True(t, strings.HasPrefix(dep.Spec.Template.Spec.Containers[0].Args[0], "--a-option"))
	assert.True(t, strings.HasPrefix(dep.Spec.Template.Spec.Containers[0].Args[1], "--b-option"))
	assert.True(t, strings.HasPrefix(dep.Spec.Template.Spec.Containers[0].Args[2], "--c-option"))

	// the following are added automatically
	assert.True(t, strings.HasPrefix(dep.Spec.Template.Spec.Containers[0].Args[3], "--sampling.strategies-file"))
}
