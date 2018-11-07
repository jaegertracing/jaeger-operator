package ingress

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/jaegertracing/jaeger-operator/pkg/apis/io/v1alpha1"
)

func TestQueryIngress(t *testing.T) {
	name := "TestQueryIngress"
	jaeger := v1alpha1.NewJaeger(name)
	ingress := NewQueryIngress(jaeger)

	dep := ingress.Get()

	assert.Contains(t, dep.Spec.Backend.ServiceName, fmt.Sprintf("%s-query", name))
}

func TestQueryIngressDisabled(t *testing.T) {
	enabled := false
	name := "TestQueryIngressDisabled"
	jaeger := v1alpha1.NewJaeger(name)
	jaeger.Spec.Ingress.Enabled = &enabled
	ingress := NewQueryIngress(jaeger)

	dep := ingress.Get()

	assert.Nil(t, dep)
}

func TestQueryIngressEnabled(t *testing.T) {
	enabled := true
	name := "TestQueryIngressEnabled"
	jaeger := v1alpha1.NewJaeger(name)
	jaeger.Spec.Ingress.Enabled = &enabled
	ingress := NewQueryIngress(jaeger)

	dep := ingress.Get()

	assert.NotNil(t, dep)
}
