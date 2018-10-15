package deployment

import (
	"testing"

	"github.com/spf13/viper"
	"github.com/stretchr/testify/assert"

	"github.com/jaegertracing/jaeger-operator/pkg/apis/io/v1alpha1"
)

const testNewAgent = "TestNewAgent"

func setDefaults() {
	viper.SetDefault(versionKey, versionValue)
	viper.SetDefault(agentImageKey, "jaegertracing/jaeger-agent")
}

func init() {
	setDefaults()
}

func reset() {
	viper.Reset()
	setDefaults()
}

func TestNewAgent(t *testing.T) {
	jaeger := v1alpha1.NewJaeger(testNewAgent)
	NewAgent(jaeger)
	assert.Contains(t, jaeger.Spec.Agent.Image, agent)
}

func TestDefaultAgentImage(t *testing.T) {
	viper.Set(agentImageKey, "org/custom-agent-image")
	viper.Set(versionKey, "123")
	defer reset()

	jaeger := v1alpha1.NewJaeger(testNewAgent)
	NewAgent(jaeger)
	assert.Equal(t, "org/custom-agent-image:123", jaeger.Spec.Agent.Image)
}

func TestGetDefaultAgentDeployment(t *testing.T) {
	jaeger := v1alpha1.NewJaeger(testNewAgent)
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
	jaeger.Spec.Agent.Strategy = daemonSetStrategy
	agent := NewAgent(jaeger)
	assert.NotNil(t, agent.Get())
}
