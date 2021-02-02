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

const (
	// LabelOperatedBy is used as the key to the label indicating which operator is managing the instance.
	LabelOperatedBy string = "jaegertracing.io/operated-by"

	// ConfigIdentity is the key to the configuration map related to the operator's identity.
	ConfigIdentity string = "identity"
)
