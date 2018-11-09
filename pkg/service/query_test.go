package service

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"k8s.io/apimachinery/pkg/util/intstr"

	"github.com/jaegertracing/jaeger-operator/pkg/apis/io/v1alpha1"
)

func TestQueryServiceNameAndPorts(t *testing.T) {
	name := "TestQueryServiceNameAndPorts"
	selector := map[string]string{"app": "myapp", "jaeger": name, "jaeger-component": "query"}

	jaeger := v1alpha1.NewJaeger(name)
	svc := NewQueryService(jaeger, selector)

	assert.Equal(t, fmt.Sprintf("%s-query", name), svc.ObjectMeta.Name)
	assert.Len(t, svc.Spec.Ports, 1)
	assert.Equal(t, int32(16686), svc.Spec.Ports[0].Port)
	assert.Equal(t, intstr.FromInt(16686), svc.Spec.Ports[0].TargetPort)
}

func TestQueryServiceNameAndPortsWithOAuthProxy(t *testing.T) {
	name := "TestQueryServiceNameAndPortsWithOAuthProxy"
	selector := map[string]string{"app": "myapp", "jaeger": name, "jaeger-component": "query"}

	jaeger := v1alpha1.NewJaeger(name)
	b := true
	jaeger.Spec.Ingress.OAuthProxy = &b
	svc := NewQueryService(jaeger, selector)

	assert.Equal(t, fmt.Sprintf("%s-query", name), svc.ObjectMeta.Name)
	assert.Len(t, svc.Spec.Ports, 1)
	assert.Equal(t, int32(443), svc.Spec.Ports[0].Port)
	assert.Equal(t, intstr.FromInt(8443), svc.Spec.Ports[0].TargetPort)
}
