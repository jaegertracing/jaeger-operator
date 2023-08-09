package ingress

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"k8s.io/apimachinery/pkg/types"

	networkingv1 "k8s.io/api/networking/v1"

	v1 "github.com/jaegertracing/jaeger-operator/apis/v1"
)

func TestCollectorIngress(t *testing.T) {
	name := "TestCollectorIngress"
	jaeger := v1.NewJaeger(types.NamespacedName{Name: name})
	ingress := NewCollectorIngress(jaeger)

	dep := ingress.Get()

	assert.Contains(t, dep.Spec.DefaultBackend.Service.Name, "testcollectoringress-collector")
}

func TestCollectorIngressDisabled(t *testing.T) {
	enabled := false
	name := "TestCollectorIngressDisabled"
	jaeger := v1.NewJaeger(types.NamespacedName{Name: name})
	jaeger.Spec.Collector.Ingress.Enabled = &enabled
	ingress := NewCollectorIngress(jaeger)

	dep := ingress.Get()

	assert.Nil(t, dep)
}

func TestCollectorIngressEnabled(t *testing.T) {
	enabled := true
	name := "TestCollectorIngressEnabled"
	jaeger := v1.NewJaeger(types.NamespacedName{Name: name})
	jaeger.Spec.Collector.Ingress.Enabled = &enabled
	ingress := NewCollectorIngress(jaeger)

	dep := ingress.Get()

	assert.NotNil(t, dep)
	assert.NotNil(t, dep.Spec.DefaultBackend)
}

func TestCollectorIngressWithPath(t *testing.T) {
	type test struct {
		name     string
		strategy v1.DeploymentStrategy
		basePath string
	}
	allInOne := test{name: "TestCollectorIngressAllInOneBasePath", strategy: v1.DeploymentStrategyAllInOne, basePath: "/jaeger"}
	production := test{name: "TestCollectorIngressProduction", strategy: v1.DeploymentStrategyProduction, basePath: "/jaeger-production"}
	streaming := test{name: "TestCollectorIngressStreaming", strategy: v1.DeploymentStrategyStreaming, basePath: "/jaeger-streaming"}

	tests := []test{allInOne, production, streaming}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			enabled := true
			jaeger := v1.NewJaeger(types.NamespacedName{Name: test.name})
			jaeger.Spec.Collector.Ingress.Enabled = &enabled
			jaeger.Spec.Strategy = test.strategy
			if test.strategy == v1.DeploymentStrategyAllInOne {
				jaeger.Spec.AllInOne.Options = v1.NewOptions(map[string]interface{}{"collector.base-path": test.basePath})
			} else {
				jaeger.Spec.Collector.Options = v1.NewOptions(map[string]interface{}{"collector.base-path": test.basePath})
			}

			ingress := NewCollectorIngress(jaeger)
			dep := ingress.Get()

			assert.NotNil(t, dep)
			assert.Nil(t, dep.Spec.DefaultBackend)
			assert.Len(t, dep.Spec.Rules, 1)

			assert.Len(t, dep.Spec.Rules[0].HTTP.Paths, 1)
			assert.Equal(t, test.basePath, dep.Spec.Rules[0].HTTP.Paths[0].Path)
			assert.Empty(t, dep.Spec.Rules[0].Host)
			assert.NotNil(t, dep.Spec.Rules[0].HTTP.Paths[0].Backend)
		})
	}
}

func TestCollectorIngressAnnotations(t *testing.T) {
	jaeger := v1.NewJaeger(types.NamespacedName{Name: "TestCollectorIngressAnnotations"})
	jaeger.Spec.Annotations = map[string]string{
		"name":  "operator",
		"hello": "jaeger",
	}
	jaeger.Spec.Collector.Ingress.Annotations = map[string]string{
		"hello":                "world", // Override top level annotation
		"prometheus.io/scrape": "false",
	}

	ingress := NewCollectorIngress(jaeger)
	dep := ingress.Get()

	assert.Equal(t, "operator", dep.Annotations["name"])
	assert.Equal(t, "world", dep.Annotations["hello"])
	assert.Equal(t, "false", dep.Annotations["prometheus.io/scrape"])
}

func TestCollectorIngressLabels(t *testing.T) {
	jaeger := v1.NewJaeger(types.NamespacedName{Name: "TestCollectorIngressLabels"})
	jaeger.Spec.Labels = map[string]string{
		"name":  "operator",
		"hello": "jaeger",
	}
	jaeger.Spec.Collector.Ingress.Labels = map[string]string{
		"hello":   "world", // Override top level annotation
		"another": "false",
	}

	ingress := NewCollectorIngress(jaeger)
	dep := ingress.Get()

	assert.Equal(t, "operator", dep.Labels["name"])
	assert.Equal(t, "world", dep.Labels["hello"])
	assert.Equal(t, "false", dep.Labels["another"])
}

func TestCollectorIngressWithHosts(t *testing.T) {
	enabled := true
	jaeger := v1.NewJaeger(types.NamespacedName{Name: "TestCollectorIngressWithHosts"})
	jaeger.Spec.Collector.Ingress.Enabled = &enabled
	jaeger.Spec.Collector.Ingress.Hosts = []string{"test-host-1"}

	ingress := NewCollectorIngress(jaeger)

	dep := ingress.Get()

	assert.NotNil(t, dep)
	assert.Nil(t, dep.Spec.DefaultBackend)
	assert.Len(t, dep.Spec.Rules, 1)

	assert.Len(t, dep.Spec.Rules[0].HTTP.Paths, 1)
	assert.Empty(t, dep.Spec.Rules[0].HTTP.Paths[0].Path)
	assert.Equal(t, networkingv1.PathType("ImplementationSpecific"), *dep.Spec.Rules[0].HTTP.Paths[0].PathType)
	assert.Equal(t, "test-host-1", dep.Spec.Rules[0].Host)
	assert.NotNil(t, dep.Spec.Rules[0].HTTP.Paths[0].Backend)
}

func TestCollectorIngressWithPathType(t *testing.T) {
	enabled := true
	jaeger := v1.NewJaeger(types.NamespacedName{Name: "TestCollectorIngressWithHosts"})
	jaeger.Spec.Collector.Ingress.Enabled = &enabled
	jaeger.Spec.Collector.Ingress.PathType = networkingv1.PathType("Prefix")
	jaeger.Spec.Collector.Ingress.Hosts = []string{"test-host-1"}

	ingress := NewCollectorIngress(jaeger)

	dep := ingress.Get()

	assert.NotNil(t, dep)
	assert.Nil(t, dep.Spec.DefaultBackend)
	assert.Len(t, dep.Spec.Rules, 1)

	assert.Len(t, dep.Spec.Rules[0].HTTP.Paths, 1)
	assert.Empty(t, dep.Spec.Rules[0].HTTP.Paths[0].Path)
	assert.Equal(t, networkingv1.PathType("Prefix"), *dep.Spec.Rules[0].HTTP.Paths[0].PathType)
	assert.Equal(t, "test-host-1", dep.Spec.Rules[0].Host)
	assert.NotNil(t, dep.Spec.Rules[0].HTTP.Paths[0].Backend)
}

func TestCollectorIngressWithMultipleHosts(t *testing.T) {
	enabled := true
	jaeger := v1.NewJaeger(types.NamespacedName{Name: "TestCollectorIngressWithMultipleHosts"})
	jaeger.Spec.Collector.Ingress.Enabled = &enabled
	jaeger.Spec.Collector.Ingress.Hosts = []string{"test-host-1", "test-host-2"}

	ingress := NewCollectorIngress(jaeger)

	dep := ingress.Get()

	assert.NotNil(t, dep)
	assert.Nil(t, dep.Spec.DefaultBackend)
	assert.Len(t, dep.Spec.Rules, 2)

	assert.Len(t, dep.Spec.Rules[0].HTTP.Paths, 1)
	assert.Empty(t, dep.Spec.Rules[0].HTTP.Paths[0].Path)
	assert.Equal(t, "test-host-1", dep.Spec.Rules[0].Host)
	assert.NotNil(t, dep.Spec.Rules[0].HTTP.Paths[0].Backend)

	assert.Len(t, dep.Spec.Rules[1].HTTP.Paths, 1)
	assert.Empty(t, dep.Spec.Rules[1].HTTP.Paths[0].Path)
	assert.Equal(t, "test-host-2", dep.Spec.Rules[1].Host)
	assert.NotNil(t, dep.Spec.Rules[1].HTTP.Paths[0].Backend)
}

func TestCollectorIngressWithoutHosts(t *testing.T) {
	enabled := true
	jaeger := v1.NewJaeger(types.NamespacedName{Name: "TestCollectorIngressWithoutHosts"})
	jaeger.Spec.Collector.Ingress.Enabled = &enabled

	ingress := NewCollectorIngress(jaeger)

	dep := ingress.Get()

	assert.NotNil(t, dep)
	assert.NotNil(t, dep.Spec.DefaultBackend)
	assert.Empty(t, dep.Spec.Rules)
}

func TestCollectorIngressQueryBasePathWithHosts(t *testing.T) {
	enabled := true
	name := "TestCollectorIngressQueryBasePathWithHosts"
	jaeger := v1.NewJaeger(types.NamespacedName{Name: name})
	jaeger.Spec.Collector.Ingress.Enabled = &enabled
	jaeger.Spec.Collector.Ingress.Hosts = []string{"test-host-1"}
	jaeger.Spec.Strategy = v1.DeploymentStrategyProduction
	jaeger.Spec.Collector.Options = v1.NewOptions(map[string]interface{}{"collector.base-path": "/jaeger"})
	ingress := NewCollectorIngress(jaeger)

	dep := ingress.Get()

	assert.NotNil(t, dep)
	assert.Nil(t, dep.Spec.DefaultBackend)
	assert.Len(t, dep.Spec.Rules, 1)

	assert.Len(t, dep.Spec.Rules[0].HTTP.Paths, 1)
	assert.Equal(t, "/jaeger", dep.Spec.Rules[0].HTTP.Paths[0].Path)
	assert.Equal(t, "test-host-1", dep.Spec.Rules[0].Host)
	assert.NotNil(t, dep.Spec.Rules[0].HTTP.Paths[0].Backend)
}

// TODO: Remove this test when ingress.secretName is removed from the spec
func TestCollectorIngressDeprecatedSecretName(t *testing.T) {
	jaeger := v1.NewJaeger(types.NamespacedName{Name: "TestCollectorIngressDeprecatedSecretName"})

	jaeger.Spec.Collector.Ingress.SecretName = "test-secret"

	ingress := NewCollectorIngress(jaeger)
	dep := ingress.Get()

	assert.Equal(t, "test-secret", dep.Spec.TLS[0].SecretName)
}

// TODO: Remove this test when ingress.secretName is removed from the spec
func TestCollectorIngressTLSOverridesDeprecatedSecretName(t *testing.T) {
	jaeger := v1.NewJaeger(types.NamespacedName{Name: "TestCollectorIngressTLSOverridesDeprecatedSecretName"})

	jaeger.Spec.Collector.Ingress.SecretName = "test-secret-secret-name"

	jaeger.Spec.Collector.Ingress.TLS = []v1.JaegerIngressTLSSpec{
		{
			SecretName: "test-secret-tls",
		},
	}

	ingress := NewCollectorIngress(jaeger)
	dep := ingress.Get()

	assert.Len(t, dep.Spec.TLS, 1)
	assert.Equal(t, "test-secret-tls", dep.Spec.TLS[0].SecretName)
}

func TestCollectorIngressTLSSecret(t *testing.T) {
	jaeger := v1.NewJaeger(types.NamespacedName{Name: "TestCollectorIngressTLSSecret"})

	jaeger.Spec.Collector.Ingress.TLS = []v1.JaegerIngressTLSSpec{
		{
			SecretName: "test-secret",
		},
	}

	ingress := NewCollectorIngress(jaeger)
	dep := ingress.Get()

	assert.Equal(t, "test-secret", dep.Spec.TLS[0].SecretName)
}

func TestCollectorIngressClass(t *testing.T) {
	jaeger := v1.NewJaeger(types.NamespacedName{Name: "TestCollectorIngressClass"})
	jaegerNoIngressNoClass := v1.NewJaeger(types.NamespacedName{Name: "TestCollectorIngressNoClass"})

	inressClassName := "nginx"
	jaeger.Spec.Collector.Ingress.IngressClassName = &inressClassName

	ingress := NewCollectorIngress(jaeger)
	ingressNoClass := NewCollectorIngress(jaegerNoIngressNoClass)

	dep := ingress.Get()

	assert.NotNil(t, dep.Spec.IngressClassName)
	assert.Equal(t, "nginx", *dep.Spec.IngressClassName)
	assert.Nil(t, ingressNoClass.Get().Spec.IngressClassName)
}

func TestCollectorIngressTLSHosts(t *testing.T) {
	jaeger := v1.NewJaeger(types.NamespacedName{Name: "TestCollectorIngressTLSHosts"})

	jaeger.Spec.Collector.Ingress.TLS = []v1.JaegerIngressTLSSpec{
		{
			Hosts: []string{"test-host-1"},
		},
		{
			Hosts: []string{"test-host-2", "test-host-3"},
		},
	}

	ingress := NewCollectorIngress(jaeger)
	dep := ingress.Get()

	assert.Len(t, dep.Spec.TLS, 2)
	assert.Len(t, dep.Spec.TLS[0].Hosts, 1)
	assert.Len(t, dep.Spec.TLS[1].Hosts, 2)
	assert.Equal(t, "test-host-1", dep.Spec.TLS[0].Hosts[0])
	assert.Equal(t, "test-host-2", dep.Spec.TLS[1].Hosts[0])
	assert.Equal(t, "test-host-3", dep.Spec.TLS[1].Hosts[1])
}
