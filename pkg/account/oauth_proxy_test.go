package account

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"k8s.io/apimachinery/pkg/types"

	v1 "github.com/jaegertracing/jaeger-operator/pkg/apis/jaegertracing/v1"
)

func TestOAuthRedirectReference(t *testing.T) {
	jaeger := v1.NewJaeger(types.NamespacedName{Name: "TestOAuthRedirectReference"})
	jaeger.Spec.Ingress.Security = v1.IngressSecurityOAuthProxy

	assert.Contains(t, getOAuthRedirectReference(jaeger), jaeger.Name)
}

func TestOAuthProxy(t *testing.T) {
	jaeger := v1.NewJaeger(types.NamespacedName{Name: "TestOAuthProxy"})
	jaeger.Spec.Ingress.Security = v1.IngressSecurityOAuthProxy

	assert.Equal(t, OAuthProxy(jaeger).Name, fmt.Sprintf("%s-ui-proxy", jaeger.Name))
}
