package v1

import (
	"errors"
	"strings"
)

// DeploymentStrategy represents the possible values for deployment strategies
// +k8s:openapi-gen=true
type DeploymentStrategy string

const (
	// DeploymentStrategyDeprecatedAllInOne represents the (deprecated) 'all-in-one' deployment strategy
	// +k8s:openapi-gen=true
	DeploymentStrategyDeprecatedAllInOne DeploymentStrategy = "all-in-one"

	// DeploymentStrategyAllInOne represents the 'allInOne' deployment strategy (default)
	// +k8s:openapi-gen=true
	DeploymentStrategyAllInOne DeploymentStrategy = "allinone"

	// DeploymentStrategyStreaming represents the 'streaming' deployment strategy
	// +k8s:openapi-gen=true
	DeploymentStrategyStreaming DeploymentStrategy = "streaming"

	// DeploymentStrategyProduction represents the 'production' deployment strategy
	// +k8s:openapi-gen=true
	DeploymentStrategyProduction DeploymentStrategy = "production"
)

// UnmarshalText implements encoding.TextUnmarshaler to ensure that JSON values in the
// strategy field of JSON jaeger specs are interpreted in a case-insensitive manner
func (ds *DeploymentStrategy) UnmarshalText(text []byte) error {
	if ds == nil {
		return errors.New("DeploymentStrategy: UnmarshalText on nil pointer")
	}

	switch strings.ToLower(string(text)) {
	default:
		*ds = DeploymentStrategyAllInOne
	case string(DeploymentStrategyDeprecatedAllInOne):
		*ds = DeploymentStrategyDeprecatedAllInOne
	case string(DeploymentStrategyStreaming):
		*ds = DeploymentStrategyStreaming
	case string(DeploymentStrategyProduction):
		*ds = DeploymentStrategyProduction
	}

	return nil
}
