package ingress

import (
	"context"
	"testing"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"

	"github.com/stretchr/testify/assert"
	"k8s.io/apimachinery/pkg/types"

	networkingv1 "k8s.io/api/networking/v1"

	v1 "github.com/jaegertracing/jaeger-operator/apis/v1"
)

func TestQueryIngress(t *testing.T) {
	name := "TestQueryIngress"
	jaeger := v1.NewJaeger(types.NamespacedName{Name: name})
	ingress := NewQueryIngress(jaeger)

	dep := ingress.Get()

	assert.Contains(t, dep.Spec.DefaultBackend.Service.Name, "testqueryingress-query")
}

func TestQueryIngressDisabled(t *testing.T) {
	enabled := false
	name := "TestQueryIngressDisabled"
	jaeger := v1.NewJaeger(types.NamespacedName{Name: name})
	jaeger.Spec.Ingress.Enabled = &enabled
	ingress := NewQueryIngress(jaeger)

	dep := ingress.Get()

	assert.Nil(t, dep)
}

func TestQueryIngressEnabled(t *testing.T) {
	enabled := true
	name := "TestQueryIngressEnabled"
	jaeger := v1.NewJaeger(types.NamespacedName{Name: name})
	jaeger.Spec.Ingress.Enabled = &enabled
	ingress := NewQueryIngress(jaeger)

	dep := ingress.Get()

	assert.NotNil(t, dep)
	assert.NotNil(t, dep.Spec.DefaultBackend)
}

func TestIngressWithPath(t *testing.T) {
	type test struct {
		name     string
		strategy v1.DeploymentStrategy
		basePath string
	}
	allInOne := test{name: "TestQueryIngressAllInOneBasePath", strategy: v1.DeploymentStrategyAllInOne, basePath: "/jaeger"}
	production := test{name: "TestQueryIngressProduction", strategy: v1.DeploymentStrategyProduction, basePath: "/jaeger-production"}
	streaming := test{name: "TestQueryIngressStreaming", strategy: v1.DeploymentStrategyStreaming, basePath: "/jaeger-streaming"}

	tests := []test{allInOne, production, streaming}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			enabled := true
			jaeger := v1.NewJaeger(types.NamespacedName{Name: test.name})
			jaeger.Spec.Ingress.Enabled = &enabled
			jaeger.Spec.Strategy = test.strategy
			if test.strategy == v1.DeploymentStrategyAllInOne {
				jaeger.Spec.AllInOne.Options = v1.NewOptions(map[string]interface{}{"query.base-path": test.basePath})
			} else {
				jaeger.Spec.Query.Options = v1.NewOptions(map[string]interface{}{"query.base-path": test.basePath})
			}

			ingress := NewQueryIngress(jaeger)
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

func TestQueryIngressAnnotations(t *testing.T) {
	jaeger := v1.NewJaeger(types.NamespacedName{Name: "TestQueryIngressAnnotations"})
	jaeger.Spec.Annotations = map[string]string{
		"name":  "operator",
		"hello": "jaeger",
	}
	jaeger.Spec.Ingress.Annotations = map[string]string{
		"hello":                "world", // Override top level annotation
		"prometheus.io/scrape": "false",
	}

	ingress := NewQueryIngress(jaeger)
	dep := ingress.Get()

	assert.Equal(t, "operator", dep.Annotations["name"])
	assert.Equal(t, "world", dep.Annotations["hello"])
	assert.Equal(t, "false", dep.Annotations["prometheus.io/scrape"])
}

func TestQueryIngressLabels(t *testing.T) {
	jaeger := v1.NewJaeger(types.NamespacedName{Name: "TestQueryIngressLabels"})
	jaeger.Spec.Labels = map[string]string{
		"name":  "operator",
		"hello": "jaeger",
	}
	jaeger.Spec.Ingress.Labels = map[string]string{
		"hello":   "world", // Override top level annotation
		"another": "false",
	}

	ingress := NewQueryIngress(jaeger)
	dep := ingress.Get()

	assert.Equal(t, "operator", dep.Labels["name"])
	assert.Equal(t, "world", dep.Labels["hello"])
	assert.Equal(t, "false", dep.Labels["another"])
}

func TestQueryIngressWithHosts(t *testing.T) {
	enabled := true
	jaeger := v1.NewJaeger(types.NamespacedName{Name: "TestQueryIngressWithHosts"})
	jaeger.Spec.Ingress.Enabled = &enabled
	jaeger.Spec.Ingress.Hosts = []string{"test-host-1"}

	ingress := NewQueryIngress(jaeger)

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

func TestQueryIngressWithPathType(t *testing.T) {
	enabled := true
	jaeger := v1.NewJaeger(types.NamespacedName{Name: "TestQueryIngressWithHosts"})
	jaeger.Spec.Ingress.Enabled = &enabled
	jaeger.Spec.Ingress.PathType = networkingv1.PathType("Prefix")
	jaeger.Spec.Ingress.Hosts = []string{"test-host-1"}

	ingress := NewQueryIngress(jaeger)

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

func TestQueryIngressWithMultipleHosts(t *testing.T) {
	enabled := true
	jaeger := v1.NewJaeger(types.NamespacedName{Name: "TestQueryIngressWithMultipleHosts"})
	jaeger.Spec.Ingress.Enabled = &enabled
	jaeger.Spec.Ingress.Hosts = []string{"test-host-1", "test-host-2"}

	ingress := NewQueryIngress(jaeger)

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

func TestQueryIngressWithoutHosts(t *testing.T) {
	enabled := true
	jaeger := v1.NewJaeger(types.NamespacedName{Name: "TestQueryIngressWithoutHosts"})
	jaeger.Spec.Ingress.Enabled = &enabled

	ingress := NewQueryIngress(jaeger)

	dep := ingress.Get()

	assert.NotNil(t, dep)
	assert.NotNil(t, dep.Spec.DefaultBackend)
	assert.Empty(t, dep.Spec.Rules)
}

func TestQueryIngressQueryBasePathWithHosts(t *testing.T) {
	enabled := true
	name := "TestQueryIngressQueryBasePathWithHosts"
	jaeger := v1.NewJaeger(types.NamespacedName{Name: name})
	jaeger.Spec.Ingress.Enabled = &enabled
	jaeger.Spec.Ingress.Hosts = []string{"test-host-1"}
	jaeger.Spec.Strategy = v1.DeploymentStrategyProduction
	jaeger.Spec.Query.Options = v1.NewOptions(map[string]interface{}{"query.base-path": "/jaeger"})
	ingress := NewQueryIngress(jaeger)

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
func TestQueryIngressDeprecatedSecretName(t *testing.T) {
	jaeger := v1.NewJaeger(types.NamespacedName{Name: "TestQueryIngressDeprecatedSecretName"})

	jaeger.Spec.Ingress.SecretName = "test-secret"

	ingress := NewQueryIngress(jaeger)
	dep := ingress.Get()

	assert.Equal(t, "test-secret", dep.Spec.TLS[0].SecretName)
}

// TODO: Remove this test when ingress.secretName is removed from the spec
func TestQueryIngressTLSOverridesDeprecatedSecretName(t *testing.T) {
	jaeger := v1.NewJaeger(types.NamespacedName{Name: "TestQueryIngressTLSOverridesDeprecatedSecretName"})

	jaeger.Spec.Ingress.SecretName = "test-secret-secret-name"

	jaeger.Spec.Ingress.TLS = []v1.JaegerIngressTLSSpec{
		{
			SecretName: "test-secret-tls",
		},
	}

	ingress := NewQueryIngress(jaeger)
	dep := ingress.Get()

	assert.Len(t, dep.Spec.TLS, 1)
	assert.Equal(t, "test-secret-tls", dep.Spec.TLS[0].SecretName)
}

func TestQueryIngressTLSSecret(t *testing.T) {
	jaeger := v1.NewJaeger(types.NamespacedName{Name: "TestQueryIngressTLSSecret"})

	jaeger.Spec.Ingress.TLS = []v1.JaegerIngressTLSSpec{
		{
			SecretName: "test-secret",
		},
	}

	ingress := NewQueryIngress(jaeger)
	dep := ingress.Get()

	assert.Equal(t, "test-secret", dep.Spec.TLS[0].SecretName)
}

func TestQueryIngressClass(t *testing.T) {
	jaeger := v1.NewJaeger(types.NamespacedName{Name: "TestQueryIngressClass"})
	jaegerNoIngressNoClass := v1.NewJaeger(types.NamespacedName{Name: "TestQueryIngressNoClass"})

	inressClassName := "nginx"
	jaeger.Spec.Ingress.IngressClassName = &inressClassName

	ingress := NewQueryIngress(jaeger)
	ingressNoClass := NewQueryIngress(jaegerNoIngressNoClass)

	dep := ingress.Get()

	assert.NotNil(t, dep.Spec.IngressClassName)
	assert.Equal(t, "nginx", *dep.Spec.IngressClassName)
	assert.Nil(t, ingressNoClass.Get().Spec.IngressClassName)
}

func TestForDefaultIngressClass(t *testing.T) {
	jaegerNoIngressNoClass := v1.NewJaeger(types.NamespacedName{Name: "TestQueryIngressNoClass"})

	// for storing all ingress controller names installed inside cluster
	AvailableIngressController := make(map[string]bool)

	config, err := rest.InClusterConfig()
	if err != nil {
		t.Fatalf("%v", err)
	}
	clientSet, err := kubernetes.NewForConfig(config)
	if err != nil {
		t.Fatalf("%v", err)
	}
	ingressList, err := clientSet.NetworkingV1().IngressClasses().List(context.Background(), metav1.ListOptions{})
	if err != nil {
		t.Fatalf("%v", err)
	}

	// check for all available ingress controller installed in cluster
	for _, ingress := range ingressList.Items {
		AvailableIngressController[ingress.Name] = false
		for k, v := range ingress.Annotations {
			if k == "ingressclass.kubernetes.io/is-default-class" {
				if v == "true" {
					AvailableIngressController[ingress.Name] = true
				}
			}
		}
	}

	getIngressClass, err := getInClusterAvailableIngressClass()
	if err != nil {
		if len(AvailableIngressController) > 0 {
			t.Fatalf("%v", err)
		} else {
			ingressNoClass := NewQueryIngress(jaegerNoIngressNoClass)
			assert.Nil(t, ingressNoClass.Get().Spec.IngressClassName)
		}
	} else {
		jaegerNoIngressNoClass.Spec.Ingress.IngressClassName = &getIngressClass
	}

	ingressNoClass := NewQueryIngress(jaegerNoIngressNoClass)

	dep := ingressNoClass.Get()

	defaultIngressClass := getDefaultIngressController(AvailableIngressController)
	if len(defaultIngressClass) > 0 {
		assert.Equal(t, defaultIngressClass, *dep.Spec.IngressClassName)
	} else {
		if _, ok := AvailableIngressController["nginx"]; ok {
			assert.Equal(t, "nginx", *dep.Spec.IngressClassName)
		} else {
			ingressNoClass := NewQueryIngress(jaegerNoIngressNoClass)
			assert.Nil(t, ingressNoClass.Get().Spec.IngressClassName)
		}
	}
}

func TestQueryIngressTLSHosts(t *testing.T) {
	jaeger := v1.NewJaeger(types.NamespacedName{Name: "TestQueryIngressTLSHosts"})

	jaeger.Spec.Ingress.TLS = []v1.JaegerIngressTLSSpec{
		{
			Hosts: []string{"test-host-1"},
		},
		{
			Hosts: []string{"test-host-2", "test-host-3"},
		},
	}

	ingress := NewQueryIngress(jaeger)
	dep := ingress.Get()

	assert.Len(t, dep.Spec.TLS, 2)
	assert.Len(t, dep.Spec.TLS[0].Hosts, 1)
	assert.Len(t, dep.Spec.TLS[1].Hosts, 2)
	assert.Equal(t, "test-host-1", dep.Spec.TLS[0].Hosts[0])
	assert.Equal(t, "test-host-2", dep.Spec.TLS[1].Hosts[0])
	assert.Equal(t, "test-host-3", dep.Spec.TLS[1].Hosts[1])
}

func getDefaultIngressController(m map[string]bool) string {
	for key, value := range m {
		if value {
			return key
		}
	}
	return "" // return an empty string if no key has a true value
}
