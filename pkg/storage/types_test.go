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

func TestValidTypes(t *testing.T) {
	assert.Len(t, ValidTypes(), 5)
	assert.Contains(t, ValidTypes(), "memory")
	assert.Contains(t, ValidTypes(), "elasticsearch")
	assert.Contains(t, ValidTypes(), "cassandra")
	assert.Contains(t, ValidTypes(), "kafka")
	assert.Contains(t, ValidTypes(), "badger")
}
