package route

import (
	"fmt"
	"testing"

	"github.com/openshift/api/route/v1"
	"github.com/stretchr/testify/assert"

	"github.com/jaegertracing/jaeger-operator/pkg/apis/io/v1alpha1"
)

func TestQueryRoute(t *testing.T) {
	name := "TestQueryRoute"
	jaeger := v1alpha1.NewJaeger(name)
	route := NewQueryRoute(jaeger)

	dep := route.Get()

	assert.Contains(t, dep.Spec.To.Name, fmt.Sprintf("%s-query", name))
}

func TestQueryRouteDisabled(t *testing.T) {
	enabled := false
	name := "TestQueryRouteDisabled"
	jaeger := v1alpha1.NewJaeger(name)
	jaeger.Spec.Ingress.Enabled = &enabled
	route := NewQueryRoute(jaeger)

	dep := route.Get()

	assert.Nil(t, dep)
}

func TestQueryRouteEnabled(t *testing.T) {
	enabled := true
	name := "TestQueryRouteEnabled"
	jaeger := v1alpha1.NewJaeger(name)
	jaeger.Spec.Ingress.Enabled = &enabled
	route := NewQueryRoute(jaeger)

	dep := route.Get()

	assert.NotNil(t, dep)
}

func TestQueryRouteTerminationTypeWithOAuthProxy(t *testing.T) {
	jaeger := v1alpha1.NewJaeger("TestQueryRouteTerminationTypeWithOAuthProxy")
	jaeger.Spec.Ingress.Security = v1alpha1.IngressSecurityOAuthProxy
	route := NewQueryRoute(jaeger)

	r := route.Get()
	assert.Equal(t, v1.TLSTerminationReencrypt, r.Spec.TLS.Termination)
}

func TestQueryRouteTerminationTypeWithoutOAuthProxy(t *testing.T) {
	jaeger := v1alpha1.NewJaeger("TestQueryRouteTerminationTypeWithOAuthProxy")
	jaeger.Spec.Ingress.Security = v1alpha1.IngressSecurityNone
	route := NewQueryRoute(jaeger)

	r := route.Get()
	assert.Equal(t, v1.TLSTerminationEdge, r.Spec.TLS.Termination)
}
