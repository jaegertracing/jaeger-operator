package deployment

import (
	"fmt"
	"testing"

	"github.com/spf13/viper"
	"github.com/stretchr/testify/assert"

	"github.com/jaegertracing/jaeger-operator/pkg/apis/io/v1alpha1"
)

func init() {
	viper.SetDefault("jaeger-version", "1.6")
	viper.SetDefault("jaeger-query-image", "jaegertracing/all-in-one")
}

func TestQueryNegativeSize(t *testing.T) {
	jaeger := v1alpha1.NewJaeger("TestQueryNegativeSize")
	jaeger.Spec.Query.Size = -1

	query := NewQuery(jaeger)
	dep := query.Get()
	assert.Equal(t, int32(1), *dep.Spec.Replicas)
}

func TestQueryDefaultSize(t *testing.T) {
	jaeger := v1alpha1.NewJaeger("TestQueryDefaultSize")
	jaeger.Spec.Query.Size = 0

	query := NewQuery(jaeger)
	dep := query.Get()
	assert.Equal(t, int32(1), *dep.Spec.Replicas)
}

func TestDefaultQueryImage(t *testing.T) {
	viper.Set("jaeger-query-image", "org/custom-query-image")
	viper.Set("jaeger-version", "123")
	defer viper.Reset()

	query := NewQuery(v1alpha1.NewJaeger("TestQueryImage"))
	dep := query.Get()
	containers := dep.Spec.Template.Spec.Containers

	assert.Len(t, containers, 1)
	assert.Equal(t, "org/custom-query-image:123", containers[0].Image)
}

func TestQueryPodName(t *testing.T) {
	name := "TestQueryPodName"
	query := NewQuery(v1alpha1.NewJaeger(name))
	dep := query.Get()

	assert.Contains(t, dep.ObjectMeta.Name, fmt.Sprintf("%s-query", name))
}

func TestQueryServices(t *testing.T) {
	query := NewQuery(v1alpha1.NewJaeger("TestQueryServices"))
	svcs := query.Services()

	assert.Len(t, svcs, 1)
}

func TestQueryIngresses(t *testing.T) {
	newBool := func(value bool) *bool {
		return &value
	}

	subTestCases := []struct {
		name                   string
		ingressSpec            v1alpha1.JaegerIngressSpec
		expectedIngressesCount int
	}{
		{
			name:                   "IngressEnabledDefault",
			ingressSpec:            v1alpha1.JaegerIngressSpec{},
			expectedIngressesCount: 1,
		},
		{
			name:                   "IngressEnabledFalse",
			ingressSpec:            v1alpha1.JaegerIngressSpec{Enabled: newBool(false)},
			expectedIngressesCount: 0,
		},
		{
			name:                   "IngressEnabledTrue",
			ingressSpec:            v1alpha1.JaegerIngressSpec{Enabled: newBool(true)},
			expectedIngressesCount: 1,
		},
	}

	for _, stc := range subTestCases {
		t.Run(stc.name, func(t *testing.T) {
			query := NewQuery(v1alpha1.NewJaeger("TestQueryIngresses"))
			query.jaeger.Spec.Query.Ingress = stc.ingressSpec
			ingresses := query.Ingresses()

			assert.Len(t, ingresses, stc.expectedIngressesCount)
		})
	}
}
