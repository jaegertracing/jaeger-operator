package account

import (
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
