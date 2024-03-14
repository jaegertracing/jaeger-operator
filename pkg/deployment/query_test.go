package deployment

import (
	"fmt"
	"strings"
	"testing"

	"github.com/spf13/viper"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/intstr"

	v1 "github.com/jaegertracing/jaeger-operator/apis/v1"
	"github.com/jaegertracing/jaeger-operator/pkg/version"
)

func init() {
	viper.SetDefault("jaeger-query-image", "jaegertracing/all-in-one")
}

func TestQueryNegativeReplicas(t *testing.T) {
	size := int32(-1)
	jaeger := v1.NewJaeger(types.NamespacedName{Name: "TestQueryNegativeReplicas"})
	jaeger.Spec.Query.Replicas = &size

	query := NewQuery(jaeger)
	dep := query.Get()
	assert.Equal(t, size, *dep.Spec.Replicas)
}

func TestQueryDefaultSize(t *testing.T) {
	jaeger := v1.NewJaeger(types.NamespacedName{Name: "TestQueryDefaultSize"})

	query := NewQuery(jaeger)
	dep := query.Get()
	assert.Nil(t, dep.Spec.Replicas) // we let Kubernetes define the default
}

func TestQueryReplicaSize(t *testing.T) {
	size := int32(0)
	jaeger := v1.NewJaeger(types.NamespacedName{Name: "TestQueryReplicaSize"})
	jaeger.Spec.Query.Replicas = &size

	ingester := NewQuery(jaeger)
	dep := ingester.Get()
	assert.Equal(t, int32(0), *dep.Spec.Replicas)
}

func TestDefaultQueryImage(t *testing.T) {
	viper.Set("jaeger-query-image", "org/custom-query-image")
	defer viper.Reset()

	jaeger := v1.NewJaeger(types.NamespacedName{Name: "TestQueryImage"})
	query := NewQuery(jaeger)
	dep := query.Get()
	containers := dep.Spec.Template.Spec.Containers

	assert.Len(t, containers, 1)
	assert.Empty(t, jaeger.Spec.Query.Image)
	assert.Equal(t, "org/custom-query-image:"+version.Get().Jaeger, containers[0].Image)
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
	assert.Equal(t, "operator", dep.Spec.Selector.MatchLabels["name"])
	assert.Equal(t, "world", dep.Spec.Selector.MatchLabels["hello"])
	assert.Equal(t, "false", dep.Spec.Selector.MatchLabels["another"])
}

func TestQuerySecrets(t *testing.T) {
	jaeger := v1.NewJaeger(types.NamespacedName{Name: "TestQuerySecrets"})
	secret := "mysecret"
	jaeger.Spec.Storage.SecretName = secret

	query := NewQuery(jaeger)
	dep := query.Get()

	assert.Equal(t, "mysecret", dep.Spec.Template.Spec.Containers[0].EnvFrom[0].SecretRef.LocalObjectReference.Name)
}

func TestQueryImagePullSecrets(t *testing.T) {
	jaeger := v1.NewJaeger(types.NamespacedName{Name: "TestAllInOneImagePullSecrets"})
	const pullSecret = "mysecret"
	jaeger.Spec.ImagePullSecrets = []corev1.LocalObjectReference{
		{
			Name: pullSecret,
		},
	}

	query := NewQuery(jaeger)
	dep := query.Get()

	assert.Equal(t, pullSecret, dep.Spec.Template.Spec.ImagePullSecrets[0].Name)
}

func TestQueryImagePullPolicy(t *testing.T) {
	jaeger := v1.NewJaeger(types.NamespacedName{Name: "TestQueryImagePullPolicy"})
	const pullPolicy = corev1.PullPolicy("Always")
	jaeger.Spec.ImagePullPolicy = corev1.PullPolicy("Always")

	query := NewQuery(jaeger)
	dep := query.Get()

	assert.Equal(t, pullPolicy, dep.Spec.Template.Spec.Containers[0].ImagePullPolicy)
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
	assert.Equal(t, "globalVolume", podSpec.Containers[0].VolumeMounts[0].Name)
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
	assert.False(t, podSpec.Containers[0].VolumeMounts[0].ReadOnly)
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
	assert.Equal(t, "/data2", podSpec.Volumes[0].VolumeSource.HostPath.Path)
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

func TestQueryServiceLinks(t *testing.T) {
	query := NewQuery(v1.NewJaeger(types.NamespacedName{Name: "TestQueryServiceLinks"}))
	dep := query.Get()
	falseVar := false
	assert.Equal(t, &falseVar, dep.Spec.Template.Spec.EnableServiceLinks)
}

func TestQueryTracingDisabled(t *testing.T) {
	jaeger := v1.NewJaeger(types.NamespacedName{Name: "TestQueryJaegerDisabled"})
	falseVar := false
	jaeger.Spec.Query.TracingEnabled = &falseVar
	query := NewQuery(jaeger)
	dep := query.Get()
	assert.Equal(t, "true", getEnvVarByName(dep.Spec.Template.Spec.Containers[0].Env, "JAEGER_DISABLED").Value)
}

func TestQueryPriorityClassName(t *testing.T) {
	priorityClassName := "test-class"
	jaeger := v1.NewJaeger(types.NamespacedName{Name: "my-instance"})
	jaeger.Spec.Query.PriorityClassName = priorityClassName
	q := NewQuery(jaeger)
	dep := q.Get()
	assert.Equal(t, priorityClassName, dep.Spec.Template.Spec.PriorityClassName)
}

func TestQueryRollingUpdateStrategyType(t *testing.T) {
	strategy := appsv1.DeploymentStrategy{
		Type: appsv1.RollingUpdateDeploymentStrategyType,
		RollingUpdate: &appsv1.RollingUpdateDeployment{
			MaxUnavailable: &intstr.IntOrString{},
			MaxSurge:       &intstr.IntOrString{},
		},
	}
	jaeger := v1.NewJaeger(types.NamespacedName{Name: "my-instance"})
	jaeger.Spec.Query.Strategy = &strategy
	q := NewQuery(jaeger)
	dep := q.Get()
	assert.Equal(t, strategy.Type, dep.Spec.Strategy.Type)
}

func TestQueryEmptyStrategyType(t *testing.T) {
	jaeger := v1.NewJaeger(types.NamespacedName{Name: "my-instance"})
	q := NewQuery(jaeger)
	dep := q.Get()
	assert.Equal(t, appsv1.RecreateDeploymentStrategyType, dep.Spec.Strategy.Type)
}

func TestQueryLivenessProbe(t *testing.T) {
	livenessProbe := &corev1.Probe{
		ProbeHandler: corev1.ProbeHandler{
			HTTPGet: &corev1.HTTPGetAction{
				Path: "/",
				Port: intstr.FromInt(int(16687)),
			},
		},
		InitialDelaySeconds: 60,
		PeriodSeconds:       60,
		FailureThreshold:    60,
	}
	jaeger := v1.NewJaeger(types.NamespacedName{Name: "my-instance"})
	jaeger.Spec.Query.LivenessProbe = livenessProbe
	q := NewQuery(jaeger)
	dep := q.Get()
	assert.Equal(t, livenessProbe, dep.Spec.Template.Spec.Containers[0].LivenessProbe)
}

func TestQueryEmptyEmptyLivenessProbe(t *testing.T) {
	jaeger := v1.NewJaeger(types.NamespacedName{Name: "my-instance"})
	q := NewQuery(jaeger)
	dep := q.Get()
	assert.Equal(t, &corev1.Probe{
		ProbeHandler: corev1.ProbeHandler{
			HTTPGet: &corev1.HTTPGetAction{
				Path: "/",
				Port: intstr.FromInt(int(16687)),
			},
		},
		InitialDelaySeconds: 5,
		PeriodSeconds:       15,
		FailureThreshold:    5,
	}, dep.Spec.Template.Spec.Containers[0].LivenessProbe)
}

func TestQueryGRPCPlugin(t *testing.T) {
	jaeger := v1.NewJaeger(types.NamespacedName{Name: "TestQueryGRPCPlugin"})
	jaeger.Spec.Storage.Type = v1.JaegerGRPCPluginStorage
	jaeger.Spec.Storage.GRPCPlugin.Image = "plugin/plugin:1.0"
	jaeger.Spec.Storage.Options = v1.NewOptions(map[string]interface{}{
		"grpc-storage-plugin.binary": "/plugin/plugin",
	})

	query := Query{jaeger: jaeger}
	dep := query.Get()

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

func TestQueryContainerSecurityContext(t *testing.T) {
	trueVar := true
	idVar := int64(1234)
	securityContextVar := corev1.SecurityContext{
		RunAsNonRoot: &trueVar,
		RunAsGroup:   &idVar,
		RunAsUser:    &idVar,
	}
	jaeger := v1.NewJaeger(types.NamespacedName{Name: "my-instance"})
	jaeger.Spec.Query.ContainerSecurityContext = &securityContextVar

	q := NewQuery(jaeger)
	dep := q.Get()

	assert.Equal(t, securityContextVar, *dep.Spec.Template.Spec.Containers[0].SecurityContext)
}

func TestQueryContainerSecurityContextOverride(t *testing.T) {
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
	jaeger.Spec.Query.ContainerSecurityContext = &overrideSecurityContextVar

	q := NewQuery(jaeger)
	dep := q.Get()

	assert.Equal(t, overrideSecurityContextVar, *dep.Spec.Template.Spec.Containers[0].SecurityContext)
}

func TestQueryNodeSelector(t *testing.T) {
	jaeger := v1.NewJaeger(types.NamespacedName{Name: "my-instance"})
	nodeSelector := map[string]string{
		"agentpool": "service",
	}
	jaeger.Spec.Query.NodeSelector = nodeSelector

	q := NewQuery(jaeger)
	dep := q.Get()

	assert.Equal(t, nodeSelector, dep.Spec.Template.Spec.NodeSelector)
}

func TestQueryPrometheusMetricStorage(t *testing.T) {
	jaeger := v1.NewJaeger(types.NamespacedName{Name: "TestQueryPrometheusMetricStorage"})

	jaeger.Spec.Query.MetricsStorage.Type = "prometheus"
	jaeger.Spec.Query.MetricsStorage.ServerUrl = "http://prometheus:9090"

	d := NewQuery(jaeger).Get()

	assert.Equal(t, "prometheus", getEnvVarByName(d.Spec.Template.Spec.Containers[0].Env, "METRICS_STORAGE_TYPE").Value)
	assert.NotEmpty(t, getEnvVarByName(d.Spec.Template.Spec.Containers[0].Env, "PROMETHEUS_SERVER_URL").Value)
}
