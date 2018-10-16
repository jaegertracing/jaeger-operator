package deployment

import (
	"testing"

	"github.com/spf13/viper"
	"github.com/stretchr/testify/assert"
	"k8s.io/api/core/v1"

	"github.com/jaegertracing/jaeger-operator/pkg/apis/io/v1alpha1"
)

func init() {
	viper.SetDefault(versionKey, versionValue)
	viper.SetDefault(allInOneImageKey, allInOneValue)
}

func TestDefaultAllInOneImage(t *testing.T) {
	viper.Set(allInOneImageKey, "org/custom-all-in-one-image")
	viper.Set(versionKey, "123")
	defer viper.Reset()

	d := NewAllInOne(v1alpha1.NewJaeger("TestAllInOneDefaultImage")).Get()

	assert.Len(t, d.Spec.Template.Spec.Containers, 1)
	assert.Equal(t, "org/custom-all-in-one-image:123", d.Spec.Template.Spec.Containers[0].Image)

	envvars := []v1.EnvVar{
		v1.EnvVar{
			Name:  "SPAN_STORAGE_TYPE",
			Value: "",
		},
		v1.EnvVar{
			Name:  "COLLECTOR_ZIPKIN_HTTP_PORT",
			Value: "9411",
		},
	}
	assert.Equal(t, envvars, d.Spec.Template.Spec.Containers[0].Env)
}

func TestAllInOneHasOwner(t *testing.T) {
	name := "TestAllInOneHasOwner"
	a := NewAllInOne(v1alpha1.NewJaeger(name))
	assert.Equal(t, name, a.Get().ObjectMeta.Name)
}

func TestAllInOneNumberOfServices(t *testing.T) {
	name := "TestNumberOfServices"
	services := NewAllInOne(v1alpha1.NewJaeger(name)).Services()
	assert.Len(t, services, 4) // collector, query, agent,zipkin

	for _, svc := range services {
		owners := svc.ObjectMeta.OwnerReferences
		assert.Equal(t, name, owners[0].Name)
	}
}

func TestAllInOneNumberOfIngresses(t *testing.T) {
	name := "TestAllInOneNumberOfIngresses"
	newBool := func(value bool) *bool {
		return &value
	}

	subTestCases := []struct {
		name                   string
		ingressSpec            v1alpha1.JaegerIngressSpec
		expectedIngressesCount int
	}{
		{
			name:                   ingressEnabledDefault,
			ingressSpec:            v1alpha1.JaegerIngressSpec{},
			expectedIngressesCount: 1,
		},
		{
			name:                   ingressEnabledFalse,
			ingressSpec:            v1alpha1.JaegerIngressSpec{Enabled: newBool(false)},
			expectedIngressesCount: 0,
		},
		{
			name:                   ingressEnabledTrue,
			ingressSpec:            v1alpha1.JaegerIngressSpec{Enabled: newBool(true)},
			expectedIngressesCount: 1,
		},
	}

	for _, stc := range subTestCases {
		t.Run(stc.name, func(t *testing.T) {
			jaeger := v1alpha1.NewJaeger(name)
			jaeger.Spec.AllInOne.Ingress = stc.ingressSpec
			ingresses := NewAllInOne(jaeger).Ingresses()
			assert.Len(t, ingresses, stc.expectedIngressesCount)
		})
	}
}
