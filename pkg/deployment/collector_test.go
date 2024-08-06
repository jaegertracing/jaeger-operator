package deployment

import (
	"fmt"
	"strings"
	"testing"

	"github.com/spf13/viper"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	appsv1 "k8s.io/api/apps/v1"
	autoscalingv2 "k8s.io/api/autoscaling/v2"
	autoscalingv2beta2 "k8s.io/api/autoscaling/v2beta2"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/intstr"

	v1 "github.com/jaegertracing/jaeger-operator/apis/v1"
	"github.com/jaegertracing/jaeger-operator/pkg/autodetect"
	"github.com/jaegertracing/jaeger-operator/pkg/util"
	"github.com/jaegertracing/jaeger-operator/pkg/version"
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
			Name:  "COLLECTOR_ZIPKIN_HOST_PORT",
			Value: ":9411",
		},
		{
			Name:  "COLLECTOR_OTLP_ENABLED",
			Value: "true",
		},
		{
			Name:  "COLLECTOR_OTLP_GRPC_HOST_PORT",
			Value: "0.0.0.0:4317",
		},
		{
			Name:  "COLLECTOR_OTLP_HTTP_HOST_PORT",
			Value: "0.0.0.0:4318",
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
	assert.Equal(t, "operator", dep.Spec.Selector.MatchLabels["name"])
	assert.Equal(t, "world", dep.Spec.Selector.MatchLabels["hello"])
	assert.Equal(t, "false", dep.Spec.Selector.MatchLabels["another"])
}

func TestCollectorSecrets(t *testing.T) {
	jaeger := v1.NewJaeger(types.NamespacedName{Name: "my-instance"})
	secret := "mysecret"
	jaeger.Spec.Storage.SecretName = secret

	collector := NewCollector(jaeger)
	dep := collector.Get()

	assert.Equal(t, "mysecret", dep.Spec.Template.Spec.Containers[0].EnvFrom[0].SecretRef.LocalObjectReference.Name)
}

func TestCollectorImagePullSecrets(t *testing.T) {
	jaeger := v1.NewJaeger(types.NamespacedName{Name: "TestAllInOneImagePullSecrets"})
	const pullSecret = "mysecret"
	jaeger.Spec.ImagePullSecrets = []corev1.LocalObjectReference{
		{
			Name: pullSecret,
		},
	}

	collector := NewCollector(jaeger)
	dep := collector.Get()

	assert.Equal(t, pullSecret, dep.Spec.Template.Spec.ImagePullSecrets[0].Name)
}

func TestCollectorKafkaSecrets(t *testing.T) {
	jaeger := v1.NewJaeger(types.NamespacedName{Name: "kafka-instance"})
	secret := "mysecret"
	jaeger.Spec.Strategy = v1.DeploymentStrategyStreaming
	jaeger.Spec.Collector.KafkaSecretName = secret

	collector := NewCollector(jaeger)
	dep := collector.Get()

	assert.Equal(t, "mysecret", dep.Spec.Template.Spec.Containers[0].EnvFrom[0].SecretRef.LocalObjectReference.Name)
}

func TestCollectorImagePullPolicy(t *testing.T) {
	jaeger := v1.NewJaeger(types.NamespacedName{Name: "TestCollectorImagePullPolicy"})
	const pullPolicy = corev1.PullPolicy("Always")
	jaeger.Spec.ImagePullPolicy = corev1.PullPolicy("Always")

	collector := NewCollector(jaeger)
	dep := collector.Get()

	assert.Equal(t, pullPolicy, dep.Spec.Template.Spec.Containers[0].ImagePullPolicy)
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
	assert.Equal(t, "globalVolume", podSpec.Containers[0].VolumeMounts[0].Name)
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
	assert.False(t, podSpec.Containers[0].VolumeMounts[0].ReadOnly)
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
	assert.Equal(t, "/data2", podSpec.Volumes[0].VolumeSource.HostPath.Path)
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
				Type: v1.JaegerESStorage,
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
			Value: string(v1.JaegerESStorage),
		},
		{
			Name:  "COLLECTOR_ZIPKIN_HOST_PORT",
			Value: ":9411",
		},
		{
			Name:  "COLLECTOR_OTLP_ENABLED",
			Value: "true",
		},
		{
			Name:  "COLLECTOR_OTLP_GRPC_HOST_PORT",
			Value: "0.0.0.0:4317",
		},
		{
			Name:  "COLLECTOR_OTLP_HTTP_HOST_PORT",
			Value: "0.0.0.0:4318",
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
			Name:  "COLLECTOR_ZIPKIN_HOST_PORT",
			Value: ":9411",
		},
		{
			Name:  "COLLECTOR_OTLP_ENABLED",
			Value: "true",
		},
		{
			Name:  "COLLECTOR_OTLP_GRPC_HOST_PORT",
			Value: "0.0.0.0:4317",
		},
		{
			Name:  "COLLECTOR_OTLP_HTTP_HOST_PORT",
			Value: "0.0.0.0:4318",
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
			Name:  "COLLECTOR_ZIPKIN_HOST_PORT",
			Value: ":9411",
		},
		{
			Name:  "COLLECTOR_OTLP_ENABLED",
			Value: "true",
		},
		{
			Name:  "COLLECTOR_OTLP_GRPC_HOST_PORT",
			Value: "0.0.0.0:4317",
		},
		{
			Name:  "COLLECTOR_OTLP_HTTP_HOST_PORT",
			Value: "0.0.0.0:4318",
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

func TestCollectorAutoscalersOnByDefaultV2(t *testing.T) {
	// prepare
	viper.Set(v1.FlagAutoscalingVersion, v1.FlagAutoscalingVersionV2)
	jaeger := v1.NewJaeger(types.NamespacedName{Name: "my-instance"})
	c := NewCollector(jaeger)

	// test
	a := c.Autoscalers()
	hpa := a[0].(*autoscalingv2.HorizontalPodAutoscaler)

	// verify
	assert.Len(t, a, 1)
	assert.Len(t, hpa.Spec.Metrics, 2)

	assert.Contains(t, []corev1.ResourceName{hpa.Spec.Metrics[0].Resource.Name, hpa.Spec.Metrics[1].Resource.Name}, corev1.ResourceCPU)
	assert.Contains(t, []corev1.ResourceName{hpa.Spec.Metrics[0].Resource.Name, hpa.Spec.Metrics[1].Resource.Name}, corev1.ResourceMemory)

	assert.Equal(t, int32(90), *hpa.Spec.Metrics[0].Resource.Target.AverageUtilization)
	assert.Equal(t, int32(90), *hpa.Spec.Metrics[1].Resource.Target.AverageUtilization)
}

func TestCollectorAutoscalersOnByDefaultV2Beta2(t *testing.T) {
	// prepare
	viper.Set(v1.FlagAutoscalingVersion, v1.FlagAutoscalingVersionV2Beta2)
	jaeger := v1.NewJaeger(types.NamespacedName{Name: "my-instance"})
	c := NewCollector(jaeger)

	// test
	a := c.Autoscalers()
	hpa := a[0].(*autoscalingv2beta2.HorizontalPodAutoscaler)

	// verify
	assert.Len(t, a, 1)
	assert.Len(t, hpa.Spec.Metrics, 2)

	assert.Contains(t, []corev1.ResourceName{hpa.Spec.Metrics[0].Resource.Name, hpa.Spec.Metrics[1].Resource.Name}, corev1.ResourceCPU)
	assert.Contains(t, []corev1.ResourceName{hpa.Spec.Metrics[0].Resource.Name, hpa.Spec.Metrics[1].Resource.Name}, corev1.ResourceMemory)

	assert.Equal(t, int32(90), *hpa.Spec.Metrics[0].Resource.Target.AverageUtilization)
	assert.Equal(t, int32(90), *hpa.Spec.Metrics[1].Resource.Target.AverageUtilization)
}

func TestCollectorAutoscalersDisabledByExplicitReplicaSize(t *testing.T) {
	// prepare
	viper.Set(v1.FlagAutoscalingVersion, v1.FlagAutoscalingVersionV2)
	tests := []int32{int32(0), int32(1)}

	for _, test := range tests {
		jaeger := v1.NewJaeger(types.NamespacedName{Name: "my-instance"})
		jaeger.Spec.Collector.Replicas = &test
		c := NewCollector(jaeger)

		// test
		a := c.Autoscalers()

		// verify
		assert.Empty(t, a)
	}
}

func TestCollectorAutoscalersDisabledByExplicitOption(t *testing.T) {
	// prepare
	viper.Set(v1.FlagAutoscalingVersion, v1.FlagAutoscalingVersionV2)
	disabled := false
	jaeger := v1.NewJaeger(types.NamespacedName{Name: "my-instance"})
	jaeger.Spec.Collector.Autoscale = &disabled
	c := NewCollector(jaeger)

	// test
	a := c.Autoscalers()

	// verify
	assert.Empty(t, a)
}

func TestCollectorAutoscalersSetMaxReplicas(t *testing.T) {
	// prepare
	viper.Set(v1.FlagAutoscalingVersion, v1.FlagAutoscalingVersionV2)
	maxReplicas := int32(2)
	jaeger := v1.NewJaeger(types.NamespacedName{Name: "my-instance"})
	jaeger.Spec.Collector.MaxReplicas = &maxReplicas
	c := NewCollector(jaeger)

	// test
	a := c.Autoscalers()
	hpa := a[0].(*autoscalingv2.HorizontalPodAutoscaler)

	// verify
	assert.Len(t, a, 1)
	assert.Equal(t, maxReplicas, hpa.Spec.MaxReplicas)
}

func TestCollectoArgumentsOpenshiftTLS(t *testing.T) {
	autodetect.OperatorConfiguration.SetPlatform(autodetect.OpenShiftPlatform)
	defer viper.Reset()
	for _, tt := range []struct {
		name            string
		options         v1.Options
		expectedArgs    []string
		nonExpectedArgs []string
	}{
		{
			name: "Openshift CA",
			options: v1.NewOptions(map[string]interface{}{
				"a-option": "a-value",
			}),
			expectedArgs: []string{
				"--a-option=a-value",
				"--collector.grpc.tls.enabled=true",
				"--collector.grpc.tls.cert=/etc/tls-config/tls.crt",
				"--collector.grpc.tls.key=/etc/tls-config/tls.key",
				"--sampling.strategies-file",
			},
		},
		{
			name: "Custom CA",
			options: v1.NewOptions(map[string]interface{}{
				"a-option":                   "a-value",
				"collector.grpc.tls.enabled": "true",
				"collector.grpc.tls.cert":    "/my/custom/cert",
				"collector.grpc.tls.key":     "/my/custom/key",
			}),
			expectedArgs: []string{
				"--a-option=a-value",
				"--collector.grpc.tls.enabled=true",
				"--collector.grpc.tls.cert=/my/custom/cert",
				"--collector.grpc.tls.key=/my/custom/key",
				"--sampling.strategies-file",
			},
		},
		{
			name: "Explicit disable TLS",
			options: v1.NewOptions(map[string]interface{}{
				"a-option":                   "a-value",
				"collector.grpc.tls.enabled": "false",
			}),
			expectedArgs: []string{
				"--a-option=a-value",
				"--collector.grpc.tls.enabled=false",
				"--sampling.strategies-file",
			},
			nonExpectedArgs: []string{
				"--collector.grpc.tls.enabled=true",
			},
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			jaeger := v1.NewJaeger(types.NamespacedName{Name: "my-instance"})
			jaeger.Spec.Collector.Options = tt.options

			a := NewCollector(jaeger)
			dep := a.Get()

			// verify
			assert.Len(t, dep.Spec.Template.Spec.Containers, 1)
			assert.Len(t, dep.Spec.Template.Spec.Containers[0].Args, len(tt.expectedArgs))

			for _, arg := range tt.expectedArgs {
				assert.NotEmpty(t, util.FindItem(arg, dep.Spec.Template.Spec.Containers[0].Args))
			}

			if tt.nonExpectedArgs != nil {
				for _, arg := range tt.nonExpectedArgs {
					assert.Empty(t, util.FindItem(arg, dep.Spec.Template.Spec.Containers[0].Args))
				}
			}
		})
	}
}

func TestCollectorServiceLinks(t *testing.T) {
	c := NewCollector(v1.NewJaeger(types.NamespacedName{Name: "my-instance"}))
	dep := c.Get()
	falseVar := false
	assert.Equal(t, &falseVar, dep.Spec.Template.Spec.EnableServiceLinks)
}

func TestCollectorPriorityClassName(t *testing.T) {
	priorityClassName := "test-class"
	jaeger := v1.NewJaeger(types.NamespacedName{Name: "my-instance"})
	jaeger.Spec.Collector.PriorityClassName = priorityClassName
	c := NewCollector(jaeger)
	dep := c.Get()
	assert.Equal(t, priorityClassName, dep.Spec.Template.Spec.PriorityClassName)
}

func TestCollectorRollingUpdateStrategyType(t *testing.T) {
	strategy := appsv1.DeploymentStrategy{
		Type: appsv1.RollingUpdateDeploymentStrategyType,
		RollingUpdate: &appsv1.RollingUpdateDeployment{
			MaxUnavailable: &intstr.IntOrString{},
			MaxSurge:       &intstr.IntOrString{},
		},
	}
	jaeger := v1.NewJaeger(types.NamespacedName{Name: "my-instance"})
	jaeger.Spec.Collector.Strategy = &strategy
	c := NewCollector(jaeger)
	dep := c.Get()
	assert.Equal(t, strategy.Type, dep.Spec.Strategy.Type)
}

func TestCollectorEmptyStrategyType(t *testing.T) {
	jaeger := v1.NewJaeger(types.NamespacedName{Name: "my-instance"})
	c := NewCollector(jaeger)
	dep := c.Get()
	assert.Equal(t, appsv1.RecreateDeploymentStrategyType, dep.Spec.Strategy.Type)
}

func TestCollectorLivenessProbe(t *testing.T) {
	livenessProbe := &corev1.Probe{
		ProbeHandler: corev1.ProbeHandler{
			HTTPGet: &corev1.HTTPGetAction{
				Path: "/",
				Port: intstr.FromInt(int(14269)),
			},
		},
		InitialDelaySeconds: 60,
		PeriodSeconds:       60,
		FailureThreshold:    60,
	}
	jaeger := v1.NewJaeger(types.NamespacedName{Name: "my-instance"})
	jaeger.Spec.Collector.LivenessProbe = livenessProbe
	c := NewCollector(jaeger)
	dep := c.Get()
	assert.Equal(t, livenessProbe, dep.Spec.Template.Spec.Containers[0].LivenessProbe)
}

func TestCollectorEmptyEmptyLivenessProbe(t *testing.T) {
	jaeger := v1.NewJaeger(types.NamespacedName{Name: "my-instance"})
	c := NewCollector(jaeger)
	dep := c.Get()
	assert.Equal(t, &corev1.Probe{
		ProbeHandler: corev1.ProbeHandler{
			HTTPGet: &corev1.HTTPGetAction{
				Path: "/",
				Port: intstr.FromInt(int(14269)),
			},
		},
		InitialDelaySeconds: 5,
		PeriodSeconds:       15,
		FailureThreshold:    5,
	}, dep.Spec.Template.Spec.Containers[0].LivenessProbe)
}

func TestCollectorGRPCPlugin(t *testing.T) {
	jaeger := v1.NewJaeger(types.NamespacedName{Name: "TestCollectorGRPCPlugin"})
	jaeger.Spec.Storage.Type = v1.JaegerGRPCPluginStorage
	jaeger.Spec.Storage.GRPCPlugin.Image = "plugin/plugin:1.0"
	jaeger.Spec.Storage.Options = v1.NewOptions(map[string]interface{}{
		"grpc-storage-plugin.binary": "/plugin/plugin",
	})

	collector := Collector{jaeger: jaeger}
	dep := collector.Get()

	assert.Equal(t, []corev1.Container{
		{
			Image: "plugin/plugin:1.0",
			Name:  "install-plugin",
			VolumeMounts: []corev1.VolumeMount{
				{
					Name:      "testcollectorgrpcplugin-sampling-configuration-volume",
					MountPath: "/etc/jaeger/sampling",
					ReadOnly:  true,
				},
				{
					Name:      "plugin-volume",
					MountPath: "/plugin",
				},
			},
		},
	}, dep.Spec.Template.Spec.InitContainers)
	require.Len(t, dep.Spec.Template.Spec.Containers, 1)
	assert.Equal(t, []string{"--grpc-storage-plugin.binary=/plugin/plugin", "--sampling.strategies-file=/etc/jaeger/sampling/sampling.json"}, dep.Spec.Template.Spec.Containers[0].Args)
}

func TestCollectorContainerSecurityContext(t *testing.T) {
	trueVar := true
	idVar := int64(1234)
	securityContextVar := corev1.SecurityContext{
		RunAsNonRoot: &trueVar,
		RunAsGroup:   &idVar,
		RunAsUser:    &idVar,
	}
	jaeger := v1.NewJaeger(types.NamespacedName{Name: "my-instance"})
	jaeger.Spec.Collector.ContainerSecurityContext = &securityContextVar

	c := NewCollector(jaeger)
	dep := c.Get()

	assert.Equal(t, securityContextVar, *dep.Spec.Template.Spec.Containers[0].SecurityContext)
}

func TestCollectorContainerSecurityContextOverride(t *testing.T) {
	trueVar := true
	idVar1 := int64(1234)
	idVar2 := int64(4321)
	securityContextVar := corev1.SecurityContext{
		RunAsNonRoot: &trueVar,
		RunAsGroup:   &idVar1,
		RunAsUser:    &idVar1,
	}
	overrideSecurityContextVar := corev1.SecurityContext{
		RunAsNonRoot: &trueVar,
		RunAsGroup:   &idVar2,
		RunAsUser:    &idVar2,
	}
	jaeger := v1.NewJaeger(types.NamespacedName{Name: "my-instance"})
	jaeger.Spec.ContainerSecurityContext = &securityContextVar
	jaeger.Spec.Collector.ContainerSecurityContext = &overrideSecurityContextVar

	c := NewCollector(jaeger)
	dep := c.Get()

	assert.Equal(t, overrideSecurityContextVar, *dep.Spec.Template.Spec.Containers[0].SecurityContext)
}

func TestCollectorNodeSelector(t *testing.T) {
	jaeger := v1.NewJaeger(types.NamespacedName{Name: "my-instance"})
	nodeSelector := map[string]string{
		"agentpool": "service",
	}
	jaeger.Spec.Collector.NodeSelector = nodeSelector

	c := NewCollector(jaeger)
	dep := c.Get()

	assert.Equal(t, nodeSelector, dep.Spec.Template.Spec.NodeSelector)
}

func TestCollectorLifecyle(t *testing.T) {
	jaeger := v1.NewJaeger(types.NamespacedName{Name: "my-instance"})
	lifecycle := &corev1.Lifecycle{
		PreStop: &corev1.LifecycleHandler{
			Exec: &corev1.ExecAction{
				Command: []string{"command"},
			},
		},
	}
	jaeger.Spec.Collector.Lifecycle = lifecycle

	c := NewCollector(jaeger)
	dep := c.Get()

	assert.Equal(t, lifecycle, dep.Spec.Template.Spec.Containers[0].Lifecycle)
}

func TestCollectorTerminationGracePeriodSeconds(t *testing.T) {
	jaeger := v1.NewJaeger(types.NamespacedName{Name: "my-instance"})
	terminationGracePeriodSeconds := int64(10)
	jaeger.Spec.Collector.TerminationGracePeriodSeconds = &terminationGracePeriodSeconds

	c := NewCollector(jaeger)
	dep := c.Get()

	assert.Equal(t, terminationGracePeriodSeconds, *dep.Spec.Template.Spec.TerminationGracePeriodSeconds)
}
