package storage

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestDefaultPrefix(t *testing.T) {
	assert.Equal(t, "anystorage", OptionsPrefix("anystorage"))
}

func TestElasticsearchPrefix(t *testing.T) {
	assert.Equal(t, "es", OptionsPrefix("elasticsearch"))
}

func TestGRPCPluginPrefix(t *testing.T) {
	assert.Equal(t, "grpc-storage-plugin", OptionsPrefix("grpc-plugin"))
}

func TestValidTypes(t *testing.T) {
	assert.Len(t, ValidTypes(), 6)
	assert.Contains(t, ValidTypes(), "memory")
	assert.Contains(t, ValidTypes(), "elasticsearch")
	assert.Contains(t, ValidTypes(), "cassandra")
	assert.Contains(t, ValidTypes(), "kafka")
	assert.Contains(t, ValidTypes(), "badger")
	assert.Contains(t, ValidTypes(), "grpc-plugin")
}
