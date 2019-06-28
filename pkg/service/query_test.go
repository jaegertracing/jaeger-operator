package service

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/intstr"

	v1 "github.com/jaegertracing/jaeger-operator/pkg/apis/jaegertracing/v1"
)

func TestQueryServiceNameAndPorts(t *testing.T) {
	name := "TestQueryServiceNameAndPorts"
	selector := map[string]string{"app": "myapp", "jaeger": name, "jaeger-component": "query"}

	jaeger := v1.NewJaeger(types.NamespacedName{Name: name})
	svc := NewQueryService(jaeger, selector)

	assert.Equal(t, "testqueryservicenameandports-query", svc.ObjectMeta.Name)
	assert.Len(t, svc.Spec.Ports, 1)
	assert.Equal(t, int32(16686), svc.Spec.Ports[0].Port)
	assert.Equal(t, intstr.FromInt(16686), svc.Spec.Ports[0].TargetPort)
	assert.Len(t, svc.Spec.ClusterIP, 0) // make sure we get a cluster IP
}

func TestQueryDottedServiceName(t *testing.T) {
	name := "TestQueryDottedServiceName.With.Dots"
	selector := map[string]string{"app": "myapp", "jaeger": name, "jaeger-component": "query"}

	jaeger := v1.NewJaeger(types.NamespacedName{Name: name})
	svc := NewQueryService(jaeger, selector)

	assert.Equal(t, "testquerydottedservicename-with-dots-query", svc.ObjectMeta.Name)
}

func TestQueryServiceNameAndPortsWithOAuthProxy(t *testing.T) {
	name := "TestQueryServiceNameAndPortsWithOAuthProxy"
	selector := map[string]string{"app": "myapp", "jaeger": name, "jaeger-component": "query"}

	jaeger := v1.NewJaeger(types.NamespacedName{Name: name})
	jaeger.Spec.Ingress.Security = v1.IngressSecurityOAuthProxy
	svc := NewQueryService(jaeger, selector)

	assert.Equal(t, "testqueryservicenameandportswithoauthproxy-query", svc.ObjectMeta.Name)
	assert.Len(t, svc.Spec.Ports, 1)
	assert.Equal(t, int32(443), svc.Spec.Ports[0].Port)
	assert.Equal(t, intstr.FromInt(8443), svc.Spec.Ports[0].TargetPort)
}
