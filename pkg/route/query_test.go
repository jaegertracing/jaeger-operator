package route

import (
	"fmt"
	"testing"

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
