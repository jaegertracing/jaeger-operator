package version

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestDefaultJaegerMajorMinor(t *testing.T) {
	assert.Equal(t, "0.0", DefaultJaegerMajorMinor())
}
