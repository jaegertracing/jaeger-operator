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

func TestGetSicedarDeployment(t *testing.T) {
	jaeger := v1alpha1.NewJaeger("TestNewAgent")
	jaeger.Spec.Agent.Strategy = "sidecar"
	agent := NewAgent(jaeger)
	assert.Nil(t, agent.Get()) // it's not implemented yet
}

func TestGetDaemonSetDeployment(t *testing.T) {
	jaeger := v1alpha1.NewJaeger("TestNewAgent")
	jaeger.Spec.Agent.Strategy = "daemonset"
	agent := NewAgent(jaeger)
	assert.NotNil(t, agent.Get())
}

func TestInjectSidecar(t *testing.T) {
	jaeger := v1alpha1.NewJaeger("TestNewAgent")
	dep := NewQuery(jaeger).Get()
	agent := NewAgent(jaeger)

	dep = agent.InjectSidecar(*dep)

	assert.Len(t, dep.Spec.Template.Spec.Containers, 2)
	assert.Contains(t, dep.Spec.Template.Spec.Containers[1].Image, "jaeger-agent")
}
