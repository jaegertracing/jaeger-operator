package upgrade

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestVersions(t *testing.T) {
	maptoTest := map[string]upgradeFunction{
		"1.17.0": noop,
		"1.15.4": noop,
		"1.15.0": noop,
		"1.16.1": noop,
		"1.12.2": noop,
	}
	sortedSemVersions := []string{
		"1.12.2", "1.15.0", "1.15.4", "1.16.1", "1.17.0",
	}

	semVersions, err := versions(maptoTest)
	assert.NoError(t, err)
	for i, v := range semVersions {
		assert.Equal(t, v.String(), sortedSemVersions[i])
	}
}

func TestVersionsError(t *testing.T) {
	maptoTest := map[string]upgradeFunction{
		"1.17.0": noop,
		"1.15.4": noop,
		"1.15.0": noop,
		"1,16.1": noop,
		"1.12.2": noop,
	}

	_, err := versions(maptoTest)
	assert.Error(t, err)
}

func TestVersionMapIsValid(t *testing.T) {
	_, err := versions(upgrades)
	assert.NoError(t, err)
}
