package deployment

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/jaegertracing/jaeger-operator/pkg/apis/io/v1alpha1"
)

func TestNewAgent(t *testing.T) {
	jaeger := v1alpha1.NewJaeger("TestNewAgent")
	NewAgent(jaeger)
	assert.Contains(t, jaeger.Spec.Agent.Image, "jaeger-agent")
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
	assert.Nil(t, agent.Get()) // it's not implemented yet
}

func TestInjectSidecar(t *testing.T) {
	jaeger := v1alpha1.NewJaeger("TestNewAgent")
	dep := NewQuery(jaeger).Get()
	agent := NewAgent(jaeger)

	dep = agent.InjectSidecar(*dep)

	assert.Len(t, dep.Spec.Template.Spec.Containers, 2)
	assert.Contains(t, dep.Spec.Template.Spec.Containers[1].Image, "jaeger-agent")
}
