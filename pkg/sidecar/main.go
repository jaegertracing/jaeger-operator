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

package sidecar

var (
	// Annotation is the annotation name to look for when deciding whether or not to inject.
	Annotation = "sidecar.jaegertracing.io/inject"
	// Label is the label name the operator put on injected deployments.
	Label = "sidecar.jaegertracing.io/injected"
)
