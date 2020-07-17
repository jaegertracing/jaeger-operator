package inject

import (
	"fmt"
	"sort"
	"testing"
	"strings"

	v1 "github.com/jaegertracing/jaeger-operator/pkg/apis/jaegertracing/v1"
	"github.com/jaegertracing/jaeger-operator/pkg/config/ca"
	"github.com/jaegertracing/jaeger-operator/pkg/util"

	"github.com/spf13/viper"
	"github.com/stretchr/testify/assert"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	"k8s.io/apimachinery/pkg/types"

	"github.com/jaegertracing/jaeger-operator/pkg/deployment"
	"github.com/jaegertracing/jaeger-operator/pkg/service"
)

func TestOAuthProxyContainerIsNotAddedByDefault(t *testing.T) {
	jaeger := v1.NewJaeger(types.NamespacedName{Name: "my-instance"})
	dep := OAuthProxy(jaeger, deployment.NewQuery(jaeger).Get())
	assert.Len(t, dep.Spec.Template.Spec.Containers, 1)
	assert.Equal(t, "jaeger-query", dep.Spec.Template.Spec.Containers[0].Name)
}

func TestNoneOpenShiftOAuthProxyContainerIsAdded(t *testing.T) {
	jaeger := v1.NewJaeger(types.NamespacedName{Name: "my-instance"})
	jaeger.Spec.Ingress.Security = v1.IngressSecurityOAuthProxy
    defer viper.Reset()
	dep := OAuthProxy(jaeger, deployment.NewQuery(jaeger).Get())
	assert.Len(t, dep.Spec.Template.Spec.Containers, 2)
	assert.Equal(t, "oauth-proxy", dep.Spec.Template.Spec.Containers[1].Name)
}

func TestOpenShiftOAuthProxyContainerIsAdded(t *testing.T) {
	viper.Set("platform", "openshift")
    defer viper.Reset()
	jaeger := v1.NewJaeger(types.NamespacedName{Name: "my-instance"})
	jaeger.Spec.Ingress.Security = v1.IngressSecurityOAuthProxy

	dep := OAuthProxy(jaeger, deployment.NewQuery(jaeger).Get())
	assert.Len(t, dep.Spec.Template.Spec.Containers, 2)
	assert.Equal(t, "oauth-proxy", dep.Spec.Template.Spec.Containers[1].Name)
}

func TestOpenShiftOAuthProxyTLSSecretVolumeIsAdded(t *testing.T) {
    viper.Set("platform", "openshift")
    defer viper.Reset()
	jaeger := v1.NewJaeger(types.NamespacedName{Name: "my-instance"})
	jaeger.Spec.Ingress.Security = v1.IngressSecurityOAuthProxy
	dep := OAuthProxy(jaeger, deployment.NewQuery(jaeger).Get())
	assert.Len(t, dep.Spec.Template.Spec.Volumes, 2)
	assert.Equal(t, dep.Spec.Template.Spec.Volumes[1].Name, service.GetTLSSecretNameForQueryService(jaeger))
}

func TestOAuthProxyTLSSecretVolumeIsNotAddedByDefault(t *testing.T) {
	jaeger := v1.NewJaeger(types.NamespacedName{Name: "my-instance"})
	dep := OAuthProxy(jaeger, deployment.NewQuery(jaeger).Get())
	assert.Len(t, dep.Spec.Template.Spec.Volumes, 0)
}

func TestOpenShiftOAuthProxyConsistentServiceAccountName(t *testing.T) {
	// see https://github.com/openshift/oauth-proxy/issues/95
	viper.Set("platform", "openshift")
    defer viper.Reset()
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

func TestOpenShiftOAuthProxyWithCustomSAR(t *testing.T) {
    viper.Set("platform", "openshift")
    defer viper.Reset()
	jaeger := v1.NewJaeger(types.NamespacedName{Name: "my-instance"})
	jaeger.Spec.Ingress.Security = v1.IngressSecurityOAuthProxy
	jaeger.Spec.Ingress.Openshift.SAR = `{"namespace": "default", "resource": "pods", "verb": "get"}`
	dep := OAuthProxy(jaeger, deployment.NewQuery(jaeger).Get())

	found := false
	for _, a := range dep.Spec.Template.Spec.Containers[1].Args {
		if a == fmt.Sprintf("--openshift-sar=%s", jaeger.Spec.Ingress.Openshift.SAR) {
			found = true
		}
	}
	assert.True(t, found)
}

func TestOpenShiftOAuthProxyWithHtpasswdFile(t *testing.T) {
    viper.Set("platform", "openshift")
    defer viper.Reset()
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

func TestNoneOpenShiftVolumesNotMountedByDefault(t *testing.T) {
    jaeger := v1.NewJaeger(types.NamespacedName{Name: "my-instance"})
    jaeger.Spec.Ingress.Security = v1.IngressSecurityOAuthProxy
    jaeger.Spec.Query.OauthProxy = new(v1.JaegerQueryOauthProxySpec)

    dep := OAuthProxy(jaeger, deployment.NewQuery(jaeger).Get())

    assert.Len(t, dep.Spec.Template.Spec.Containers[1].VolumeMounts, 0)
}

func TestNoneOpenShiftMountVolume(t *testing.T) {
    jaeger := v1.NewJaeger(types.NamespacedName{Name: "my-instance"})
    jaeger.Spec.Ingress.Security = v1.IngressSecurityOAuthProxy

    jaeger.Spec.Query.OauthProxy = new(v1.JaegerQueryOauthProxySpec)
    jaeger.Spec.Query.OauthProxy.VolumeMounts = []corev1.VolumeMount{{
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
    assert.Len(t, dep.Spec.Template.Spec.Containers[1].VolumeMounts, 1)
}


func TestOpenShiftMountVolumeSpecifiedAtMainSpec(t *testing.T) {
    viper.Set("platform", "openshift")
    defer viper.Reset()
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

func TestOpenShiftDoNotMountWhenNotNeeded(t *testing.T) {
    viper.Set("platform", "openshift")
    defer viper.Reset()
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

func TestOpenShiftOAuthProxyWithCustomDelegateURLs(t *testing.T) {
	viper.Set("auth-delegator-available", true)
    viper.Set("platform", "openshift")
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

func TestOpenShiftOAuthProxyWithCustomDelegateURLsWithoutProperClusterRole(t *testing.T) {
	viper.Set("auth-delegator-available", false)
    viper.Set("platform", "openshift")
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
	jaeger.Spec.Query.OauthProxy = new(v1.JaegerQueryOauthProxySpec)
    o := v1.NewOptions(map[string]interface{}{
        "upstream-url": "http://127.0.0.1:9090",
        "client-id": "jaeger-client",
        "client-secret": "12345678-9abc-def0-1234-56789abcdef0",
        "redirection-url": "https://jaeger-gatekeeper.example.com",
    })
    jaeger.Spec.Query.OauthProxy.Options = o

	dep := OAuthProxy(jaeger, deployment.NewQuery(jaeger).Get())

	sortedArgs := make([]string, len(dep.Spec.Template.Spec.Containers[1].Args))
	copy(sortedArgs, dep.Spec.Template.Spec.Containers[1].Args)
	sort.Strings(sortedArgs)

	assert.Len(t, dep.Spec.Template.Spec.Containers, 2)
	assert.Len(t, dep.Spec.Template.Spec.Containers[1].Args, 4)
	assert.Equal(t, sortedArgs, dep.Spec.Template.Spec.Containers[1].Args)
}

func TestOpenShiftOAuthProxyOrderOfArguments(t *testing.T) {
	viper.Set("platform", "openshift")
	defer viper.Reset()
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

func TestNoneOpenShiftOAuthProxyDefaultImageIsKeycloakGatekeeper(t *testing.T) {
    jaeger := v1.NewJaeger(types.NamespacedName{Name: "my-instance"})
    jaeger.Spec.Ingress.Security = v1.IngressSecurityOAuthProxy
    jaeger.Spec.Query.OauthProxy = new(v1.JaegerQueryOauthProxySpec)
    viper.SetDefault("oauth-proxy-image", "quay.io/keycloak/keycloak-gatekeeper:10.0.0")
    dep := OAuthProxy(jaeger, deployment.NewQuery(jaeger).Get())

    assert.Len(t, dep.Spec.Template.Spec.Containers, 2)
    assert.Equal(t, "", jaeger.Spec.Query.OauthProxy.Image)
    assert.Equal(t, "quay.io/keycloak/keycloak-gatekeeper", strings.Split(dep.Spec.Template.Spec.Containers[1].Image, ":")[0])
}

func TestNoneOpenShiftOAuthProxySetImageOtherThanDefault(t *testing.T) {
    jaeger := v1.NewJaeger(types.NamespacedName{Name: "my-instance"})
    jaeger.Spec.Ingress.Security = v1.IngressSecurityOAuthProxy
    jaeger.Spec.Query.OauthProxy = new(v1.JaegerQueryOauthProxySpec)
    jaeger.Spec.Query.OauthProxy.Image = "quay.io/anotherImage:1.0.0"

    dep := OAuthProxy(jaeger, deployment.NewQuery(jaeger).Get())

    assert.Len(t, dep.Spec.Template.Spec.Containers, 2)
    assert.Equal(t, jaeger.Spec.Query.OauthProxy.Image, dep.Spec.Template.Spec.Containers[1].Image)
}

func TestOAuthProxyResourceLimits(t *testing.T) {
    var platformTests = []struct {
        platform string
    }{
        {"default"},
        {"openshift"},
    }

    for _, pftest := range platformTests {
        if pftest.platform != "default" {
            viper.Set("platform", pftest.platform)
            defer viper.Reset()
        }

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
}

func findCookieSecret(containers []corev1.Container) (string, bool) {
	for _, container := range containers {
		if container.Name == "oauth-proxy" {
			return util.FindItem("--cookie-secret=", container.Args), true
		}
	}
	return "", false
}

func TestOpenShiftPropagateOAuthCookieSecret(t *testing.T) {
    viper.Set("platform", "openshift")
    defer viper.Reset()
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
	viper.Set("platform", v1.FlagPlatformOpenShift)
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
