package account

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/jaegertracing/jaeger-operator/pkg/apis/io/v1alpha1"
)

func TestOAuthRedirectReference(t *testing.T) {
	b := true
	jaeger := v1alpha1.NewJaeger("TestOAuthRedirectReference")
	jaeger.Spec.Ingress.OAuthProxy = &b

	assert.Contains(t, getOAuthRedirectReference(jaeger), jaeger.Name)
}

func TestOAuthProxy(t *testing.T) {
	b := true
	jaeger := v1alpha1.NewJaeger("TestOAuthProxy")
	jaeger.Spec.Ingress.OAuthProxy = &b

	assert.Equal(t, OAuthProxy(jaeger).Name, fmt.Sprintf("%s-ui-proxy", jaeger.Name))
}
