package configmap

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/jaegertracing/jaeger-operator/pkg/apis/io/v1alpha1"
)

func TestNoUIConfig(t *testing.T) {
	jaeger := v1alpha1.NewJaeger("TestNoUIConfig")

	config := NewUIConfig(jaeger)
	dep := config.Get()
	assert.Nil(t, dep)
}

func TestWithEmptyUIConfig(t *testing.T) {
	uiconfig := v1alpha1.NewFreeForm(map[string]interface{}{})
	jaeger := v1alpha1.NewJaeger("TestWithEmptyUIConfig")
	jaeger.Spec.UI.Options = uiconfig

	config := NewUIConfig(jaeger)
	dep := config.Get()
	assert.Nil(t, dep)
}

func TestWithUIConfig(t *testing.T) {
	uiconfig := v1alpha1.NewFreeForm(map[string]interface{}{
		"tracking": map[string]interface{}{
			"gaID": "UA-000000-2",
		},
	})
	json := `{"tracking":{"gaID":"UA-000000-2"}}`
	jaeger := v1alpha1.NewJaeger("TestWithUIConfig")
	jaeger.Spec.UI.Options = uiconfig

	config := NewUIConfig(jaeger)
	dep := config.Get()
	assert.Equal(t, json, dep.Data["ui"])
}

func TestUpdateNoUIConfig(t *testing.T) {
	jaeger := v1alpha1.NewJaeger("TestUpdateNoUIConfig")

	commonSpec := v1alpha1.JaegerCommonSpec{}
	options := []string{}

	Update(jaeger, &commonSpec, &options)
	assert.Len(t, commonSpec.Volumes, 0)
	assert.Len(t, commonSpec.VolumeMounts, 0)
	assert.Len(t, options, 0)
}

func TestUpdateWithUIConfig(t *testing.T) {
	uiconfig := v1alpha1.NewFreeForm(map[string]interface{}{
		"tracking": map[string]interface{}{
			"gaID": "UA-000000-2",
		},
	})
	jaeger := v1alpha1.NewJaeger("TestUpdateWithUIConfig")
	jaeger.Spec.UI.Options = uiconfig

	commonSpec := v1alpha1.JaegerCommonSpec{}
	options := []string{}

	Update(jaeger, &commonSpec, &options)
	assert.Len(t, commonSpec.Volumes, 1)
	assert.Len(t, commonSpec.VolumeMounts, 1)
	assert.Len(t, options, 1)
	assert.Equal(t, "--query.ui-config=/etc/config/ui.json", options[0])
}
