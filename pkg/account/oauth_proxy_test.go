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

	assert.Equal(t, fmt.Sprintf("%s-ui-proxy", jaeger.Name), OAuthProxy(jaeger).Name)
}

func TestOAuthOverrideServiceAccountForQuery(t *testing.T) {
	jaeger := v1.NewJaeger(types.NamespacedName{Name: "TestOAuthOverrideServiceAccountForQuery"})
	jaeger.Spec.Ingress.Security = v1.IngressSecurityOAuthProxy
	jaeger.Spec.Query.ServiceAccount = "my-own-sa"

	assert.Equal(t, "my-own-sa", OAuthProxy(jaeger).Name)
}

func TestOAuthOverrideServiceAccountForAllComponents(t *testing.T) {
	jaeger := v1.NewJaeger(types.NamespacedName{Name: "TestOAuthOverrideServiceAccountForAllComponents"})
	jaeger.Spec.Ingress.Security = v1.IngressSecurityOAuthProxy
	jaeger.Spec.ServiceAccount = "my-own-sa"

	assert.Equal(t, "my-own-sa", OAuthProxy(jaeger).Name)
}
