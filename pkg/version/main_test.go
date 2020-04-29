package version

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestDefaultJaegerMajorMinor(t *testing.T) {
	original := defaultJaeger
	defaultJaeger = "0.0.0"
	defer func() {
		defaultJaeger = original
	}()
	assert.Equal(t, "0.0", DefaultJaegerMajorMinor())
}
