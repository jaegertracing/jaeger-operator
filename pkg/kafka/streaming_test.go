package kafka

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/types"

	v1 "github.com/jaegertracing/jaeger-operator/pkg/apis/jaegertracing/v1"
)

func TestKafkaUserName(t *testing.T) {
	// prepare
	jaeger := v1.NewJaeger(types.NamespacedName{Name: "my-instance"})

	// test
	u := User(jaeger)

	// verify
	assert.Equal(t, jaeger.Name, u.GetName())

	contentMap, err := u.Spec.GetMap()
	assert.NoError(t, err)
	v, found, err := unstructured.NestedString(contentMap, "authentication", "type")
	assert.NoError(t, err)
	assert.True(t, found)
	assert.Equal(t, "tls", v)
}

func TestKafkaName(t *testing.T) {
	// prepare
	jaeger := v1.NewJaeger(types.NamespacedName{Name: "my-instance"})

	// test
	u := Persistent(jaeger)

	// verify
	assert.Equal(t, jaeger.Name, u.GetName())
}
