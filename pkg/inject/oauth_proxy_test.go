package inject

import (
	"fmt"
	"sort"
	"testing"
	"time"

	"github.com/spf13/viper"
	"github.com/stretchr/testify/assert"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"

	v1 "github.com/jaegertracing/jaeger-operator/apis/v1"
	"github.com/jaegertracing/jaeger-operator/pkg/autodetect"
	"github.com/jaegertracing/jaeger-operator/pkg/config/ca"
	"github.com/jaegertracing/jaeger-operator/pkg/deployment"
	"github.com/jaegertracing/jaeger-operator/pkg/service"
	"github.com/jaegertracing/jaeger-operator/pkg/util"
)

func TestOAuthProxyContainerIsNotAddedByDefault(t *testing.T) {
	jaeger := v1.NewJaeger(types.NamespacedName{Name: "my-instance"})
	dep := OAuthProxy(jaeger, deployment.NewQuery(jaeger).Get())
	assert.Len(t, dep.Spec.Template.Spec.Containers, 1)
	assert.Equal(t, "jaeger-query", dep.Spec.Template.Spec.Containers[0].Name)
}

func TestOAuthProxyContainerIsAdded(t *testing.T) {
	jaeger := v1.NewJaeger(types.NamespacedName{Name: "my-instance"})
	jaeger.Spec.Ingress.Security = v1.IngressSecurityOAuthProxy
	dep := OAuthProxy(jaeger, deployment.NewQuery(jaeger).Get())
	assert.Len(t, dep.Spec.Template.Spec.Containers, 2)
	assert.Equal(t, "oauth-proxy", dep.Spec.Template.Spec.Containers[1].Name)
}

func TestOAuthProxyTLSSecretVolumeIsAdded(t *testing.T) {
	jaeger := v1.NewJaeger(types.NamespacedName{Name: "my-instance"})
	jaeger.Spec.Ingress.Security = v1.IngressSecurityOAuthProxy
	dep := OAuthProxy(jaeger, deployment.NewQuery(jaeger).Get())
	assert.Len(t, dep.Spec.Template.Spec.Volumes, 1)
	assert.Equal(t, dep.Spec.Template.Spec.Volumes[0].Name, service.GetTLSSecretNameForQueryService(jaeger))
}

func TestOAuthProxyTLSSecretVolumeIsNotAddedByDefault(t *testing.T) {
	jaeger := v1.NewJaeger(types.NamespacedName{Name: "my-instance"})
	dep := OAuthProxy(jaeger, deployment.NewQuery(jaeger).Get())
	assert.Empty(t, dep.Spec.Template.Spec.Volumes)
}

func TestOAuthProxyConsistentServiceAccountName(t *testing.T) {
	// see https://github.com/openshift/oauth-proxy/issues/95
	jaeger := v1.NewJaeger(types.NamespacedName{Name: "my-instance"})
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
	jaeger := v1.NewJaeger(types.NamespacedName{Name: "my-instance"})
	jaeger.Spec.Ingress.Security = v1.IngressSecurityOAuthProxy
	sar := `{"namespace": "default", "resource": "pods", "verb": "get"}`
	jaeger.Spec.Ingress.Openshift.SAR = &sar
	dep := OAuthProxy(jaeger, deployment.NewQuery(jaeger).Get())

	found := false
	for _, a := range dep.Spec.Template.Spec.Containers[1].Args {
		if a == fmt.Sprintf("--openshift-sar=%s", *jaeger.Spec.Ingress.Openshift.SAR) {
			found = true
		}
	}
	assert.True(t, found)
}

func TestOAuthProxyWithTimeout(t *testing.T) {
	jaeger := v1.NewJaeger(types.NamespacedName{Name: "my-instance"})
	jaeger.Spec.Ingress.Security = v1.IngressSecurityOAuthProxy

	timeout := metav1.Duration{
		Duration: time.Second * 70,
	}
	jaeger.Spec.Ingress.Openshift.Timeout = &timeout
	dep := OAuthProxy(jaeger, deployment.NewQuery(jaeger).Get())

	found := false
	for _, a := range dep.Spec.Template.Spec.Containers[1].Args {
		if a == "--upstream-timeout=1m10s" {
			found = true
		}
	}
	assert.True(t, found)
}

func TestOAuthProxyWithHtpasswdFile(t *testing.T) {
	jaeger := v1.NewJaeger(types.NamespacedName{Name: "my-instance"})
	jaeger.Spec.Ingress.Security = v1.IngressSecurityOAuthProxy
	jaeger.Spec.Ingress.Openshift.HtpasswdFile = "/etc/htpasswd"
	dep := OAuthProxy(jaeger, deployment.NewQuery(jaeger).Get())

	found := false
	for _, a := range dep.Spec.Template.Spec.Containers[1].Args {
		if a == fmt.Sprintf("--htpasswd-file=%s", jaeger.Spec.Ingress.Openshift.HtpasswdFile) {
			found = true
		}
	}
	assert.True(t, found)
}

func TestMountVolumeSpecifiedAtMainSpec(t *testing.T) {
	jaeger := v1.NewJaeger(types.NamespacedName{Name: "my-instance"})
	jaeger.Spec.Ingress.Security = v1.IngressSecurityOAuthProxy
	jaeger.Spec.Ingress.Openshift.HtpasswdFile = "/etc/passwd"
	jaeger.Spec.VolumeMounts = []corev1.VolumeMount{{
		Name: "the-volume",
	}}
	dep := OAuthProxy(jaeger, deployment.NewQuery(jaeger).Get())

	found := false
	for _, a := range dep.Spec.Template.Spec.Containers[1].VolumeMounts {
		if a.Name == "the-volume" {
			found = true
		}
	}
	assert.True(t, found)
}

func TestDoNotMountWhenNotNeeded(t *testing.T) {
	jaeger := v1.NewJaeger(types.NamespacedName{Name: "my-instance"})
	jaeger.Spec.Ingress.Security = v1.IngressSecurityOAuthProxy
	jaeger.Spec.VolumeMounts = []corev1.VolumeMount{{
		Name: "the-volume",
	}}
	dep := OAuthProxy(jaeger, deployment.NewQuery(jaeger).Get())

	found := false
	for _, a := range dep.Spec.Template.Spec.Containers[1].VolumeMounts {
		if a.Name == "the-volume" {
			found = true
		}
	}
	assert.False(t, found)
}

func TestOAuthProxyWithCustomDelegateURLs(t *testing.T) {
	autodetect.OperatorConfiguration.SetAuthDelegatorAvailability(autodetect.AuthDelegatorAvailabilityYes)
	defer viper.Reset()

	jaeger := v1.NewJaeger(types.NamespacedName{Name: "my-instance"})
	jaeger.Spec.Ingress.Security = v1.IngressSecurityOAuthProxy
	jaeger.Spec.Ingress.Openshift.DelegateUrls = `{"/":{"namespace": "{{ .Release.Namespace }}", "resource": "pods", "verb": "get"}}`
	dep := OAuthProxy(jaeger, deployment.NewQuery(jaeger).Get())

	found := false
	for _, a := range dep.Spec.Template.Spec.Containers[1].Args {
		if a == fmt.Sprintf("--openshift-delegate-urls=%s", jaeger.Spec.Ingress.Openshift.DelegateUrls) {
			found = true
		}
	}
	assert.True(t, found)
}

func TestOAuthProxyWithCustomDelegateURLsWithoutProperClusterRole(t *testing.T) {
	autodetect.OperatorConfiguration.SetAuthDelegatorAvailability(autodetect.AuthDelegatorAvailabilityNo)
	defer func() {
		viper.Reset()
		setDefaults()
	}()

	jaeger := v1.NewJaeger(types.NamespacedName{Name: "my-instance"})
	jaeger.Spec.Ingress.Security = v1.IngressSecurityOAuthProxy
	jaeger.Spec.Ingress.Openshift.DelegateUrls = `{"/":{"namespace": "{{ .Release.Namespace }}", "resource": "pods", "verb": "get"}}`
	dep := OAuthProxy(jaeger, deployment.NewQuery(jaeger).Get())

	found := false
	for _, a := range dep.Spec.Template.Spec.Containers[1].Args {
		if a == fmt.Sprintf("--openshift-delegate-urls=%s", jaeger.Spec.Ingress.Openshift.DelegateUrls) {
			found = true
		}
	}
	assert.False(t, found)
}

func TestOAuthProxyOrderOfArguments(t *testing.T) {
	jaeger := v1.NewJaeger(types.NamespacedName{Name: "my-instance"})
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
	jaeger := v1.NewJaeger(types.NamespacedName{Name: "my-instance"})
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

func findCookieSecret(containers []corev1.Container) (string, bool) {
	for _, container := range containers {
		if container.Name == "oauth-proxy" {
			return util.FindItem("--cookie-secret=", container.Args), true
		}
	}
	return "", false
}

func TestPropagateOAuthCookieSecret(t *testing.T) {
	jaeger := v1.NewJaeger(types.NamespacedName{Name: "my-instance"})
	jaeger.Spec.Ingress.Security = v1.IngressSecurityOAuthProxy
	depSrc := OAuthProxy(jaeger, deployment.NewQuery(jaeger).Get())
	depDst := OAuthProxy(jaeger, deployment.NewQuery(jaeger).Get())
	srcSecret, _ := findCookieSecret(depSrc.Spec.Template.Spec.Containers)
	dstSecret, _ := findCookieSecret(depDst.Spec.Template.Spec.Containers)
	assert.NotEqual(t, srcSecret, dstSecret)
	resultSpec := PropagateOAuthCookieSecret(depSrc.Spec, depDst.Spec)
	resultSecret, _ := findCookieSecret(resultSpec.Template.Spec.Containers)
	assert.Equal(t, srcSecret, resultSecret)
}

func TestTrustedCAVolumeIsUsed(t *testing.T) {
	autodetect.OperatorConfiguration.SetPlatform(autodetect.OpenShiftPlatform)
	defer func() {
		viper.Reset()
		setDefaults()
	}()

	// prepare
	jaeger := v1.NewJaeger(types.NamespacedName{Name: "my-instance"})
	jaeger.Spec.Ingress.Security = v1.IngressSecurityOAuthProxy

	// test
	dep := OAuthProxy(jaeger, deployment.NewQuery(jaeger).Get())

	// verify
	assert.Len(t, dep.Spec.Template.Spec.Containers, 2)

	// get the oauth proxy container, and verify that the volumemount was added
	for _, c := range dep.Spec.Template.Spec.Containers {
		if c.Name == "oauth-proxy" {
			for _, v := range c.VolumeMounts {
				if v.Name == ca.TrustedCAName(jaeger) {
					return
				}
			}
		}
	}

	assert.Fail(t, "couldn't find the OAuth Proxy container")
}
