package deployment

import (
	"fmt"
	"strings"
	"testing"

	"github.com/spf13/viper"
	"github.com/stretchr/testify/assert"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/intstr"

	v1 "github.com/jaegertracing/jaeger-operator/apis/v1"
	"github.com/jaegertracing/jaeger-operator/pkg/autodetect"
	"github.com/jaegertracing/jaeger-operator/pkg/config/ca"
	"github.com/jaegertracing/jaeger-operator/pkg/util"
	"github.com/jaegertracing/jaeger-operator/pkg/version"
)

func setDefaults() {
	viper.SetDefault("jaeger-agent-image", "jaegertracing/jaeger-agent")
}

func init() {
	setDefaults()
}

func reset() {
	viper.Reset()
	setDefaults()
}

func TestNewAgent(t *testing.T) {
	jaeger := v1.NewJaeger(types.NamespacedName{Name: "my-instance"})
	jaeger.Spec.Agent.Strategy = "daemonset"

	d := NewAgent(jaeger).Get()
	assert.Empty(t, jaeger.Spec.Agent.Image)
	assert.Contains(t, d.Spec.Template.Spec.Containers[0].Image, "jaeger-agent")
}

func TestDefaultAgentImage(t *testing.T) {
	viper.Set("jaeger-agent-image", "org/custom-agent-image")
	defer reset()

	jaeger := v1.NewJaeger(types.NamespacedName{Name: "my-instance"})
	jaeger.Spec.Agent.Strategy = "daemonset"

	d := NewAgent(jaeger).Get()
	assert.Empty(t, jaeger.Spec.Agent.Image)
	assert.Equal(t, "org/custom-agent-image:"+version.Get().Agent, d.Spec.Template.Spec.Containers[0].Image)
}

func TestGetDefaultAgentDeployment(t *testing.T) {
	jaeger := v1.NewJaeger(types.NamespacedName{Name: "my-instance"})
	agent := NewAgent(jaeger)
	assert.Nil(t, agent.Get()) // it's not implemented yet
}

func TestGetSidecarDeployment(t *testing.T) {
	jaeger := v1.NewJaeger(types.NamespacedName{Name: "my-instance"})
	jaeger.Spec.Agent.Strategy = "sidecar"
	agent := NewAgent(jaeger)
	assert.Nil(t, agent.Get()) // it's not implemented yet
}

func TestGetDaemonSetDeployment(t *testing.T) {
	jaeger := v1.NewJaeger(types.NamespacedName{Name: "my-instance"})
	jaeger.Spec.Agent.Strategy = "daemonset"
	agent := NewAgent(jaeger)

	ds := agent.Get()
	assert.NotNil(t, ds)
}

func TestDaemonSetAgentAnnotations(t *testing.T) {
	jaeger := v1.NewJaeger(types.NamespacedName{Name: "my-instance"})
	jaeger.Spec.Agent.Strategy = "daemonset"
	jaeger.Spec.Annotations = map[string]string{
		"name":  "operator",
		"hello": "jaeger",
	}
	jaeger.Spec.Agent.Annotations = map[string]string{
		"hello":                "world", // Override top level annotation
		"prometheus.io/scrape": "false", // Override implicit value
	}

	agent := NewAgent(jaeger)
	dep := agent.Get()

	assert.Equal(t, "operator", dep.Spec.Template.Annotations["name"])
	assert.Equal(t, "false", dep.Spec.Template.Annotations["sidecar.istio.io/inject"])
	assert.Equal(t, "world", dep.Spec.Template.Annotations["hello"])
	assert.Equal(t, "false", dep.Spec.Template.Annotations["prometheus.io/scrape"])
	assert.Equal(t, "disabled", dep.Spec.Template.Annotations["linkerd.io/inject"])
}

func TestDaemonSetAgentLabels(t *testing.T) {
	jaeger := v1.NewJaeger(types.NamespacedName{Name: "my-instance"})
	jaeger.Spec.Agent.Strategy = "daemonset"
	jaeger.Spec.Labels = map[string]string{
		"name":  "operator",
		"hello": "jaeger",
	}
	jaeger.Spec.Agent.Labels = map[string]string{
		"hello":   "world", // Override top level label
		"another": "false",
	}

	agent := NewAgent(jaeger)
	dep := agent.Get()

	assert.Equal(t, "operator", dep.Spec.Template.Labels["name"])
	assert.Equal(t, "world", dep.Spec.Template.Labels["hello"])
	assert.Equal(t, "false", dep.Spec.Template.Labels["another"])
	assert.Equal(t, "operator", dep.Spec.Selector.MatchLabels["name"])
	assert.Equal(t, "world", dep.Spec.Selector.MatchLabels["hello"])
	assert.Equal(t, "false", dep.Spec.Selector.MatchLabels["another"])
}

func TestDaemonSetAgentResources(t *testing.T) {
	jaeger := v1.NewJaeger(types.NamespacedName{Name: "my-instance"})
	jaeger.Spec.Agent.Strategy = "daemonset"
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
	jaeger.Spec.Agent.Resources = corev1.ResourceRequirements{
		Limits: corev1.ResourceList{
			corev1.ResourceLimitsCPU:    *resource.NewQuantity(2048, resource.BinarySI),
			corev1.ResourceLimitsMemory: *resource.NewQuantity(123, resource.DecimalSI),
		},
		Requests: corev1.ResourceList{
			corev1.ResourceRequestsCPU:    *resource.NewQuantity(2048, resource.BinarySI),
			corev1.ResourceRequestsMemory: *resource.NewQuantity(123, resource.DecimalSI),
		},
	}

	agent := NewAgent(jaeger)
	dep := agent.Get()

	assert.Equal(t, *resource.NewQuantity(2048, resource.BinarySI), dep.Spec.Template.Spec.Containers[0].Resources.Limits[corev1.ResourceLimitsCPU])
	assert.Equal(t, *resource.NewQuantity(2048, resource.BinarySI), dep.Spec.Template.Spec.Containers[0].Resources.Requests[corev1.ResourceRequestsCPU])
	assert.Equal(t, *resource.NewQuantity(123, resource.DecimalSI), dep.Spec.Template.Spec.Containers[0].Resources.Limits[corev1.ResourceLimitsMemory])
	assert.Equal(t, *resource.NewQuantity(123, resource.DecimalSI), dep.Spec.Template.Spec.Containers[0].Resources.Requests[corev1.ResourceRequestsMemory])
	assert.Equal(t, *resource.NewQuantity(512, resource.DecimalSI), dep.Spec.Template.Spec.Containers[0].Resources.Limits[corev1.ResourceLimitsEphemeralStorage])
	assert.Equal(t, *resource.NewQuantity(512, resource.DecimalSI), dep.Spec.Template.Spec.Containers[0].Resources.Requests[corev1.ResourceRequestsEphemeralStorage])
}

func TestAgentLabels(t *testing.T) {
	jaeger := v1.NewJaeger(types.NamespacedName{Name: "my-instance"})
	jaeger.Spec.Agent.Strategy = "daemonset"
	a := NewAgent(jaeger)
	dep := a.Get()
	assert.Equal(t, "jaeger-operator", dep.Spec.Template.Labels["app.kubernetes.io/managed-by"])
	assert.Equal(t, "agent", dep.Spec.Template.Labels["app.kubernetes.io/component"])
	assert.Equal(t, a.jaeger.Name, dep.Spec.Template.Labels["app.kubernetes.io/instance"])
	assert.Equal(t, fmt.Sprintf("%s-agent", a.jaeger.Name), dep.Spec.Template.Labels["app.kubernetes.io/name"])
}

func TestAgentOrderOfArguments(t *testing.T) {
	jaeger := v1.NewJaeger(types.NamespacedName{Name: "my-instance"})
	jaeger.Spec.Agent.Strategy = "daemonset"
	jaeger.Spec.Agent.Options = v1.NewOptions(map[string]interface{}{
		"b-option": "b-value",
		"a-option": "a-value",
		"c-option": "c-value",
	})

	a := NewAgent(jaeger)
	dep := a.Get()

	assert.Len(t, dep.Spec.Template.Spec.Containers, 1)
	assert.Len(t, dep.Spec.Template.Spec.Containers[0].Args, 4)
	assert.True(t, strings.HasPrefix(dep.Spec.Template.Spec.Containers[0].Args[0], "--a-option"))
	assert.True(t, strings.HasPrefix(dep.Spec.Template.Spec.Containers[0].Args[1], "--b-option"))
	assert.True(t, strings.HasPrefix(dep.Spec.Template.Spec.Containers[0].Args[2], "--c-option"))

	// the following are added automatically
	assert.True(t, strings.HasPrefix(dep.Spec.Template.Spec.Containers[0].Args[3], "--reporter.grpc.host-port"))
}

func TestAgentCustomReporterPort(t *testing.T) {
	jaeger := v1.NewJaeger(types.NamespacedName{Name: "my-instance"})
	jaeger.Spec.Agent.Strategy = "daemonset"
	jaeger.Spec.Agent.Options = v1.NewOptions(map[string]interface{}{
		"reporter.grpc.host-port": "collector:5000",
	})

	a := NewAgent(jaeger)
	dep := a.Get()

	assert.Equal(t, "--reporter.grpc.host-port=collector:5000", dep.Spec.Template.Spec.Containers[0].Args[0])
}

func TestAgentArgumentsOpenshiftTLS(t *testing.T) {
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
				"--reporter.grpc.host-port=dns:///my-instance-collector-headless.test:14250",
				"--reporter.grpc.tls.enabled=true",
				"--reporter.grpc.tls.ca=" + ca.ServiceCAPath,
				"--reporter.grpc.tls.server-name=my-instance-collector-headless.test.svc.cluster.local",
			},
		},
		{
			name: "Custom CA",
			options: v1.NewOptions(map[string]interface{}{
				"a-option":                  "a-value",
				"reporter.grpc.tls.enabled": "true",
				"reporter.grpc.tls.ca":      "/my/custom/ca",
			}),
			expectedArgs: []string{
				"--a-option=a-value",
				"--reporter.grpc.host-port=dns:///my-instance-collector-headless.test:14250",
				"--reporter.grpc.tls.enabled=true",
				"--reporter.grpc.tls.ca=/my/custom/ca",
			},
		},
		{
			name: "Explicit disable TLS",
			options: v1.NewOptions(map[string]interface{}{
				"a-option":                  "a-value",
				"reporter.grpc.tls.enabled": "false",
			}),
			expectedArgs: []string{
				"--a-option=a-value",
				"--reporter.grpc.host-port=dns:///my-instance-collector-headless.test:14250",
				"--reporter.grpc.tls.enabled=false",
			},
			nonExpectedArgs: []string{
				"--reporter.grpc.tls.enabled=true",
			},
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			jaeger := v1.NewJaeger(types.NamespacedName{
				Name:      "my-instance",
				Namespace: "test",
			})
			jaeger.Spec.Agent.Strategy = "daemonset"
			jaeger.Spec.Agent.Options = tt.options

			a := NewAgent(jaeger)
			dep := a.Get()

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

			assert.Len(t, dep.Spec.Template.Spec.Volumes, 2)
			assert.Len(t, dep.Spec.Template.Spec.Containers[0].VolumeMounts, 2)
		})
	}
}

func TestAgentImagePullSecrets(t *testing.T) {
	jaeger := v1.NewJaeger(types.NamespacedName{Name: "TestAllInOneImagePullSecrets"})
	const pullSecret = "mysecret"
	jaeger.Spec.Agent.Strategy = "daemonset"
	jaeger.Spec.Agent.ImagePullSecrets = []corev1.LocalObjectReference{
		{
			Name: pullSecret,
		},
	}

	agent := NewAgent(jaeger)
	dep := agent.Get()

	assert.Equal(t, pullSecret, dep.Spec.Template.Spec.ImagePullSecrets[0].Name)
}

func TestAgentImagePullPolicy(t *testing.T) {
	jaeger := v1.NewJaeger(types.NamespacedName{Name: "TestAgentImagePullPolicy"})
	const pullPolicy = corev1.PullPolicy("Always")
	jaeger.Spec.Agent.Strategy = "daemonset"
	jaeger.Spec.ImagePullPolicy = corev1.PullPolicy("Always")

	agent := NewAgent(jaeger)
	dep := agent.Get()

	assert.Equal(t, pullPolicy, dep.Spec.Template.Spec.Containers[0].ImagePullPolicy)
}

func TestAgentServiceLinks(t *testing.T) {
	jaeger := v1.NewJaeger(types.NamespacedName{Name: "my-instance"})
	jaeger.Spec.Agent.Strategy = "daemonset"
	a := NewAgent(jaeger)
	dep := a.Get()
	falseVar := false
	assert.Equal(t, &falseVar, dep.Spec.Template.Spec.EnableServiceLinks)
	assert.Equal(t, falseVar, dep.Spec.Template.Spec.HostNetwork)
}

func TestAgentHostNetwork(t *testing.T) {
	trueVar := true
	jaeger := v1.NewJaeger(types.NamespacedName{Name: "my-instance"})
	jaeger.Spec.Agent.Strategy = "daemonset"
	jaeger.Spec.Agent.HostNetwork = &trueVar
	a := NewAgent(jaeger)
	dep := a.Get()
	assert.Equal(t, trueVar, dep.Spec.Template.Spec.HostNetwork)
}

func TestAgentDNSPolicyWithHostNetwork(t *testing.T) {
	trueVar := true
	jaeger := v1.NewJaeger(types.NamespacedName{Name: "my-instance"})
	jaeger.Spec.Agent.Strategy = "daemonset"
	jaeger.Spec.Agent.HostNetwork = &trueVar
	a := NewAgent(jaeger)
	dep := a.Get()
	assert.Equal(t, trueVar, dep.Spec.Template.Spec.HostNetwork)
	assert.Equal(t, corev1.DNSClusterFirstWithHostNet, dep.Spec.Template.Spec.DNSPolicy)
}

func TestAgentPriorityClassName(t *testing.T) {
	priorityClassName := "test-class"
	jaeger := v1.NewJaeger(types.NamespacedName{Name: "my-instance"})
	jaeger.Spec.Agent.Strategy = "daemonset"
	jaeger.Spec.Agent.PriorityClassName = priorityClassName
	a := NewAgent(jaeger)
	dep := a.Get()
	assert.Equal(t, priorityClassName, dep.Spec.Template.Spec.PriorityClassName)
}

func TestAgentLivenessProbe(t *testing.T) {
	livenessProbe := &corev1.Probe{
		ProbeHandler: corev1.ProbeHandler{
			HTTPGet: &corev1.HTTPGetAction{
				Path: "/",
				Port: intstr.FromInt(int(14271)),
			},
		},
		InitialDelaySeconds: 60,
		PeriodSeconds:       60,
		FailureThreshold:    60,
	}
	jaeger := v1.NewJaeger(types.NamespacedName{Name: "my-instance"})
	jaeger.Spec.Agent.Strategy = "daemonset"
	jaeger.Spec.Agent.LivenessProbe = livenessProbe
	a := NewAgent(jaeger)
	dep := a.Get()
	assert.Equal(t, livenessProbe, dep.Spec.Template.Spec.Containers[0].LivenessProbe)
}

func TestAgentEmptyEmptyLivenessProbe(t *testing.T) {
	jaeger := v1.NewJaeger(types.NamespacedName{Name: "my-instance"})
	jaeger.Spec.Agent.Strategy = "daemonset"
	a := NewAgent(jaeger)
	dep := a.Get()
	assert.Equal(t, &corev1.Probe{
		ProbeHandler: corev1.ProbeHandler{
			HTTPGet: &corev1.HTTPGetAction{
				Path: "/",
				Port: intstr.FromInt(int(14271)),
			},
		},
		InitialDelaySeconds: 5,
		PeriodSeconds:       15,
		FailureThreshold:    5,
	}, dep.Spec.Template.Spec.Containers[0].LivenessProbe)
}

func TestAgentContainerSecurityContext(t *testing.T) {
	trueVar := true
	idVar := int64(1234)
	securityContextVar := corev1.SecurityContext{
		RunAsNonRoot: &trueVar,
		RunAsGroup:   &idVar,
		RunAsUser:    &idVar,
	}
	jaeger := v1.NewJaeger(types.NamespacedName{Name: "my-instance"})
	jaeger.Spec.Agent.Strategy = "daemonset"
	jaeger.Spec.Agent.ContainerSecurityContext = &securityContextVar

	a := NewAgent(jaeger)
	dep := a.Get()

	assert.Equal(t, securityContextVar, *dep.Spec.Template.Spec.Containers[0].SecurityContext)
}

func TestAgentContainerSecurityContextOverride(t *testing.T) {
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
	jaeger.Spec.Agent.Strategy = "daemonset"
	jaeger.Spec.ContainerSecurityContext = &securityContextVar
	jaeger.Spec.Agent.ContainerSecurityContext = &overrideSecurityContextVar

	a := NewAgent(jaeger)
	dep := a.Get()

	assert.Equal(t, overrideSecurityContextVar, *dep.Spec.Template.Spec.Containers[0].SecurityContext)
}
