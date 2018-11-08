package account

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/jaegertracing/jaeger-operator/pkg/apis/io/v1alpha1"
)

func TestWithOAuthProxyNil(t *testing.T) {
	jaeger := v1alpha1.NewJaeger("TestWithOAuthProxyNil")
	assert.Nil(t, jaeger.Spec.Ingress.OAuthProxy)
	assert.Len(t, Get(jaeger), 0)
}

func TestWithOAuthProxyFalse(t *testing.T) {
	b := false
	jaeger := v1alpha1.NewJaeger("TestWithOAuthProxyFalse")
	jaeger.Spec.Ingress.OAuthProxy = &b

	assert.Len(t, Get(jaeger), 0)
}

func TestWithOAuthProxyTrue(t *testing.T) {
	b := true
	jaeger := v1alpha1.NewJaeger("TestWithOAuthProxyTrue")
	jaeger.Spec.Ingress.OAuthProxy = &b

	assert.Len(t, Get(jaeger), 1)
}
