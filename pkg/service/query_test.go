package service

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"k8s.io/apimachinery/pkg/util/intstr"

	"github.com/jaegertracing/jaeger-operator/pkg/apis/jaegertracing/v1"
)

func TestQueryServiceNameAndPorts(t *testing.T) {
	name := "TestQueryServiceNameAndPorts"
	selector := map[string]string{"app": "myapp", "jaeger": name, "jaeger-component": "query"}

	jaeger := v1.NewJaeger(name)
	svc := NewQueryService(jaeger, selector)

	assert.Equal(t, fmt.Sprintf("%s-query", name), svc.ObjectMeta.Name)
	assert.Len(t, svc.Spec.Ports, 1)
	assert.Equal(t, int32(16686), svc.Spec.Ports[0].Port)
	assert.Equal(t, intstr.FromInt(16686), svc.Spec.Ports[0].TargetPort)
	assert.Len(t, svc.Spec.ClusterIP, 0) // make sure we get a cluster IP
}

func TestQueryServiceNameAndPortsWithOAuthProxy(t *testing.T) {
	name := "TestQueryServiceNameAndPortsWithOAuthProxy"
	selector := map[string]string{"app": "myapp", "jaeger": name, "jaeger-component": "query"}

	jaeger := v1.NewJaeger(name)
	jaeger.Spec.Ingress.Security = v1.IngressSecurityOAuthProxy
	svc := NewQueryService(jaeger, selector)

	assert.Equal(t, fmt.Sprintf("%s-query", name), svc.ObjectMeta.Name)
	assert.Len(t, svc.Spec.Ports, 1)
	assert.Equal(t, int32(443), svc.Spec.Ports[0].Port)
	assert.Equal(t, intstr.FromInt(8443), svc.Spec.Ports[0].TargetPort)
}
