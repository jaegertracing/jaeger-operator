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
