package v1

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestDefaultPrefix(t *testing.T) {
	assert.Equal(t, "anystorage", JaegerStorageType("anystorage").OptionsPrefix())
}

func TestElasticsearchPrefix(t *testing.T) {
	assert.Equal(t, "es", JaegerESStorage.OptionsPrefix())
}

func TestSpanStorageType(t *testing.T) {
	assert.Equal(t, "grpc", JaegerGRPCPluginStorage.SpanStorageType())
	assert.Equal(t, "cassandra", JaegerCassandraStorage.SpanStorageType())
	assert.Equal(t, "anystorage", JaegerStorageType("anystorage").SpanStorageType())
}

func TestValidTypes(t *testing.T) {
	assert.ElementsMatch(t, ValidStorageTypes(),
		[]JaegerStorageType{
			JaegerMemoryStorage,
			JaegerCassandraStorage,
			JaegerESStorage,
			JaegerKafkaStorage,
			JaegerBadgerStorage,
			JaegerGRPCPluginStorage,
		})
}
