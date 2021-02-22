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

package config

const JaegerType = "jaeger"

// Jaeger Receiver Settings

type JaegerReceiverConfig struct {
	Protocols Protocols
}

func (*JaegerReceiverConfig) Type() string {
	return JaegerType
}

// Jaeger exporter settings

type JaegerExporterConfig struct {
	GRPCSettings `yaml:",inline,omitempty"`
}

func (*JaegerExporterConfig) Type() string {
	return JaegerType
}
