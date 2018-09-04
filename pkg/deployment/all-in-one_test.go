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
	ingresses := NewAllInOne(v1alpha1.NewJaeger(name)).Ingresses()
	assert.Len(t, ingresses, 1)
}
