package deployment

import (
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
	"github.com/jaegertracing/jaeger-operator/pkg/autodetect"
	"github.com/jaegertracing/jaeger-operator/pkg/util"
	"github.com/jaegertracing/jaeger-operator/pkg/version"
)

func init() {
	viper.SetDefault("jaeger-all-in-one-image", "jaegertracing/all-in-one")
}

func TestDefaultAllInOneImage(t *testing.T) {
	viper.Set("jaeger-all-in-one-image", "org/custom-all-in-one-image")
	defer viper.Reset()

	jaeger := v1.NewJaeger(types.NamespacedName{Name: "my-instance"})
	d := NewAllInOne(jaeger).Get()

	assert.Len(t, d.Spec.Template.Spec.Containers, 1)
	assert.Empty(t, jaeger.Spec.AllInOne.Image)
	assert.Equal(t, "org/custom-all-in-one-image:"+version.Get().Jaeger, d.Spec.Template.Spec.Containers[0].Image)

	envvars := []corev1.EnvVar{
		{
			Name:  "SPAN_STORAGE_TYPE",
			Value: "",
		},
		{
			Name:  "METRICS_STORAGE_TYPE",
			Value: "",
		},
		{
			Name:  "COLLECTOR_ZIPKIN_HOST_PORT",
			Value: ":9411",
		},
		{
			Name:  "JAEGER_DISABLED",
			Value: "false",
		},
		{
			Name:  "COLLECTOR_OTLP_ENABLED",
			Value: "true",
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

	// Deployment selectors should be the same as the template labels.
	assert.Equal(t, "operator", dep.Spec.Selector.MatchLabels["name"])
	assert.Equal(t, "world", dep.Spec.Selector.MatchLabels["hello"])
	assert.Equal(t, "false", dep.Spec.Selector.MatchLabels["another"])
}

func TestAllInOneOverwrittenDefaultLabels(t *testing.T) {
	jaeger := v1.NewJaeger(types.NamespacedName{Name: "TestAllInOneOverwrittenDefaultLabels"})
	jaeger.Spec.Labels = map[string]string{
		"name":                   "operator",
		"hello":                  "jaeger",
		"app.kubernetes.io/name": "my-jaeger", // Override default labels
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
	assert.Equal(t, "my-jaeger", dep.Spec.Template.Labels["app.kubernetes.io/name"])

	// Deployment selectors should be the same as the template labels.
	assert.Equal(t, "operator", dep.Spec.Selector.MatchLabels["name"])
	assert.Equal(t, "world", dep.Spec.Selector.MatchLabels["hello"])
	assert.Equal(t, "false", dep.Spec.Selector.MatchLabels["another"])
	assert.Equal(t, "my-jaeger", dep.Spec.Selector.MatchLabels["app.kubernetes.io/name"])

	// Service selectors should be the same as the template labels.
	services := allinone.Services()
	for _, svc := range services {
		assert.Equal(t, "operator", svc.Spec.Selector["name"])
		assert.Equal(t, "world", svc.Spec.Selector["hello"])
		assert.Equal(t, "false", svc.Spec.Selector["another"])
		assert.Equal(t, "my-jaeger", svc.Spec.Selector["app.kubernetes.io/name"])
	}
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

func TestAllInOneImagePullSecrets(t *testing.T) {
	jaeger := v1.NewJaeger(types.NamespacedName{Name: "TestAllInOneImagePullSecrets"})
	const pullSecret = "mysecret"
	jaeger.Spec.ImagePullSecrets = []corev1.LocalObjectReference{
		{
			Name: pullSecret,
		},
	}

	allInOne := NewAllInOne(jaeger)
	dep := allInOne.Get()

	assert.Equal(t, pullSecret, dep.Spec.Template.Spec.ImagePullSecrets[0].Name)
}

func TestAllInOneImagePullPolicy(t *testing.T) {
	jaeger := v1.NewJaeger(types.NamespacedName{Name: "TestAllInOneImagePullPolicy"})
	const pullPolicy = corev1.PullPolicy("Always")
	jaeger.Spec.ImagePullPolicy = corev1.PullPolicy("Always")

	allInOne := NewAllInOne(jaeger)
	dep := allInOne.Get()

	assert.Equal(t, pullPolicy, dep.Spec.Template.Spec.Containers[0].ImagePullPolicy)
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
	assert.Equal(t, "globalVolume", podSpec.Containers[0].VolumeMounts[0].Name)
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
	assert.False(t, podSpec.Containers[0].VolumeMounts[0].ReadOnly)
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
	assert.Equal(t, "/data2", podSpec.Volumes[0].VolumeSource.HostPath.Path)
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
	assert.NotEmpty(t, util.FindItem("--a-option", dep.Spec.Template.Spec.Containers[0].Args))
	assert.NotEmpty(t, util.FindItem("--b-option", dep.Spec.Template.Spec.Containers[0].Args))
	assert.NotEmpty(t, util.FindItem("--c-option", dep.Spec.Template.Spec.Containers[0].Args))

	// the following are added automatically
	assert.NotEmpty(t, util.FindItem("--sampling.strategies-file", dep.Spec.Template.Spec.Containers[0].Args))
}

func TestAllInOneArgumentsOpenshiftTLS(t *testing.T) {
	// prepare
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
				"--reporter.grpc.tls.ca",
				"--reporter.grpc.tls.enabled",
				"--reporter.grpc.tls.server-name",
			},
		},
		{
			name: "Explicit disable TLS",
			options: v1.NewOptions(map[string]interface{}{
				"a-option":                   "a-value",
				"reporter.grpc.tls.enabled":  "false",
				"collector.grpc.tls.enabled": "false",
			}),
			expectedArgs: []string{
				"--a-option=a-value",
				"--reporter.grpc.tls.enabled=false",
				"--collector.grpc.tls.enabled=false",
				"--sampling.strategies-file",
			},
			nonExpectedArgs: []string{
				"--reporter.grpc.tls.enabled=true",
				"--collector.grpc.tls.enabled=true",
			},
		},
		{
			name: "Do not implicitly enable TLS when grpc.host-port is provided",
			options: v1.NewOptions(map[string]interface{}{
				"a-option":                "a-value",
				"reporter.grpc.host-port": "my.host-port.com",
			}),
			expectedArgs: []string{
				"--a-option=a-value",
				"--reporter.grpc.host-port=my.host-port.com",
				"--sampling.strategies-file",
			},
			nonExpectedArgs: []string{
				"--reporter.grpc.tls.enabled=true",
				"--collector.grpc.tls.enabled=true",
			},
		},
	} {
		jaeger := v1.NewJaeger(types.NamespacedName{Name: "my-instance"})
		jaeger.Spec.AllInOne.Options = tt.options

		// test
		a := NewAllInOne(jaeger)
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
	}
}

func TestAllInOneServiceLinks(t *testing.T) {
	a := NewAllInOne(v1.NewJaeger(types.NamespacedName{Name: "TestAllInOneServiceLinks"}))
	dep := a.Get()
	falseVar := false
	assert.Equal(t, &falseVar, dep.Spec.Template.Spec.EnableServiceLinks)
}

func TestAllInOneTracingDisabled(t *testing.T) {
	jaeger := v1.NewJaeger(types.NamespacedName{Name: "TestAllInOneTracingDisabled"})
	falseVar := false
	jaeger.Spec.AllInOne.TracingEnabled = &falseVar
	d := NewAllInOne(jaeger).Get()
	assert.Equal(t, "true", getEnvVarByName(d.Spec.Template.Spec.Containers[0].Env, "JAEGER_DISABLED").Value)
}

func TestAllInOneRollingUpdateStrategyType(t *testing.T) {
	strategy := appsv1.DeploymentStrategy{
		Type: appsv1.RollingUpdateDeploymentStrategyType,
		RollingUpdate: &appsv1.RollingUpdateDeployment{
			MaxUnavailable: &intstr.IntOrString{},
			MaxSurge:       &intstr.IntOrString{},
		},
	}
	jaeger := v1.NewJaeger(types.NamespacedName{Name: "my-instance"})
	jaeger.Spec.AllInOne.Strategy = &strategy
	a := NewAllInOne(jaeger)
	dep := a.Get()
	assert.Equal(t, strategy.Type, dep.Spec.Strategy.Type)
}

func TestAllInOneEmptyStrategyType(t *testing.T) {
	jaeger := v1.NewJaeger(types.NamespacedName{Name: "my-instance"})
	a := NewAllInOne(jaeger)
	dep := a.Get()
	assert.Equal(t, appsv1.RecreateDeploymentStrategyType, dep.Spec.Strategy.Type)
}

func TestAllInOneLivenessProbe(t *testing.T) {
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
	jaeger.Spec.AllInOne.LivenessProbe = livenessProbe
	a := NewAllInOne(jaeger)
	dep := a.Get()
	assert.Equal(t, livenessProbe, dep.Spec.Template.Spec.Containers[0].LivenessProbe)
}

func TestAllInOneEmptyEmptyLivenessProbe(t *testing.T) {
	jaeger := v1.NewJaeger(types.NamespacedName{Name: "my-instance"})
	a := NewAllInOne(jaeger)
	dep := a.Get()
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

func TestAllInOneGRPCPlugin(t *testing.T) {
	jaeger := v1.NewJaeger(types.NamespacedName{Name: "TestAllInOneGRPCPlugin"})
	jaeger.Spec.Storage.Type = v1.JaegerGRPCPluginStorage
	jaeger.Spec.Storage.GRPCPlugin.Image = "plugin/plugin:1.0"
	jaeger.Spec.Storage.Options = v1.NewOptions(map[string]interface{}{
		"grpc-storage-plugin.binary": "/plugin/plugin",
	})

	allinone := NewAllInOne(jaeger)
	dep := allinone.Get()

	assert.Equal(t, []corev1.Container{
		{
			Image: "plugin/plugin:1.0",
			Name:  "install-plugin",
			VolumeMounts: []corev1.VolumeMount{
				{
					Name:      "testallinonegrpcplugin-sampling-configuration-volume",
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

func TestAllInOneContainerSecurityContext(t *testing.T) {
	trueVar := true
	idVar := int64(1234)
	securityContextVar := corev1.SecurityContext{
		RunAsNonRoot: &trueVar,
		RunAsGroup:   &idVar,
		RunAsUser:    &idVar,
	}
	jaeger := v1.NewJaeger(types.NamespacedName{Name: "my-instance"})
	jaeger.Spec.AllInOne.ContainerSecurityContext = &securityContextVar

	a := NewAllInOne(jaeger)
	dep := a.Get()

	assert.Equal(t, securityContextVar, *dep.Spec.Template.Spec.Containers[0].SecurityContext)
}

func TestAllInOneContainerSecurityContextOverride(t *testing.T) {
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
	jaeger.Spec.AllInOne.ContainerSecurityContext = &overrideSecurityContextVar

	a := NewAllInOne(jaeger)
	dep := a.Get()

	assert.Equal(t, overrideSecurityContextVar, *dep.Spec.Template.Spec.Containers[0].SecurityContext)
}

func TestAllInOnePriorityClassName(t *testing.T) {
	priorityClassName := "test-class"
	jaeger := v1.NewJaeger(types.NamespacedName{Name: "my-instance"})
	jaeger.Spec.AllInOne.PriorityClassName = priorityClassName
	a := NewAllInOne(jaeger)
	dep := a.Get()
	assert.Equal(t, priorityClassName, dep.Spec.Template.Spec.PriorityClassName)
}

func TestAllInOnePrometheusMetricStorage(t *testing.T) {
	jaeger := v1.NewJaeger(types.NamespacedName{Name: "TestAllInOnePrometheusMetricStorage"})

	jaeger.Spec.AllInOne.MetricsStorage.Type = "prometheus"
	jaeger.Spec.AllInOne.MetricsStorage.ServerUrl = "http://prometheus:9090"

	d := NewAllInOne(jaeger).Get()

	assert.Equal(t, "prometheus", getEnvVarByName(d.Spec.Template.Spec.Containers[0].Env, "METRICS_STORAGE_TYPE").Value)
	assert.NotEmpty(t, getEnvVarByName(d.Spec.Template.Spec.Containers[0].Env, "PROMETHEUS_SERVER_URL").Value)
}

func getEnvVarByName(vars []corev1.EnvVar, name string) corev1.EnvVar {
	envVar := corev1.EnvVar{}
	for _, v := range vars {
		if v.Name == name {
			envVar = v
		}
	}
	return envVar
}
