package ingress

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/jaegertracing/jaeger-operator/pkg/apis/io/v1alpha1"
)

func TestQueryIngress(t *testing.T) {
	name := "TestQueryIngress"
	jaeger := v1alpha1.NewJaeger(name)
	ingress := NewQueryIngress(jaeger)

	dep := ingress.Get()

	assert.Contains(t, dep.Spec.Backend.ServiceName, fmt.Sprintf("%s-query", name))
}

func TestQueryIngressDisabled(t *testing.T) {
	enabled := false
	name := "TestQueryIngressDisabled"
	jaeger := v1alpha1.NewJaeger(name)
	jaeger.Spec.Ingress.Enabled = &enabled
	ingress := NewQueryIngress(jaeger)

	dep := ingress.Get()

	assert.Nil(t, dep)
}

func TestQueryIngressEnabled(t *testing.T) {
	enabled := true
	name := "TestQueryIngressEnabled"
	jaeger := v1alpha1.NewJaeger(name)
	jaeger.Spec.Ingress.Enabled = &enabled
	ingress := NewQueryIngress(jaeger)

	dep := ingress.Get()

	assert.NotNil(t, dep)
	assert.NotNil(t, dep.Spec.Backend)
}

func TestQueryIngressAllInOneBasePath(t *testing.T) {
	enabled := true
	name := "TestQueryIngressAllInOneBasePath"
	jaeger := v1alpha1.NewJaeger(name)
	jaeger.Spec.Ingress.Enabled = &enabled
	jaeger.Spec.Strategy = "allInOne"
	jaeger.Spec.AllInOne.Options = v1alpha1.NewOptions(map[string]interface{}{"query.base-path": "/jaeger"})
	ingress := NewQueryIngress(jaeger)

	dep := ingress.Get()

	assert.NotNil(t, dep)
	assert.Nil(t, dep.Spec.Backend)
	assert.Len(t, dep.Spec.Rules, 1)
	assert.Len(t, dep.Spec.Rules[0].HTTP.Paths, 1)
	assert.Equal(t, "/jaeger", dep.Spec.Rules[0].HTTP.Paths[0].Path)
	assert.NotNil(t, dep.Spec.Rules[0].HTTP.Paths[0].Backend)
}

func TestQueryIngressQueryBasePath(t *testing.T) {
	enabled := true
	name := "TestQueryIngressQueryBasePath"
	jaeger := v1alpha1.NewJaeger(name)
	jaeger.Spec.Ingress.Enabled = &enabled
	jaeger.Spec.Strategy = "production"
	jaeger.Spec.Query.Options = v1alpha1.NewOptions(map[string]interface{}{"query.base-path": "/jaeger"})
	ingress := NewQueryIngress(jaeger)

	dep := ingress.Get()

	assert.NotNil(t, dep)
	assert.Nil(t, dep.Spec.Backend)
	assert.Len(t, dep.Spec.Rules, 1)
	assert.Len(t, dep.Spec.Rules[0].HTTP.Paths, 1)
	assert.Equal(t, "/jaeger", dep.Spec.Rules[0].HTTP.Paths[0].Path)
	assert.NotNil(t, dep.Spec.Rules[0].HTTP.Paths[0].Backend)
}

func TestQueryIngressAnnotations(t *testing.T) {
	jaeger := v1alpha1.NewJaeger("TestQueryIngressAnnotations")
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
