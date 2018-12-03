package sampling

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/jaegertracing/jaeger-operator/pkg/apis/io/v1alpha1"
)

func TestNoSamplingConfig(t *testing.T) {
	jaeger := v1alpha1.NewJaeger("TestNoSamplingConfig")

	config := NewConfig(jaeger)
	cm := config.Get()
	assert.NotNil(t, cm)
	assert.Equal(t, defaultSamplingStrategy, cm.Data["sampling"])
}

func TestWithEmptySamplingConfig(t *testing.T) {
	uiconfig := v1alpha1.NewFreeForm(map[string]interface{}{})
	jaeger := v1alpha1.NewJaeger("TestWithEmptySamplingConfig")
	jaeger.Spec.UI.Options = uiconfig

	config := NewConfig(jaeger)
	cm := config.Get()
	assert.NotNil(t, cm)
	assert.Equal(t, defaultSamplingStrategy, cm.Data["sampling"])
}

func TestWithSamplingConfig(t *testing.T) {
	samplingconfig := v1alpha1.NewFreeForm(map[string]interface{}{
		"default_strategy": map[string]interface{}{
			"type":  "probabilistic",
			"param": "20",
		},
	})
	json := `{"default_strategy":{"param":"20","type":"probabilistic"}}`
	jaeger := v1alpha1.NewJaeger("TestWithSamplingConfig")
	jaeger.Spec.Sampling.Options = samplingconfig

	config := NewConfig(jaeger)
	dep := config.Get()
	assert.Equal(t, json, dep.Data["sampling"])
}

func TestUpdateNoSamplingConfig(t *testing.T) {
	jaeger := v1alpha1.NewJaeger("TestUpdateNoSamplingConfig")

	commonSpec := v1alpha1.JaegerCommonSpec{}
	options := []string{}

	Update(jaeger, &commonSpec, &options)
	assert.Len(t, commonSpec.Volumes, 1)
	assert.Equal(t, "TestUpdateNoSamplingConfig-sampling-configuration-volume", commonSpec.Volumes[0].Name)
	assert.Len(t, commonSpec.VolumeMounts, 1)
	assert.Equal(t, "TestUpdateNoSamplingConfig-sampling-configuration-volume", commonSpec.VolumeMounts[0].Name)
	assert.Len(t, options, 1)
	assert.Equal(t, "--sampling.strategies-file=/etc/jaeger/sampling/sampling.json", options[0])
}

func TestUpdateWithSamplingConfig(t *testing.T) {
	uiconfig := v1alpha1.NewFreeForm(map[string]interface{}{
		"tracking": map[string]interface{}{
			"gaID": "UA-000000-2",
		},
	})
	jaeger := v1alpha1.NewJaeger("TestUpdateWithSamplingConfig")
	jaeger.Spec.UI.Options = uiconfig

	commonSpec := v1alpha1.JaegerCommonSpec{}
	options := []string{}

	Update(jaeger, &commonSpec, &options)
	assert.Len(t, commonSpec.Volumes, 1)
	assert.Equal(t, "TestUpdateWithSamplingConfig-sampling-configuration-volume", commonSpec.Volumes[0].Name)
	assert.Len(t, commonSpec.VolumeMounts, 1)
	assert.Equal(t, "TestUpdateWithSamplingConfig-sampling-configuration-volume", commonSpec.VolumeMounts[0].Name)
	assert.Len(t, options, 1)
	assert.Equal(t, "--sampling.strategies-file=/etc/jaeger/sampling/sampling.json", options[0])
}
