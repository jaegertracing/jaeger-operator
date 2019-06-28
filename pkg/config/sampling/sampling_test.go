package sampling

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"k8s.io/apimachinery/pkg/types"

	v1 "github.com/jaegertracing/jaeger-operator/pkg/apis/jaegertracing/v1"
)

func TestNoSamplingConfig(t *testing.T) {
	jaeger := v1.NewJaeger(types.NamespacedName{Name: "TestNoSamplingConfig"})

	config := NewConfig(jaeger)
	cm := config.Get()
	assert.NotNil(t, cm)
	assert.Equal(t, defaultSamplingStrategy, cm.Data["sampling"])
}

func TestWithEmptySamplingConfig(t *testing.T) {
	uiconfig := v1.NewFreeForm(map[string]interface{}{})
	jaeger := v1.NewJaeger(types.NamespacedName{Name: "TestWithEmptySamplingConfig"})
	jaeger.Spec.UI.Options = uiconfig

	config := NewConfig(jaeger)
	cm := config.Get()
	assert.NotNil(t, cm)
	assert.Equal(t, defaultSamplingStrategy, cm.Data["sampling"])
}

func TestWithSamplingConfig(t *testing.T) {
	samplingconfig := v1.NewFreeForm(map[string]interface{}{
		"default_strategy": map[string]interface{}{
			"type":  "probabilistic",
			"param": "20",
		},
	})
	json := `{"default_strategy":{"param":"20","type":"probabilistic"}}`
	jaeger := v1.NewJaeger(types.NamespacedName{Name: "TestWithSamplingConfig"})
	jaeger.Spec.Sampling.Options = samplingconfig

	config := NewConfig(jaeger)
	cm := config.Get()
	assert.Equal(t, json, cm.Data["sampling"])
}

func TestUpdateNoSamplingConfig(t *testing.T) {
	jaeger := v1.NewJaeger(types.NamespacedName{Name: "TestUpdateNoSamplingConfig"})

	commonSpec := v1.JaegerCommonSpec{}
	options := []string{}

	Update(jaeger, &commonSpec, &options)
	assert.Len(t, commonSpec.Volumes, 1)
	assert.Equal(t, "testupdatenosamplingconfig-sampling-configuration-volume", commonSpec.Volumes[0].Name)
	assert.Len(t, commonSpec.VolumeMounts, 1)
	assert.Equal(t, "testupdatenosamplingconfig-sampling-configuration-volume", commonSpec.VolumeMounts[0].Name)
	assert.Len(t, options, 1)
	assert.Equal(t, "--sampling.strategies-file=/etc/jaeger/sampling/sampling.json", options[0])
}

func TestUpdateWithSamplingConfig(t *testing.T) {
	uiconfig := v1.NewFreeForm(map[string]interface{}{
		"tracking": map[string]interface{}{
			"gaID": "UA-000000-2",
		},
	})
	jaeger := v1.NewJaeger(types.NamespacedName{Name: "TestUpdateWithSamplingConfig"})
	jaeger.Spec.UI.Options = uiconfig

	commonSpec := v1.JaegerCommonSpec{}
	options := []string{}

	Update(jaeger, &commonSpec, &options)
	assert.Len(t, commonSpec.Volumes, 1)
	assert.Equal(t, "testupdatewithsamplingconfig-sampling-configuration-volume", commonSpec.Volumes[0].Name)
	assert.Len(t, commonSpec.VolumeMounts, 1)
	assert.Equal(t, "testupdatewithsamplingconfig-sampling-configuration-volume", commonSpec.VolumeMounts[0].Name)
	assert.Len(t, options, 1)
	assert.Equal(t, "--sampling.strategies-file=/etc/jaeger/sampling/sampling.json", options[0])
}
