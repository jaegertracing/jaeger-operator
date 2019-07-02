package configmap

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"k8s.io/apimachinery/pkg/types"

	v1 "github.com/jaegertracing/jaeger-operator/pkg/apis/jaegertracing/v1"
)

func TestNoUIConfig(t *testing.T) {
	jaeger := v1.NewJaeger(types.NamespacedName{Name: "TestNoUIConfig"})

	config := NewUIConfig(jaeger)
	dep := config.Get()
	assert.Nil(t, dep)
}

func TestWithEmptyUIConfig(t *testing.T) {
	uiconfig := v1.NewFreeForm(map[string]interface{}{})
	jaeger := v1.NewJaeger(types.NamespacedName{Name: "TestWithEmptyUIConfig"})
	jaeger.Spec.UI.Options = uiconfig

	config := NewUIConfig(jaeger)
	dep := config.Get()
	assert.Nil(t, dep)
}

func TestWithUIConfig(t *testing.T) {
	uiconfig := v1.NewFreeForm(map[string]interface{}{
		"tracking": map[string]interface{}{
			"gaID": "UA-000000-2",
		},
	})
	json := `{"tracking":{"gaID":"UA-000000-2"}}`
	jaeger := v1.NewJaeger(types.NamespacedName{Name: "TestWithUIConfig"})
	jaeger.Spec.UI.Options = uiconfig

	config := NewUIConfig(jaeger)
	dep := config.Get()
	assert.Equal(t, json, dep.Data["ui"])
}

func TestUpdateNoUIConfig(t *testing.T) {
	jaeger := v1.NewJaeger(types.NamespacedName{Name: "TestUpdateNoUIConfig"})

	commonSpec := v1.JaegerCommonSpec{}
	options := []string{}

	Update(jaeger, &commonSpec, &options)
	assert.Len(t, commonSpec.Volumes, 0)
	assert.Len(t, commonSpec.VolumeMounts, 0)
	assert.Len(t, options, 0)
}

func TestUpdateWithUIConfig(t *testing.T) {
	uiconfig := v1.NewFreeForm(map[string]interface{}{
		"tracking": map[string]interface{}{
			"gaID": "UA-000000-2",
		},
	})
	jaeger := v1.NewJaeger(types.NamespacedName{Name: "TestUpdateWithUIConfig"})
	jaeger.Spec.UI.Options = uiconfig

	commonSpec := v1.JaegerCommonSpec{}
	options := []string{}

	Update(jaeger, &commonSpec, &options)
	assert.Len(t, commonSpec.Volumes, 1)
	assert.Len(t, commonSpec.VolumeMounts, 1)
	assert.Len(t, options, 1)
	assert.Equal(t, "--query.ui-config=/etc/config/ui.json", options[0])
}
