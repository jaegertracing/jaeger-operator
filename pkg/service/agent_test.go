package service

import (
	"testing"

	"github.com/stretchr/testify/assert"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"

	v1 "github.com/jaegertracing/jaeger-operator/apis/v1"
)

func TestAgentServiceNameAndPorts(t *testing.T) {
	name := "TestAgentServiceNameAndPorts"
	selector := map[string]string{"app": "myapp", "jaeger": name, "jaeger-component": "agent"}

	jaeger := v1.NewJaeger(types.NamespacedName{Name: name})
	svc := NewAgentService(jaeger, selector)
	assert.Equal(t, "testagentservicenameandports-agent", svc.ObjectMeta.Name)

	ports := map[int32]bool{
		5775:  false,
		5778:  false,
		6831:  false,
		6832:  false,
		14271: false,
	}

	for _, port := range svc.Spec.Ports {
		ports[port.Port] = true
		switch port.Port {
		case 5775, 6831, 6832:
			assert.Equal(t, corev1.ProtocolUDP, port.Protocol, "Expected port %v to be UDP, but wasn't", port)
		}
	}

	for k, v := range ports {
		assert.True(t, v, "Expected port %v to be specified, but wasn't", k)
	}
}
