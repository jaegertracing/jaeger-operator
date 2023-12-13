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
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/intstr"

	v1 "github.com/jaegertracing/jaeger-operator/apis/v1"
	"github.com/jaegertracing/jaeger-operator/pkg/version"
)

func init() {
	viper.SetDefault("jaeger-ingester-image", "jaegertracing/jaeger-ingester")
}

func TestIngesterNotDefined(t *testing.T) {
	jaeger := v1.NewJaeger(types.NamespacedName{Name: "TestIngesterNotDefined"})

	ingester := NewIngester(jaeger)
	assert.Nil(t, ingester.Get())
}

func TestIngesterNegativeReplicas(t *testing.T) {
	size := int32(-1)
	jaeger := newIngesterJaeger("TestIngesterNegativeReplicas")
	jaeger.Spec.Ingester.Replicas = &size

	ingester := NewIngester(jaeger)
	dep := ingester.Get()
	assert.Equal(t, size, *dep.Spec.Replicas)
}

func TestIngesterDefaultSize(t *testing.T) {
	jaeger := newIngesterJaeger("TestIngesterDefaultSize")

	ingester := NewIngester(jaeger)
	dep := ingester.Get()
	assert.Nil(t, dep.Spec.Replicas) // we let Kubernetes define the default
}

func TestIngesterReplicaSize(t *testing.T) {
	size := int32(0)
	jaeger := newIngesterJaeger("TestIngesterReplicaSize")
	jaeger.Spec.Ingester.Replicas = &size

	ingester := NewIngester(jaeger)
	dep := ingester.Get()
	assert.Equal(t, int32(0), *dep.Spec.Replicas)
}

func TestIngesterName(t *testing.T) {
	jaeger := newIngesterJaeger("TestIngesterName")
	ingester := NewIngester(jaeger)
	dep := ingester.Get()
	assert.Equal(t, "TestIngesterName-ingester", dep.ObjectMeta.Name)
}

func TestDefaultIngesterImage(t *testing.T) {
	viper.Set("jaeger-ingester-image", "org/custom-ingester-image")
	defer viper.Reset()

	jaeger := newIngesterJaeger("my-instance")
	ingester := NewIngester(jaeger)
	dep := ingester.Get()

	containers := dep.Spec.Template.Spec.Containers
	assert.Len(t, containers, 1)
	assert.Empty(t, jaeger.Spec.Ingester.Image)
	assert.Equal(t, "org/custom-ingester-image:"+version.Get().Jaeger, containers[0].Image)

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
	assert.Equal(t, "operator", dep.Spec.Selector.MatchLabels["name"])
	assert.Equal(t, "world", dep.Spec.Selector.MatchLabels["hello"])
	assert.Equal(t, "false", dep.Spec.Selector.MatchLabels["another"])
}

func TestIngesterSecrets(t *testing.T) {
	jaeger := newIngesterJaeger("TestIngesterSecrets")
	secret := "mysecret"
	jaeger.Spec.Storage.SecretName = secret

	ingester := NewIngester(jaeger)
	dep := ingester.Get()

	assert.Equal(t, "mysecret", dep.Spec.Template.Spec.Containers[0].EnvFrom[0].SecretRef.LocalObjectReference.Name)
}

func TestIngeterImagePullSecrets(t *testing.T) {
	jaeger := newIngesterJaeger("TestIngesterImagePullSecrets")
	const pullSecret = "mysecret"
	jaeger.Spec.ImagePullSecrets = []corev1.LocalObjectReference{
		{
			Name: pullSecret,
		},
	}

	ingester := NewIngester(jaeger)
	dep := ingester.Get()

	assert.Equal(t, pullSecret, dep.Spec.Template.Spec.ImagePullSecrets[0].Name)
}

func TestIngesterKafkaSecrets(t *testing.T) {
	jaeger := newIngesterJaeger("TestIngesterKafkaSecrets")
	secret := "mysecret"
	jaeger.Spec.Storage.SecretName = secret

	ingester := NewIngester(jaeger)
	dep := ingester.Get()

	assert.Equal(t, "mysecret", dep.Spec.Template.Spec.Containers[0].EnvFrom[0].SecretRef.LocalObjectReference.Name)
}

func TestIngesterImagePullPolicy(t *testing.T) {
	jaeger := newIngesterJaeger("TestIngesterImagePullPolicy")
	const pullPolicy = corev1.PullPolicy("Always")
	jaeger.Spec.ImagePullPolicy = corev1.PullPolicy("Always")

	ingester := NewIngester(jaeger)
	dep := ingester.Get()

	assert.Equal(t, pullPolicy, dep.Spec.Template.Spec.Containers[0].ImagePullPolicy)
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
	assert.Equal(t, "globalVolume", podSpec.Containers[0].VolumeMounts[0].Name)
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
	assert.False(t, podSpec.Containers[0].VolumeMounts[0].ReadOnly)
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
	assert.Equal(t, "/data2", podSpec.Volumes[0].VolumeSource.HostPath.Path)
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
			Strategy: v1.DeploymentStrategyStreaming,
			Ingester: v1.JaegerIngesterSpec{
				Options: v1.NewOptions(map[string]interface{}{
					"kafka.consumer.topic":   "mytopic",
					"kafka.consumer.brokers": "http://brokers",
				}),
			},
			Storage: v1.JaegerStorageSpec{
				Type: v1.JaegerESStorage,
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
			Value: string(v1.JaegerESStorage),
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

func TestIngesterAutoscalersOnByDefault(t *testing.T) {
	// prepare
	jaeger := v1.NewJaeger(types.NamespacedName{Name: "my-instance"})
	c := NewIngester(jaeger)

	// test
	a := c.Autoscalers()

	autoscaler := a[0].(*autoscalingv2.HorizontalPodAutoscaler)

	// verify
	assert.Len(t, a, 1)
	assert.Len(t, autoscaler.Spec.Metrics, 2)

	assert.Contains(t, []corev1.ResourceName{autoscaler.Spec.Metrics[0].Resource.Name, autoscaler.Spec.Metrics[1].Resource.Name}, corev1.ResourceCPU)
	assert.Contains(t, []corev1.ResourceName{autoscaler.Spec.Metrics[0].Resource.Name, autoscaler.Spec.Metrics[1].Resource.Name}, corev1.ResourceMemory)

	assert.Equal(t, int32(90), *autoscaler.Spec.Metrics[0].Resource.Target.AverageUtilization)
	assert.Equal(t, int32(90), *autoscaler.Spec.Metrics[1].Resource.Target.AverageUtilization)
}

func TestIngesterAutoscalersDisabledByExplicitReplicaSize(t *testing.T) {
	// prepare
	tests := []int32{int32(0), int32(1)}

	for _, test := range tests {
		jaeger := v1.NewJaeger(types.NamespacedName{Name: "my-instance"})
		jaeger.Spec.Ingester.Replicas = &test
		c := NewIngester(jaeger)

		// test
		a := c.Autoscalers()

		// verify
		assert.Empty(t, a)
	}
}

func TestIngesterAutoscalersDisabledByExplicitOption(t *testing.T) {
	// prepare
	disabled := false
	jaeger := v1.NewJaeger(types.NamespacedName{Name: "my-instance"})
	jaeger.Spec.Ingester.Autoscale = &disabled
	c := NewIngester(jaeger)

	// test
	a := c.Autoscalers()

	// verify
	assert.Empty(t, a)
}

func TestIngesterAutoscalersSetMaxReplicas(t *testing.T) {
	// prepare
	maxReplicas := int32(2)
	jaeger := v1.NewJaeger(types.NamespacedName{Name: "my-instance"})
	jaeger.Spec.Ingester.MaxReplicas = &maxReplicas
	c := NewIngester(jaeger)

	// test
	a := c.Autoscalers()
	hpa := a[0].(*autoscalingv2.HorizontalPodAutoscaler)

	// verify
	assert.Len(t, a, 1)
	assert.Equal(t, maxReplicas, hpa.Spec.MaxReplicas)
}

func newIngesterJaeger(name string) *v1.Jaeger {
	return &v1.Jaeger{
		ObjectMeta: metav1.ObjectMeta{
			Name: name,
		},
		Spec: v1.JaegerSpec{
			Strategy: v1.DeploymentStrategyStreaming,
			Ingester: v1.JaegerIngesterSpec{
				Options: v1.NewOptions(map[string]interface{}{
					"any": "option",
				}),
			},
		},
	}
}

func TestIngesterServiceLinks(t *testing.T) {
	ingester := NewIngester(newIngesterJaeger("TestIngesterServiceLinks"))
	dep := ingester.Get()
	falseVar := false
	assert.Equal(t, &falseVar, dep.Spec.Template.Spec.EnableServiceLinks)
}

func TestIngesterRollingUpdateStrategyType(t *testing.T) {
	strategy := appsv1.DeploymentStrategy{
		Type: appsv1.RollingUpdateDeploymentStrategyType,
		RollingUpdate: &appsv1.RollingUpdateDeployment{
			MaxUnavailable: &intstr.IntOrString{},
			MaxSurge:       &intstr.IntOrString{},
		},
	}
	jaeger := newIngesterJaeger("my-instance")
	jaeger.Spec.Ingester.Strategy = &strategy
	i := NewIngester(jaeger)
	dep := i.Get()
	assert.Equal(t, strategy.Type, dep.Spec.Strategy.Type)
}

func TestIngesterEmptyStrategyType(t *testing.T) {
	jaeger := newIngesterJaeger("my-instance")
	i := NewIngester(jaeger)
	dep := i.Get()
	assert.Equal(t, appsv1.RecreateDeploymentStrategyType, dep.Spec.Strategy.Type)
}

func TestIngesterLivenessProbe(t *testing.T) {
	livenessProbe := &corev1.Probe{
		ProbeHandler: corev1.ProbeHandler{
			HTTPGet: &corev1.HTTPGetAction{
				Path: "/",
				Port: intstr.FromInt(int(14270)),
			},
		},
		InitialDelaySeconds: 60,
		PeriodSeconds:       60,
		FailureThreshold:    60,
	}
	jaeger := newIngesterJaeger("my-instance")
	jaeger.Spec.Ingester.LivenessProbe = livenessProbe
	i := NewIngester(jaeger)
	dep := i.Get()
	assert.Equal(t, livenessProbe, dep.Spec.Template.Spec.Containers[0].LivenessProbe)
}

func TestIngesterEmptyEmptyLivenessProbe(t *testing.T) {
	jaeger := newIngesterJaeger("my-instance")
	i := NewIngester(jaeger)
	dep := i.Get()
	assert.Equal(t, &corev1.Probe{
		ProbeHandler: corev1.ProbeHandler{
			HTTPGet: &corev1.HTTPGetAction{
				Path: "/",
				Port: intstr.FromInt(int(14270)),
			},
		},
		InitialDelaySeconds: 5,
		PeriodSeconds:       15,
		FailureThreshold:    5,
	}, dep.Spec.Template.Spec.Containers[0].LivenessProbe)
}

func TestIngesterGRPCPlugin(t *testing.T) {
	jaeger := v1.NewJaeger(types.NamespacedName{Name: "TestIngesterGRPCPlugin"})
	jaeger.Spec.Strategy = v1.DeploymentStrategyStreaming
	jaeger.Spec.Storage.Type = v1.JaegerGRPCPluginStorage
	jaeger.Spec.Storage.GRPCPlugin.Image = "plugin/plugin:1.0"
	jaeger.Spec.Storage.Options = v1.NewOptions(map[string]interface{}{
		"grpc-storage-plugin.binary": "/plugin/plugin",
	})

	ingester := Ingester{jaeger: jaeger}
	dep := ingester.Get()

	assert.Equal(t, []corev1.Container{
		{
			Image: "plugin/plugin:1.0",
			Name:  "install-plugin",
			VolumeMounts: []corev1.VolumeMount{
				{
					Name:      "plugin-volume",
					MountPath: "/plugin",
				},
			},
		},
	}, dep.Spec.Template.Spec.InitContainers)
	require.Len(t, dep.Spec.Template.Spec.Containers, 1)
	assert.Equal(t, []string{"--grpc-storage-plugin.binary=/plugin/plugin"}, dep.Spec.Template.Spec.Containers[0].Args)
}

func TestIngesterContainerSecurityContext(t *testing.T) {
	trueVar := true
	idVar := int64(1234)
	securityContextVar := corev1.SecurityContext{
		RunAsNonRoot: &trueVar,
		RunAsGroup:   &idVar,
		RunAsUser:    &idVar,
	}
	jaeger := newIngesterJaeger("my-instance")
	jaeger.Spec.Ingester.ContainerSecurityContext = &securityContextVar

	i := NewIngester(jaeger)
	dep := i.Get()

	assert.Equal(t, securityContextVar, *dep.Spec.Template.Spec.Containers[0].SecurityContext)
}

func TestIngesterContainerSecurityContextOverride(t *testing.T) {
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
	jaeger := newIngesterJaeger("my-instance")
	jaeger.Spec.ContainerSecurityContext = &securityContextVar
	jaeger.Spec.Ingester.ContainerSecurityContext = &overrideSecurityContextVar

	i := NewIngester(jaeger)
	dep := i.Get()

	assert.Equal(t, overrideSecurityContextVar, *dep.Spec.Template.Spec.Containers[0].SecurityContext)
}

func TestIngesterNodeSelector(t *testing.T) {
	jaeger := newIngesterJaeger("my-instance")
	nodeSelector := map[string]string{
		"agentpool": "service",
	}
	jaeger.Spec.Ingester.NodeSelector = nodeSelector

	i := NewIngester(jaeger)
	dep := i.Get()

	assert.Equal(t, nodeSelector, dep.Spec.Template.Spec.NodeSelector)
}
