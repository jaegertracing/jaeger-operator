// Copyright The Jaeger Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package v2

type (
	// DeploymentStrategy represents the possible values for deployment strategies
	// +kubebuilder:validation:Enum=allinone;streaming;production
	DeploymentStrategy string
)

const (
	// DeploymentStrategyAllInOne represents the 'allInOne' deployment strategy (default).
	DeploymentStrategyAllInOne DeploymentStrategy = "allinone"

	// DeploymentStrategyStreaming represents the 'streaming' deployment strategy.
	DeploymentStrategyStreaming DeploymentStrategy = "streaming"

	// DeploymentStrategyProduction represents the 'production' deployment strategy.
	DeploymentStrategyProduction DeploymentStrategy = "production"
)
