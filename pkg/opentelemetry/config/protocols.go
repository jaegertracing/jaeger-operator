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

type Protocols struct {
	GRPC          *GRPCSettings `yaml:"grpc,omitempty"`
	ThriftHTTP    *HttpSettings `yaml:"thrift_http,omitempty"`
	ThriftCompact *UDPSettings  `yaml:"thrift_compact,omitempty"`
}

type GRPCSettings struct {
	Endpoint  string `yaml:"endpoint,omitempty"`
	TLSConfig `yaml:",inline,omitempty"`
}

type HttpSettings struct {
	Endpoint string `yaml:"endpoint,omitempty"`
}

type UDPSettings struct {
	Endpoint string `yaml:"endpoint,omitempty"`
}

type TLSConfig struct {
	Insecure bool `yaml:"insecure,omitempty"`
}

func (h HttpSettings) MarshalYAML() (interface{}, error) {
	defaultValues := HttpSettings{}
	if defaultValues == h {
		// Empty protocols should be replaced by null/nil on otel yaml config file
		return nil, nil
	}
	return h, nil
}

func (u UDPSettings) MarshalYAML() (interface{}, error) {
	defaultValues := UDPSettings{}
	if defaultValues == u {
		// Empty protocols should be replaced by null/nil on otel yaml config file
		return nil, nil
	}
	return u, nil
}

func (grpc GRPCSettings) MarshalYAML() (interface{}, error) {
	defaultValues := GRPCSettings{}
	if defaultValues == grpc {
		// Empty protocols should be replaced by null/nil on otel yaml config file
		return nil, nil
	}
	return grpc, nil
}
