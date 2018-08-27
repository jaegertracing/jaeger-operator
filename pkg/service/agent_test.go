package service

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/jaegertracing/jaeger-operator/pkg/apis/io/v1alpha1"
)

func TestAgentServiceNameAndPorts(t *testing.T) {
	name := "TestAgentServiceNameAndPorts"
	selector := map[string]string{"app": "myapp", "jaeger": name, "jaeger-component": "agent"}

	jaeger := v1alpha1.NewJaeger(name)
	svc := NewAgentService(jaeger, selector)
	assert.Equal(t, svc.ObjectMeta.Name, fmt.Sprintf("%s-agent", name))

	ports := map[int32]bool{
		5775: false,
		5778: false,
		6831: false,
		6832: false,
	}

	for _, port := range svc.Spec.Ports {
		ports[port.Port] = true
	}

	for k, v := range ports {
		assert.Equal(t, v, true, "Expected port %v to be specified, but wasn't", k)
	}

}
