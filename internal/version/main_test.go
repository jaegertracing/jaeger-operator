package version

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestFallbackVersion(t *testing.T) {
	assert.Equal(t, "0.0.0", OpenTelemetryCollector())
}

func TestVersionFromBuild(t *testing.T) {
	// prepare
	jaeger = "0.0.2" // set during the build
	defer func() {
		jaeger = ""
	}()

	assert.Equal(t, jaeger, OpenTelemetryCollector())
	assert.Contains(t, Get().String(), jaeger)
}
