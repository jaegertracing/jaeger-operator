package service

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"k8s.io/apimachinery/pkg/types"

	v1 "github.com/jaegertracing/jaeger-operator/pkg/apis/jaegertracing/v1"
)

func TestAgentServiceNameAndPorts(t *testing.T) {
	name := "TestAgentServiceNameAndPorts"
	selector := map[string]string{"app": "myapp", "jaeger": name, "jaeger-component": "agent"}

	jaeger := v1.NewJaeger(types.NamespacedName{Name: name})
	svc := NewAgentService(jaeger, selector)
	assert.Equal(t, "testagentservicenameandports-agent", svc.ObjectMeta.Name)

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
