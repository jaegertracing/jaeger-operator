package deployment

import (
	"fmt"
	"strings"
	"testing"

	"github.com/jaegertracing/jaeger-operator/pkg/version"

	"github.com/spf13/viper"
	"github.com/stretchr/testify/assert"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"

	v1 "github.com/jaegertracing/jaeger-operator/pkg/apis/jaegertracing/v1"
	"github.com/jaegertracing/jaeger-operator/pkg/util"
)

func init() {
	viper.SetDefault("jaeger-collector-image", "jaegertracing/all-in-one")
}

func TestNegativeReplicas(t *testing.T) {
	size := int32(-1)
	jaeger := v1.NewJaeger(types.NamespacedName{Name: "my-instance"})
	jaeger.Spec.Collector.Replicas = &size

	collector := NewCollector(jaeger)
	dep := collector.Get()
	assert.Equal(t, size, *dep.Spec.Replicas)
}

func TestDefaultSize(t *testing.T) {
	jaeger := v1.NewJaeger(types.NamespacedName{Name: "my-instance"})

	collector := NewCollector(jaeger)
	dep := collector.Get()
	assert.Nil(t, dep.Spec.Replicas) // we let Kubernetes define the default
}

func TestReplicaSize(t *testing.T) {
	size := int32(0)
	jaeger := v1.NewJaeger(types.NamespacedName{Name: "my-instance"})
	jaeger.Spec.Collector.Replicas = &size

	collector := NewCollector(jaeger)
	dep := collector.Get()
	assert.Equal(t, int32(0), *dep.Spec.Replicas)
}

func TestName(t *testing.T) {
	collector := NewCollector(v1.NewJaeger(types.NamespacedName{Name: "my-instance"}))
	dep := collector.Get()
	assert.Equal(t, "my-instance-collector", dep.ObjectMeta.Name)
}

func TestCollectorServices(t *testing.T) {
	collector := NewCollector(v1.NewJaeger(types.NamespacedName{Name: "my-instance"}))
	svcs := collector.Services()
	assert.Len(t, svcs, 2) // headless and cluster IP
}

func TestDefaultCollectorImage(t *testing.T) {
	viper.Set("jaeger-collector-image", "org/custom-collector-image")
	defer viper.Reset()

	jaeger := v1.NewJaeger(types.NamespacedName{Name: "my-instance"})
	collector := NewCollector(jaeger)
	dep := collector.Get()

	containers := dep.Spec.Template.Spec.Containers
	assert.Len(t, containers, 1)
	assert.Empty(t, jaeger.Spec.Collector.Image)
	assert.Equal(t, "org/custom-collector-image:"+version.Get().Jaeger, containers[0].Image)

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
	jaeger := v1.NewJaeger(types.NamespacedName{Name: "my-instance"})
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
	jaeger := v1.NewJaeger(types.NamespacedName{Name: "my-instance"})
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
	jaeger := v1.NewJaeger(types.NamespacedName{Name: "my-instance"})
	secret := "mysecret"
	jaeger.Spec.Storage.SecretName = secret

	collector := NewCollector(jaeger)
	dep := collector.Get()

	assert.Equal(t, "mysecret", dep.Spec.Template.Spec.Containers[0].EnvFrom[0].SecretRef.LocalObjectReference.Name)
}

func TestCollectorVolumeMountsWithVolumes(t *testing.T) {
	name := "my-instance"

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
	name := "my-instance"

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
	name := "my-instance"

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
	name := "my-instance"

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
	jaeger := v1.NewJaeger(types.NamespacedName{Name: "my-instance"})
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
	c := NewCollector(v1.NewJaeger(types.NamespacedName{Name: "my-instance"}))
	dep := c.Get()
	assert.Equal(t, "jaeger-operator", dep.Spec.Template.Labels["app.kubernetes.io/managed-by"])
	assert.Equal(t, "collector", dep.Spec.Template.Labels["app.kubernetes.io/component"])
	assert.Equal(t, c.jaeger.Name, dep.Spec.Template.Labels["app.kubernetes.io/instance"])
	assert.Equal(t, fmt.Sprintf("%s-collector", c.jaeger.Name), dep.Spec.Template.Labels["app.kubernetes.io/name"])
}

func TestCollectorWithDirectStorageType(t *testing.T) {
	jaeger := &v1.Jaeger{
		ObjectMeta: metav1.ObjectMeta{
			Name: "my-instance",
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
			Name: "my-instance",
		},
		Spec: v1.JaegerSpec{
			Strategy: v1.DeploymentStrategyStreaming,
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
			Name: "my-instance",
		},
		Spec: v1.JaegerSpec{
			Strategy: v1.DeploymentStrategyStreaming,
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
	jaeger := v1.NewJaeger(types.NamespacedName{Name: "my-instance"})
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

func TestCollectorAutoscalersOnByDefault(t *testing.T) {
	// prepare
	jaeger := v1.NewJaeger(types.NamespacedName{Name: "my-instance"})
	c := NewCollector(jaeger)

	// test
	a := c.Autoscalers()

	// verify
	assert.Len(t, a, 1)
	assert.Len(t, a[0].Spec.Metrics, 2)

	assert.Contains(t, []corev1.ResourceName{a[0].Spec.Metrics[0].Resource.Name, a[0].Spec.Metrics[1].Resource.Name}, corev1.ResourceCPU)
	assert.Contains(t, []corev1.ResourceName{a[0].Spec.Metrics[0].Resource.Name, a[0].Spec.Metrics[1].Resource.Name}, corev1.ResourceMemory)

	assert.Equal(t, int32(90), *a[0].Spec.Metrics[0].Resource.Target.AverageUtilization)
	assert.Equal(t, int32(90), *a[0].Spec.Metrics[1].Resource.Target.AverageUtilization)
}

func TestCollectorAutoscalersDisabledByExplicitReplicaSize(t *testing.T) {
	// prepare
	tests := []int32{int32(0), int32(1)}

	for _, test := range tests {
		jaeger := v1.NewJaeger(types.NamespacedName{Name: "my-instance"})
		jaeger.Spec.Collector.Replicas = &test
		c := NewCollector(jaeger)

		// test
		a := c.Autoscalers()

		// verify
		assert.Len(t, a, 0)
	}
}

func TestCollectorAutoscalersDisabledByExplicitOption(t *testing.T) {
	// prepare
	disabled := false
	jaeger := v1.NewJaeger(types.NamespacedName{Name: "my-instance"})
	jaeger.Spec.Collector.Autoscale = &disabled
	c := NewCollector(jaeger)

	// test
	a := c.Autoscalers()

	// verify
	assert.Len(t, a, 0)
}

func TestCollectorAutoscalersSetMaxReplicas(t *testing.T) {
	// prepare
	maxReplicas := int32(2)
	jaeger := v1.NewJaeger(types.NamespacedName{Name: "my-instance"})
	jaeger.Spec.Collector.MaxReplicas = &maxReplicas
	c := NewCollector(jaeger)

	// test
	a := c.Autoscalers()

	// verify
	assert.Len(t, a, 1)
	assert.Equal(t, maxReplicas, a[0].Spec.MaxReplicas)
}

func TestCollectoArgumentsOpenshiftTLS(t *testing.T) {
	viper.Set("platform", v1.FlagPlatformOpenShift)
	defer viper.Reset()

	jaeger := v1.NewJaeger(types.NamespacedName{Name: "my-instance"})
	jaeger.Spec.Collector.Options = v1.NewOptions(map[string]interface{}{
		"a-option": "a-value",
	})

	a := NewCollector(jaeger)
	dep := a.Get()

	assert.Len(t, dep.Spec.Template.Spec.Containers, 1)
	assert.Len(t, dep.Spec.Template.Spec.Containers[0].Args, 5)
	assert.Greater(t, len(util.FindItem("--a-option=a-value", dep.Spec.Template.Spec.Containers[0].Args)), 0)

	// the following are added automatically
	assert.Greater(t, len(util.FindItem("--collector.grpc.tls.enabled=true", dep.Spec.Template.Spec.Containers[0].Args)), 0)
	assert.Greater(t, len(util.FindItem("--collector.grpc.tls.cert=/etc/tls-config/tls.crt", dep.Spec.Template.Spec.Containers[0].Args)), 0)
	assert.Greater(t, len(util.FindItem("--collector.grpc.tls.key=/etc/tls-config/tls.key", dep.Spec.Template.Spec.Containers[0].Args)), 0)
	assert.Greater(t, len(util.FindItem("--sampling.strategies-file", dep.Spec.Template.Spec.Containers[0].Args)), 0)
}

func TestCollectorOTELConfig(t *testing.T) {
	jaeger := v1.NewJaeger(types.NamespacedName{Name: "instance"})
	jaeger.Spec.Collector.Config = v1.NewFreeForm(map[string]interface{}{"foo": "bar"})

	c := NewCollector(jaeger)
	d := c.Get()
	assert.True(t, hasArgument("--config=/etc/jaeger/otel/config.yaml", d.Spec.Template.Spec.Containers[0].Args))
	assert.True(t, hasVolume("instance-collector-otel-config", d.Spec.Template.Spec.Volumes))
	assert.True(t, hasVolumeMount("instance-collector-otel-config", d.Spec.Template.Spec.Containers[0].VolumeMounts))
}

func TestCollectorServiceLinks(t *testing.T) {
	c := NewCollector(v1.NewJaeger(types.NamespacedName{Name: "my-instance"}))
	dep := c.Get()
	falseVar := false
	assert.Equal(t, &falseVar, dep.Spec.Template.Spec.EnableServiceLinks)
}

func hasVolume(name string, volumes []corev1.Volume) bool {
	for _, v := range volumes {
		if v.Name == name {
			return true
		}
	}
	return false
}

func hasVolumeMount(name string, volumeMounts []corev1.VolumeMount) bool {
	for _, v := range volumeMounts {
		if v.Name == name {
			return true
		}
	}
	return false
}

func hasArgument(arg string, args []string) bool {
	for _, v := range args {
		if v == arg {
			return true
		}
	}
	return false
}
