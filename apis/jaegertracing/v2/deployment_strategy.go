package v2

type (
	// DeploymentStrategy represents the possible values for deployment strategies
	// +kubebuilder:validation:Enum=allinone;streaming;production
	DeploymentStrategy string
)

const (
	// DeploymentStrategyAllInOne represents the 'allInOne' deployment strategy (default)
	DeploymentStrategyAllInOne DeploymentStrategy = "allinone"

	// DeploymentStrategyStreaming represents the 'streaming' deployment strategy
	DeploymentStrategyStreaming DeploymentStrategy = "streaming"

	// DeploymentStrategyProduction represents the 'production' deployment strategy
	DeploymentStrategyProduction DeploymentStrategy = "production"
)
