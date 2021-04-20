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

import "gopkg.in/yaml.v2"

type Exporter interface {
	Type() string
}

type Receiver interface {
	Type() string
}

type Options func(configuration *Configuration)

func WithReceiver(receiver Receiver) Options {
	return func(configuration *Configuration) {
		configuration.Receiver(receiver)
	}
}

func WithExporter(exporter Exporter) Options {
	return func(configuration *Configuration) {
		configuration.Exporter(exporter)
	}
}

func NewConfiguration(options ...Options) *Configuration {
	cfg := &Configuration{
		Receivers: map[string]Receiver{},
		Exporters: map[string]Exporter{},
	}
	for _, option := range options {
		option(cfg)
	}
	return cfg
}

type Configuration struct {
	Exporters map[string]Exporter `yaml:"exporters,omitempty"`
	Receivers map[string]Receiver `yaml:"receivers,omitempty"`
	Service   serviceSettings
}

type serviceSettings struct {
	Extensions []string `yaml:"extensions,flow,omitempty"`
	Pipelines  tracesPipeline
}

type tracesPipeline struct {
	Traces pipelineSettings
}

type pipelineSettings struct {
	Receivers  []string `yaml:"receivers,flow,omitempty"`
	Processors []string `yaml:"processors,flow,omitempty"`
	Exporters  []string `yaml:"exporters,flow,omitempty"`
}

func (cfg *Configuration) String() (string, error) {
	bytes, err := yaml.Marshal(cfg)
	if err != nil {
		return "", err
	}
	return string(bytes), nil
}

func (cfg *Configuration) Exporter(exporter Exporter) {
	_, hasExporter := cfg.Exporters[exporter.Type()]
	cfg.Exporters[exporter.Type()] = exporter
	if !hasExporter {
		cfg.Service.Pipelines.Traces.Exporters = append(cfg.Service.Pipelines.Traces.Exporters, exporter.Type())
	}

}

func (cfg *Configuration) Receiver(receiver Receiver) {
	_, hasReceiver := cfg.Receivers[receiver.Type()]
	cfg.Receivers[receiver.Type()] = receiver
	if !hasReceiver {
		cfg.Service.Pipelines.Traces.Receivers = append(cfg.Service.Pipelines.Traces.Receivers, receiver.Type())

	}
}

func (cfg *Configuration) GetJaegerExporter() *JaegerExporterConfig {
	exporter := cfg.Exporters[JaegerType]
	return exporter.(*JaegerExporterConfig)
}
