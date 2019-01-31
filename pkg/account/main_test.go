package account

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/jaegertracing/jaeger-operator/pkg/apis/io/v1alpha1"
)

func TestWithSecurityNil(t *testing.T) {
	jaeger := v1alpha1.NewJaeger("TestWithOAuthProxyNil")
	assert.Equal(t, v1alpha1.IngressSecurityNone, jaeger.Spec.Ingress.Security)
	sas := Get(jaeger)
	assert.Len(t, sas, 1)
	assert.Equal(t, getMain(jaeger), sas[0])
}

func TestWithSecurityNone(t *testing.T) {
	jaeger := v1alpha1.NewJaeger("TestWithOAuthProxyFalse")
	jaeger.Spec.Ingress.Security = v1alpha1.IngressSecurityNone
	sas := Get(jaeger)
	assert.Len(t, sas, 1)
	assert.Equal(t, getMain(jaeger), sas[0])
}

func TestWithSecurityOAuthProxy(t *testing.T) {
	jaeger := v1alpha1.NewJaeger("TestWithOAuthProxyTrue")
	jaeger.Spec.Ingress.Security = v1alpha1.IngressSecurityOAuthProxy

	assert.Len(t, Get(jaeger), 2)
}

func TestJaegerName(t *testing.T) {
	jaeger := v1alpha1.NewJaeger("foo")
	assert.Equal(t, "foo", JaegerServiceAccountFor(jaeger))
}
