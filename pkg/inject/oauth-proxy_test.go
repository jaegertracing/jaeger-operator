package inject

import (
	"fmt"
	"sort"
	"testing"

	"github.com/spf13/viper"
	"github.com/stretchr/testify/assert"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	"k8s.io/apimachinery/pkg/types"

	v1 "github.com/jaegertracing/jaeger-operator/pkg/apis/jaegertracing/v1"
	"github.com/jaegertracing/jaeger-operator/pkg/deployment"
	"github.com/jaegertracing/jaeger-operator/pkg/service"
)

func TestOAuthProxyContainerIsNotAddedByDefault(t *testing.T) {
	jaeger := v1.NewJaeger(types.NamespacedName{Name: "TestOAuthProxyContainerIsNotAddedByDefault"})
	dep := OAuthProxy(jaeger, deployment.NewQuery(jaeger).Get())
	assert.Len(t, dep.Spec.Template.Spec.Containers, 1)
	assert.Equal(t, "jaeger-query", dep.Spec.Template.Spec.Containers[0].Name)
}

func TestOAuthProxyContainerIsAdded(t *testing.T) {
	jaeger := v1.NewJaeger(types.NamespacedName{Name: "TestOAuthProxyContainerIsAdded"})
	jaeger.Spec.Ingress.Security = v1.IngressSecurityOAuthProxy
	dep := OAuthProxy(jaeger, deployment.NewQuery(jaeger).Get())
	assert.Len(t, dep.Spec.Template.Spec.Containers, 2)
	assert.Equal(t, "oauth-proxy", dep.Spec.Template.Spec.Containers[1].Name)
}

func TestOAuthProxyTLSSecretVolumeIsAdded(t *testing.T) {
	jaeger := v1.NewJaeger(types.NamespacedName{Name: "TestOAuthProxyTLSSecretVolumeIsAdded"})
	jaeger.Spec.Ingress.Security = v1.IngressSecurityOAuthProxy
	dep := OAuthProxy(jaeger, deployment.NewQuery(jaeger).Get())
	assert.Len(t, dep.Spec.Template.Spec.Volumes, 1)
	assert.Equal(t, dep.Spec.Template.Spec.Volumes[0].Name, service.GetTLSSecretNameForQueryService(jaeger))
}

func TestOAuthProxyTLSSecretVolumeIsNotAddedByDefault(t *testing.T) {
	jaeger := v1.NewJaeger(types.NamespacedName{Name: "TestOAuthProxyTLSSecretVolumeIsNotAddedByDefault"})
	dep := OAuthProxy(jaeger, deployment.NewQuery(jaeger).Get())
	assert.Len(t, dep.Spec.Template.Spec.Volumes, 0)
}

func TestOAuthProxyConsistentServiceAccountName(t *testing.T) {
	// see https://github.com/openshift/oauth-proxy/issues/95
	jaeger := v1.NewJaeger(types.NamespacedName{Name: "TestOAuthProxyConsistentServiceAccountName"})
	jaeger.Spec.Ingress.Security = v1.IngressSecurityOAuthProxy
	dep := OAuthProxy(jaeger, deployment.NewQuery(jaeger).Get())

	found := false
	for _, a := range dep.Spec.Template.Spec.Containers[1].Args {
		if a == fmt.Sprintf("--openshift-service-account=%s", dep.Spec.Template.Spec.ServiceAccountName) {
			found = true
		}
	}
	assert.True(t, found)
}

func TestOAuthProxyWithCustomSAR(t *testing.T) {
	jaeger := v1.NewJaeger(types.NamespacedName{Name: "TestOAuthProxyWithCustomSAR"})
	jaeger.Spec.Ingress.Security = v1.IngressSecurityOAuthProxy
	jaeger.Spec.Ingress.OpenShift.SAR = `{"namespace": "default", "resource": "pods", "verb": "get"}`
	dep := OAuthProxy(jaeger, deployment.NewQuery(jaeger).Get())

	found := false
	for _, a := range dep.Spec.Template.Spec.Containers[1].Args {
		if a == fmt.Sprintf("--openshift-sar=%s", jaeger.Spec.Ingress.OpenShift.SAR) {
			found = true
		}
	}
	assert.True(t, found)
}

func TestOAuthProxyWithCustomDelegateURLs(t *testing.T) {
	viper.Set("auth-delegator-available", true)
	defer viper.Reset()

	jaeger := v1.NewJaeger(types.NamespacedName{Name: "TestOAuthProxyWithCustomDelegateURLs"})
	jaeger.Spec.Ingress.Security = v1.IngressSecurityOAuthProxy
	jaeger.Spec.Ingress.OpenShift.DelegateURLs = `{"/":{"namespace": "{{ .Release.Namespace }}", "resource": "pods", "verb": "get"}}`
	dep := OAuthProxy(jaeger, deployment.NewQuery(jaeger).Get())

	found := false
	for _, a := range dep.Spec.Template.Spec.Containers[1].Args {
		if a == fmt.Sprintf("--openshift-delegate-urls=%s", jaeger.Spec.Ingress.OpenShift.DelegateURLs) {
			found = true
		}
	}
	assert.True(t, found)
}

func TestOAuthProxyWithCustomDelegateURLsWithoutProperClusterRole(t *testing.T) {
	viper.Set("auth-delegator-available", false)
	defer func() {
		viper.Reset()
		setDefaults()
	}()

	jaeger := v1.NewJaeger(types.NamespacedName{Name: "TestOAuthProxyWithCustomDelegateURLs"})
	jaeger.Spec.Ingress.Security = v1.IngressSecurityOAuthProxy
	jaeger.Spec.Ingress.OpenShift.DelegateURLs = `{"/":{"namespace": "{{ .Release.Namespace }}", "resource": "pods", "verb": "get"}}`
	dep := OAuthProxy(jaeger, deployment.NewQuery(jaeger).Get())

	found := false
	for _, a := range dep.Spec.Template.Spec.Containers[1].Args {
		if a == fmt.Sprintf("--openshift-delegate-urls=%s", jaeger.Spec.Ingress.OpenShift.DelegateURLs) {
			found = true
		}
	}
	assert.False(t, found)
}

func TestOAuthProxyOrderOfArguments(t *testing.T) {
	jaeger := v1.NewJaeger(types.NamespacedName{Name: "TestOAuthProxyConsistentServiceAccountName"})
	jaeger.Spec.Ingress.Security = v1.IngressSecurityOAuthProxy
	dep := OAuthProxy(jaeger, deployment.NewQuery(jaeger).Get())

	sortedArgs := make([]string, len(dep.Spec.Template.Spec.Containers[1].Args))
	copy(sortedArgs, dep.Spec.Template.Spec.Containers[1].Args)
	sort.Strings(sortedArgs)

	assert.Len(t, dep.Spec.Template.Spec.Containers, 2)
	assert.Len(t, dep.Spec.Template.Spec.Containers[1].Args, 7)
	assert.Equal(t, sortedArgs, dep.Spec.Template.Spec.Containers[1].Args)
}

func TestOAuthProxyResourceLimits(t *testing.T) {
	jaeger := v1.NewJaeger(types.NamespacedName{Name: "TestOAuthProxyResourceLimits"})
	jaeger.Spec.Resources = corev1.ResourceRequirements{
		Limits: corev1.ResourceList{
			corev1.ResourceLimitsCPU:              *resource.NewQuantity(1024, resource.BinarySI),
			corev1.ResourceLimitsEphemeralStorage: *resource.NewQuantity(512, resource.DecimalSI),
		},
		Requests: corev1.ResourceList{
			corev1.ResourceRequestsCPU:              *resource.NewQuantity(1024, resource.BinarySI),
			corev1.ResourceRequestsEphemeralStorage: *resource.NewQuantity(512, resource.DecimalSI),
		},
	}
	jaeger.Spec.Ingress.Resources = corev1.ResourceRequirements{
		Limits: corev1.ResourceList{
			corev1.ResourceLimitsCPU:    *resource.NewQuantity(2048, resource.BinarySI),
			corev1.ResourceLimitsMemory: *resource.NewQuantity(123, resource.DecimalSI),
		},
		Requests: corev1.ResourceList{
			corev1.ResourceRequestsCPU:    *resource.NewQuantity(2048, resource.BinarySI),
			corev1.ResourceRequestsMemory: *resource.NewQuantity(123, resource.DecimalSI),
		},
	}
	jaeger.Spec.Ingress.Security = v1.IngressSecurityOAuthProxy
	dep := OAuthProxy(jaeger, deployment.NewQuery(jaeger).Get())
	assert.Equal(t, *resource.NewQuantity(2048, resource.BinarySI), dep.Spec.Template.Spec.Containers[1].Resources.Limits[corev1.ResourceLimitsCPU])
	assert.Equal(t, *resource.NewQuantity(2048, resource.BinarySI), dep.Spec.Template.Spec.Containers[1].Resources.Requests[corev1.ResourceRequestsCPU])
	assert.Equal(t, *resource.NewQuantity(123, resource.DecimalSI), dep.Spec.Template.Spec.Containers[1].Resources.Limits[corev1.ResourceLimitsMemory])
	assert.Equal(t, *resource.NewQuantity(123, resource.DecimalSI), dep.Spec.Template.Spec.Containers[1].Resources.Requests[corev1.ResourceRequestsMemory])
	assert.Equal(t, *resource.NewQuantity(512, resource.DecimalSI), dep.Spec.Template.Spec.Containers[1].Resources.Limits[corev1.ResourceLimitsEphemeralStorage])
	assert.Equal(t, *resource.NewQuantity(512, resource.DecimalSI), dep.Spec.Template.Spec.Containers[1].Resources.Requests[corev1.ResourceRequestsEphemeralStorage])
}
