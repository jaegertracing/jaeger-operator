package v1

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestUnmarshalJSON(t *testing.T) {
	tcs := map[string]struct {
		json     string
		expected DeploymentStrategy
	}{
		"allInOne":     {json: `"allInOne"`, expected: DeploymentStrategyAllInOne},
		"streaming":    {json: `"streaming"`, expected: DeploymentStrategyStreaming},
		"production":   {json: `"production"`, expected: DeploymentStrategyProduction},
		"all-in-one":   {json: `"all-in-one"`, expected: DeploymentStrategyDeprecatedAllInOne},
		"ALLinONE":     {json: `"ALLinONE"`, expected: DeploymentStrategyAllInOne},
		"StReAmInG":    {json: `"StReAmInG"`, expected: DeploymentStrategyStreaming},
		"Production":   {json: `"Production"`, expected: DeploymentStrategyProduction},
		"All-IN-One":   {json: `"All-IN-One"`, expected: DeploymentStrategyDeprecatedAllInOne},
		"random value": {json: `"random value"`, expected: DeploymentStrategyAllInOne},
		"empty string": {json: `""`, expected: DeploymentStrategyAllInOne},
	}

	for name, tc := range tcs {
		t.Run(name, func(t *testing.T) {
			ds := DeploymentStrategy("")
			err := json.Unmarshal([]byte(tc.json), &ds)
			require.NoError(t, err)
			assert.Equal(t, tc.expected, ds)
		})
	}
}

func TestMarshalJSON(t *testing.T) {
	tcs := map[string]struct {
		strategy DeploymentStrategy
		expected string
	}{
		"allinone":   {strategy: DeploymentStrategyAllInOne, expected: `"allinone"`},
		"streaming":  {strategy: DeploymentStrategyStreaming, expected: `"streaming"`},
		"production": {strategy: DeploymentStrategyProduction, expected: `"production"`},
		"all-in-one": {strategy: DeploymentStrategyDeprecatedAllInOne, expected: `"all-in-one"`},
	}

	for name, tc := range tcs {
		t.Run(name, func(t *testing.T) {
			data, err := json.Marshal(tc.strategy)
			require.NoError(t, err)
			assert.Equal(t, tc.expected, string(data))
		})
	}
}
