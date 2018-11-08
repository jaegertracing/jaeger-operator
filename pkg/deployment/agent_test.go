package deployment

import (
	"testing"

	"github.com/spf13/viper"
	"github.com/stretchr/testify/assert"

	"github.com/jaegertracing/jaeger-operator/pkg/apis/io/v1alpha1"
)

func setDefaults() {
	viper.SetDefault("jaeger-version", "1.6")
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
	jaeger := v1alpha1.NewJaeger("TestNewAgent")
	NewAgent(jaeger)
	assert.Contains(t, jaeger.Spec.Agent.Image, "jaeger-agent")
}

func TestDefaultAgentImage(t *testing.T) {
	viper.Set("jaeger-agent-image", "org/custom-agent-image")
	viper.Set("jaeger-version", "123")
	defer reset()

	jaeger := v1alpha1.NewJaeger("TestNewAgent")
	NewAgent(jaeger)
	assert.Equal(t, "org/custom-agent-image:123", jaeger.Spec.Agent.Image)
}

func TestGetDefaultAgentDeployment(t *testing.T) {
	jaeger := v1alpha1.NewJaeger("TestNewAgent")
	agent := NewAgent(jaeger)
	assert.Nil(t, agent.Get()) // it's not implemented yet
}

func TestGetSidecarDeployment(t *testing.T) {
	jaeger := v1alpha1.NewJaeger("TestNewAgent")
	jaeger.Spec.Agent.Strategy = "sidecar"
	agent := NewAgent(jaeger)
	assert.Nil(t, agent.Get()) // it's not implemented yet
}

func TestGetDaemonSetDeployment(t *testing.T) {
	jaeger := v1alpha1.NewJaeger("TestNewAgent")
	jaeger.Spec.Agent.Strategy = "daemonset"
	agent := NewAgent(jaeger)

	ds := agent.Get()
	assert.NotNil(t, ds)
}

func TestDaemonSetAgentAnnotations(t *testing.T) {
	jaeger := v1alpha1.NewJaeger("TestDaemonSetAgentAnnotations")
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
}
