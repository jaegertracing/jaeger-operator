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

	v1 "github.com/jaegertracing/jaeger-operator/pkg/apis/jaegertracing/v1"
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
	assert.Equal(t, "org/custom-agent-image:0.0.0", d.Spec.Template.Spec.Containers[0].Image)
}

func TestGetDefaultAgentDeployment(t *testing.T) {
	jaeger := v1.NewJaeger(types.NamespacedName{Name: "TestNewAgent"})
	agent := NewAgent(jaeger)
	assert.Nil(t, agent.Get()) // it's not implemented yet
}

func TestGetSidecarDeployment(t *testing.T) {
	jaeger := v1.NewJaeger(types.NamespacedName{Name: "TestNewAgent"})
	jaeger.Spec.Agent.Strategy = "sidecar"
	agent := NewAgent(jaeger)
	assert.Nil(t, agent.Get()) // it's not implemented yet
}

func TestGetDaemonSetDeployment(t *testing.T) {
	jaeger := v1.NewJaeger(types.NamespacedName{Name: "TestNewAgent"})
	jaeger.Spec.Agent.Strategy = "daemonset"
	agent := NewAgent(jaeger)

	ds := agent.Get()
	assert.NotNil(t, ds)
}

func TestDaemonSetAgentAnnotations(t *testing.T) {
	jaeger := v1.NewJaeger(types.NamespacedName{Name: "TestDaemonSetAgentAnnotations"})
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
	jaeger := v1.NewJaeger(types.NamespacedName{Name: "TestDaemonSetAgentLabels"})
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
}

func TestDaemonSetAgentResources(t *testing.T) {
	jaeger := v1.NewJaeger(types.NamespacedName{Name: "TestDaemonSetAgentResources"})
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
	jaeger := v1.NewJaeger(types.NamespacedName{Name: "TestAgentLabels"})
	jaeger.Spec.Agent.Strategy = "daemonset"
	a := NewAgent(jaeger)
	dep := a.Get()
	assert.Equal(t, "jaeger-operator", dep.Spec.Template.Labels["app.kubernetes.io/managed-by"])
	assert.Equal(t, "agent", dep.Spec.Template.Labels["app.kubernetes.io/component"])
	assert.Equal(t, a.jaeger.Name, dep.Spec.Template.Labels["app.kubernetes.io/instance"])
	assert.Equal(t, fmt.Sprintf("%s-agent", a.jaeger.Name), dep.Spec.Template.Labels["app.kubernetes.io/name"])
}

func TestAgentDNSPolicyAndHostNetwork(t *testing.T) {
	jaeger := v1.NewJaeger(types.NamespacedName{Name: "my-instance"})
	jaeger.Spec.Agent.Strategy = "daemonset"
	type dnsPolicyAndHostNetwork struct {
		JaegerDNSPolicy   corev1.DNSPolicy
		JaegerHostNetwork bool
		AgentDNSPolicy    corev1.DNSPolicy
		AgentHostNetwork  bool
	}
	type result struct {
		DNSPolicy   corev1.DNSPolicy
		HostNetwork bool
	}

	tests := []dnsPolicyAndHostNetwork{
		{}, // default
		{
			AgentDNSPolicy:   corev1.DNSDefault,
			AgentHostNetwork: true,
		},
		{
			JaegerDNSPolicy:   corev1.DNSClusterFirstWithHostNet,
			JaegerHostNetwork: true,
			AgentDNSPolicy:    corev1.DNSDefault,
			AgentHostNetwork:  false,
		}, // merge
	}
	expected := []result{
		{
			DNSPolicy:   "",
			HostNetwork: false,
		},
		{
			DNSPolicy:   corev1.DNSDefault,
			HostNetwork: true,
		},
		{
			DNSPolicy:   corev1.DNSDefault,
			HostNetwork: true,
		},
	}

	for idx, config := range tests {
		jaeger.Spec.DNSPolicy = config.JaegerDNSPolicy
		jaeger.Spec.HostNetwork = config.JaegerHostNetwork
		jaeger.Spec.Agent.DNSPolicy = config.AgentDNSPolicy
		jaeger.Spec.Agent.HostNetwork = config.AgentHostNetwork
		a := NewAgent(jaeger)
		dep := a.Get()
		assert.Equal(t, expected[idx].DNSPolicy, dep.Spec.Template.Spec.DNSPolicy)
		assert.Equal(t, expected[idx].HostNetwork, dep.Spec.Template.Spec.HostNetwork)
	}
}

func TestAgentOrderOfArguments(t *testing.T) {
	jaeger := v1.NewJaeger(types.NamespacedName{Name: "TestAgentOrderOfArguments"})
	jaeger.Spec.Agent.Strategy = "daemonset"
	jaeger.Spec.Agent.Options = v1.NewOptions(map[string]interface{}{
		"b-option": "b-value",
		"a-option": "a-value",
		"c-option": "c-value",
	})

	a := NewAgent(jaeger)
	dep := a.Get()

	assert.Len(t, dep.Spec.Template.Spec.Containers, 1)
	assert.Len(t, dep.Spec.Template.Spec.Containers[0].Args, 5)
	assert.True(t, strings.HasPrefix(dep.Spec.Template.Spec.Containers[0].Args[0], "--a-option"))
	assert.True(t, strings.HasPrefix(dep.Spec.Template.Spec.Containers[0].Args[1], "--b-option"))
	assert.True(t, strings.HasPrefix(dep.Spec.Template.Spec.Containers[0].Args[2], "--c-option"))

	// the following are added automatically
	assert.True(t, strings.HasPrefix(dep.Spec.Template.Spec.Containers[0].Args[3], "--reporter.grpc.host-port"))
	assert.True(t, strings.HasPrefix(dep.Spec.Template.Spec.Containers[0].Args[4], "--reporter.type"))
}

func TestAgentOverrideReporterType(t *testing.T) {
	jaeger := v1.NewJaeger(types.NamespacedName{Name: "TestAgentOrderOfArguments"})
	jaeger.Spec.Agent.Strategy = "daemonset"
	jaeger.Spec.Agent.Options = v1.NewOptions(map[string]interface{}{
		"reporter.type":             "thrift",
		"reporter.thrift.host-port": "collector:14267",
	})

	a := NewAgent(jaeger)
	dep := a.Get()

	assert.Equal(t, "--reporter.thrift.host-port=collector:14267", dep.Spec.Template.Spec.Containers[0].Args[0])
	assert.Equal(t, "--reporter.type=thrift", dep.Spec.Template.Spec.Containers[0].Args[1])
}
