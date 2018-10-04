package deployment

import (
	"testing"

	"github.com/spf13/viper"
	"github.com/stretchr/testify/assert"

	"github.com/jaegertracing/jaeger-operator/pkg/apis/io/v1alpha1"
)

func init() {
	viper.SetDefault("jaeger-version", "1.6")
	viper.SetDefault("jaeger-all-in-one-image", "jaegertracing/all-in-one")
}

func TestDefaultAllInOneImage(t *testing.T) {
	viper.Set("jaeger-all-in-one-image", "org/custom-all-in-one-image")
	viper.Set("jaeger-version", "123")
	defer viper.Reset()

	d := NewAllInOne(v1alpha1.NewJaeger("TestAllInOneDefaultImage")).Get()

	assert.Len(t, d.Spec.Template.Spec.Containers, 1)
	assert.Equal(t, "org/custom-all-in-one-image:123", d.Spec.Template.Spec.Containers[0].Image)
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
			jaeger := v1alpha1.NewJaeger(name)
			jaeger.Spec.AllInOne.Ingress = stc.ingressSpec
			ingresses := NewAllInOne(jaeger).Ingresses()
			assert.Len(t, ingresses, stc.expectedIngressesCount)
		})
	}
}

func TestPrometheusAnnotations(t *testing.T) {
	name := "TestPrometheusAnnotations"
	a := NewAllInOne(v1alpha1.NewJaeger(name))
	annotations := a.Get().Annotations
	assert.Equal(t, 2, len(annotations))

	for k := range annotations {
		assert.Contains(t, k, "prometheus.io/")
	}
}

func TestAllInOneLabels(t *testing.T) {
	name := "TestAllInOneLabels"
	k, v := "some-label-name", "some-label-value"
	labels := map[string]string{k: v}

	j := v1alpha1.NewJaeger(name)
	j.Spec.AllInOne.Labels = labels

	a := NewAllInOne(j)

	// test the deployments
	dep := a.Get()
	assert.Equal(t, len(labels)+len(a.selector()), len(dep.Labels))
	assert.Equal(t, len(labels)+len(a.selector()), len(dep.Spec.Template.Labels))
	assert.Equal(t, v, dep.Labels[k])
	assert.Equal(t, v, dep.Spec.Template.Labels[k])

	// then the services
	for _, svc := range a.Services() {
		assert.Equal(t, len(labels)+len(a.selector()), len(svc.Labels), "Wrong label count for service %v", svc.Name)
		assert.Equal(t, v, svc.Labels[k], "Couldn't find %v for service %v", k, svc.Name)
	}

	// and finally, ingresses
	i := a.Ingresses()[0]
	assert.Equal(t, len(labels), len(i.Labels))
	assert.Equal(t, v, i.Labels[k])
}

func TestAllInOneAnnotations(t *testing.T) {
	name := "TestAllInOneAnnotations"
	k, v := "some-annotation-name", "some-annotation-value"
	annotations := map[string]string{k: v}

	j := v1alpha1.NewJaeger(name)
	j.Spec.AllInOne.Annotations = annotations

	a := NewAllInOne(j)

	// test the deployments
	dep := a.Get()
	assert.Equal(t, len(annotations)+2, len(dep.ObjectMeta.Annotations))    // see TestPrometheusAnnotations
	assert.Equal(t, len(annotations)+2, len(dep.Spec.Template.Annotations)) // see TestPrometheusAnnotations
	assert.Equal(t, v, dep.Annotations[k])
	assert.Equal(t, v, dep.Spec.Template.Annotations[k])

	// then the services
	for _, svc := range a.Services() {
		assert.Equal(t, len(annotations), len(svc.Annotations))
		assert.Equal(t, v, svc.Annotations[k])
	}

	// and finally, ingresses
	i := a.Ingresses()[0]
	assert.Equal(t, len(annotations), len(i.Annotations))
	assert.Equal(t, v, i.Annotations[k])
}
