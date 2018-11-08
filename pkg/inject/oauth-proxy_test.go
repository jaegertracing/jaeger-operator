package inject

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/jaegertracing/jaeger-operator/pkg/apis/io/v1alpha1"
	"github.com/jaegertracing/jaeger-operator/pkg/deployment"
	"github.com/jaegertracing/jaeger-operator/pkg/service"
)

func TestOAuthProxyContainerIsNotAddedByDefault(t *testing.T) {
	jaeger := v1alpha1.NewJaeger("TestOAuthProxyContainerIsNotAddedByDefault")
	dep := OAuthProxy(jaeger, deployment.NewQuery(jaeger).Get())
	assert.Len(t, dep.Spec.Template.Spec.Containers, 1)
	assert.Equal(t, "jaeger-query", dep.Spec.Template.Spec.Containers[0].Name)
}

func TestOAuthProxyContainerIsAdded(t *testing.T) {
	jaeger := v1alpha1.NewJaeger("TestOAuthProxyContainerIsAdded")
	b := true
	jaeger.Spec.Ingress.OAuthProxy = &b
	dep := OAuthProxy(jaeger, deployment.NewQuery(jaeger).Get())
	assert.Len(t, dep.Spec.Template.Spec.Containers, 2)
	assert.Equal(t, "oauth-proxy", dep.Spec.Template.Spec.Containers[1].Name)
}

func TestOAuthProxyTLSSecretVolumeIsAdded(t *testing.T) {
	jaeger := v1alpha1.NewJaeger("TestOAuthProxyTLSSecretVolumeIsAdded")
	b := true
	jaeger.Spec.Ingress.OAuthProxy = &b
	dep := OAuthProxy(jaeger, deployment.NewQuery(jaeger).Get())
	assert.Len(t, dep.Spec.Template.Spec.Volumes, 1)
	assert.Equal(t, dep.Spec.Template.Spec.Volumes[0].Name, service.GetTLSSecretNameForQueryService(jaeger))
}

func TestOAuthProxyTLSSecretVolumeIsNotAddedByDefault(t *testing.T) {
	jaeger := v1alpha1.NewJaeger("TestOAuthProxyTLSSecretVolumeIsAdded")
	dep := OAuthProxy(jaeger, deployment.NewQuery(jaeger).Get())
	assert.Len(t, dep.Spec.Template.Spec.Volumes, 0)
}
