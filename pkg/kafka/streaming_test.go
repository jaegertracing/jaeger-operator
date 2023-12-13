package kafka

import (
	"testing"

	"github.com/spf13/viper"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/types"

	v1 "github.com/jaegertracing/jaeger-operator/apis/v1"
)

func TestKafkaUserName(t *testing.T) {
	// prepare
	jaeger := v1.NewJaeger(types.NamespacedName{Name: "my-instance"})

	// test
	u := User(jaeger)

	// verify
	assert.Equal(t, jaeger.Name, u.GetName())

	contentMap, err := u.Spec.GetMap()
	require.NoError(t, err)
	v, found, err := unstructured.NestedString(contentMap, "authentication", "type")
	require.NoError(t, err)
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

func TestKafkaSizing(t *testing.T) {
	// prepare
	jaeger := v1.NewJaeger(types.NamespacedName{Name: "my-instance"})

	// test
	u := Persistent(jaeger)

	// verify
	contentMap, err := u.Spec.GetMap()
	require.NoError(t, err)
	v, found, err := unstructured.NestedFieldNoCopy(contentMap, "kafka", "replicas")
	require.NoError(t, err)
	assert.True(t, found)
	assert.EqualValues(t, 3, v)

	storage, found, err := unstructured.NestedMap(contentMap, "kafka", "storage")
	require.NoError(t, err)
	assert.True(t, found)

	volumes, found, err := unstructured.NestedSlice(storage, "volumes")
	require.NoError(t, err)
	assert.True(t, found)
	assert.Len(t, volumes, 1)
	assert.Equal(t, "100Gi", volumes[0].(map[string]interface{})["size"])

	v, found, err = unstructured.NestedFieldNoCopy(contentMap, "zookeeper", "replicas")
	require.NoError(t, err)
	assert.True(t, found)
	assert.EqualValues(t, 3, v)
}

func TestKafkaMinimalSizing(t *testing.T) {
	// prepare
	viper.Set("kafka-provisioning-minimal", true)
	defer viper.Reset()
	jaeger := v1.NewJaeger(types.NamespacedName{Name: "my-instance"})

	// test
	u := Persistent(jaeger)

	// verify
	contentMap, err := u.Spec.GetMap()
	require.NoError(t, err)
	v, found, err := unstructured.NestedFieldNoCopy(contentMap, "kafka", "replicas")
	require.NoError(t, err)
	assert.True(t, found)
	assert.EqualValues(t, 1, v)

	storage, found, err := unstructured.NestedMap(contentMap, "kafka", "storage")
	require.NoError(t, err)
	assert.True(t, found)

	volumes, found, err := unstructured.NestedSlice(storage, "volumes")
	require.NoError(t, err)
	assert.True(t, found)
	assert.Len(t, volumes, 1)
	assert.Equal(t, "10Gi", volumes[0].(map[string]interface{})["size"])

	v, found, err = unstructured.NestedFieldNoCopy(contentMap, "zookeeper", "replicas")
	require.NoError(t, err)
	assert.True(t, found)
	assert.EqualValues(t, 1, v)
}
