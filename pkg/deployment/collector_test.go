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
	viper.SetDefault(collector, allInOneValue)
}

func TestNegativeSize(t *testing.T) {
	jaeger := v1alpha1.NewJaeger("TestNegativeSize")
	jaeger.Spec.Collector.Size = -1

	collector := NewCollector(jaeger)
	dep := collector.Get()
	assert.Equal(t, int32(1), *dep.Spec.Replicas)
}

func TestDefaultSize(t *testing.T) {
	jaeger := v1alpha1.NewJaeger("TestDefaultSize")
	jaeger.Spec.Collector.Size = 0

	collector := NewCollector(jaeger)
	dep := collector.Get()
	assert.Equal(t, int32(1), *dep.Spec.Replicas)
}

func TestName(t *testing.T) {
	collector := NewCollector(v1alpha1.NewJaeger("TestName"))
	dep := collector.Get()
	assert.Equal(t, "TestName-collector", dep.ObjectMeta.Name)
}

func TestCollectorServices(t *testing.T) {
	collector := NewCollector(v1alpha1.NewJaeger("TestName"))
	svcs := collector.Services()
	assert.Len(t, svcs, 2)
}

func TestDefaultCollectorImage(t *testing.T) {
	viper.Set(collector, "org/custom-collector-image")
	viper.Set(versionKey, "123")
	defer viper.Reset()

	collector := NewCollector(v1alpha1.NewJaeger("TestCollectorImage"))
	dep := collector.Get()

	containers := dep.Spec.Template.Spec.Containers
	assert.Len(t, containers, 1)
	assert.Equal(t, "org/custom-collector-image:123", containers[0].Image)

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
	assert.Equal(t, envvars, containers[0].Env)
}
