package v1alpha1

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestFreeForm(t *testing.T) {
	uiconfig := `{"es":{"password":"changeme","server-urls":"http://elasticsearch:9200","username":"elastic"}}`
	o := NewFreeForm(map[string]interface{}{
		"es": map[string]interface{}{
			"server-urls": "http://elasticsearch:9200",
			"username":    "elastic",
			"password":    "changeme",
		},
	})
	json, err := o.MarshalJSON()
	assert.NoError(t, err)
	assert.NotNil(t, json)
	assert.Equal(t, uiconfig, string(o.json))
}

func TestFreeFormUnmarhalMarshal(t *testing.T) {
	uiconfig := `{"es":{"password":"changeme","server-urls":"http://elasticsearch:9200","username":"elastic"}}`
	o := NewFreeForm(nil)
	o.UnmarshalJSON([]byte(uiconfig))
	json, err := o.MarshalJSON()
	assert.NoError(t, err)
	assert.NotNil(t, json)
	assert.Equal(t, uiconfig, string(o.json))
}

func TestFreeFormIsEmptyFalse(t *testing.T) {
	o := NewFreeForm(map[string]interface{}{
		"es": map[string]interface{}{
			"server-urls": "http://elasticsearch:9200",
			"username":    "elastic",
			"password":    "changeme",
		},
	})
	assert.False(t, o.IsEmpty())
}

func TestFreeFormIsEmptyTrue(t *testing.T) {
	o := NewFreeForm(map[string]interface{}{})
	assert.True(t, o.IsEmpty())
}

func TestFreeFormIsEmptyNilTrue(t *testing.T) {
	o := NewFreeForm(nil)
	assert.True(t, o.IsEmpty())
}
