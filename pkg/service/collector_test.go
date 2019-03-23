package service

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/jaegertracing/jaeger-operator/pkg/apis/jaegertracing/v1"
)

func TestCollectorServiceNameAndPorts(t *testing.T) {
	name := "TestCollectorServiceNameAndPorts"
	selector := map[string]string{"app": "myapp", "jaeger": name, "jaeger-component": "collector"}

	jaeger := v1.NewJaeger(name)
	svcs := NewCollectorServices(jaeger, selector)

	assert.Equal(t, svcs[0].Name, fmt.Sprintf("%s-collector-headless", name))
	assert.Equal(t, svcs[1].Name, fmt.Sprintf("%s-collector", name))

	ports := map[int32]bool{
		9411:  false,
		14250: false,
		14267: false,
		14268: false,
	}

	svc := svcs[0]
	for _, port := range svc.Spec.Ports {
		ports[port.Port] = true
	}

	for k, v := range ports {
		assert.Equal(t, v, true, "Expected port %v to be specified, but wasn't", k)
	}

	// we ensure the ports are the same for both services
	assert.Equal(t, svcs[0].Spec.Ports, svcs[1].Spec.Ports)
}

func TestCollectorServiceWithClusterIPEmptyAndNone(t *testing.T) {
	name := "TestCollectorServiceWithClusterIP"
	selector := map[string]string{"app": "myapp", "jaeger": name, "jaeger-component": "collector"}

	jaeger := v1.NewJaeger(name)
	svcs := NewCollectorServices(jaeger, selector)

	// we want two services, one headless (load balanced by the client, possibly via DNS)
	// and one with a cluster IP (load balanced by kube-proxy)
	assert.Len(t, svcs, 2)
	assert.NotEqual(t, svcs[0].Name, svcs[1].Name) // they can't have the same name
	assert.Equal(t, "None", svcs[0].Spec.ClusterIP)
	assert.Len(t, svcs[1].Spec.ClusterIP, 0)
}
