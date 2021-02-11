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

package agent

import (
	"fmt"

	otelv1alpha1 "github.com/open-telemetry/opentelemetry-operator/api/v1alpha1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/jaegertracing/jaeger-operator/internal/config"

	"errors"

	"gopkg.in/yaml.v2"

	jaegertracingv2 "github.com/jaegertracing/jaeger-operator/apis/jaegertracing/v2"
	"github.com/jaegertracing/jaeger-operator/pkg/naming"
	"github.com/jaegertracing/jaeger-operator/pkg/util"
)

// TODO: Better way of doing this..
func DefaultConfig() string {
	return `
    receivers:
      otlp:
        protocols:
          grpc:
      jaeger:
        protocols:
          grpc:
    exporters:
      jaeger:
        endpoint: xxxx

    service:
      pipelines:
        traces:
          receivers: [otlp, jaeger]
          exporters: [jaeger]`
}

var (
	// ErrInvalidYAML represents an error in the format of the configuration file.
	ErrInvalidYAML = errors.New("couldn't parse the opentelemetry-collector configuration")
)

func configFromString(configStr string) (map[interface{}]interface{}, error) {
	config := make(map[interface{}]interface{})
	if err := yaml.Unmarshal([]byte(configStr), &config); err != nil {
		return nil, ErrInvalidYAML
	}

	return config, nil
}

func stringFromConfig(cfg map[interface{}]interface{}) (string, error) {
	out, err := yaml.Marshal(cfg)
	if err != nil {
		return "", err
	}
	return string(out), nil

}

func otelModeFromStrategy(strategy jaegertracingv2.AgentStrategy) otelv1alpha1.Mode {
	if strategy == jaegertracingv2.AgentDaemonSet {
		return otelv1alpha1.ModeDaemonSet
	}
	return otelv1alpha1.ModeSidecar
}

func setCollectorEndpoint(instance jaegertracingv2.Jaeger, confMap map[interface{}]interface{}) map[interface{}]interface{} {
	exportersProperty := confMap["exporters"]
	exporters := exportersProperty.(map[interface{}]interface{})
	jaegerProperty := exporters["jaeger"]
	jaegerExporter := jaegerProperty.(map[interface{}]interface{})
	jaegerExporter["endpoint"] = fmt.Sprintf("%s.svc:14250", naming.CollectorHeadlessService(instance))
	return confMap
}

func Get(jaeger jaegertracingv2.Jaeger, cfg config.Config) *otelv1alpha1.OpenTelemetryCollector {

	configString := jaeger.Spec.Agent.Config
	if configString == "" {
		configString = DefaultConfig()
	}

	confMap, err := configFromString(configString)
	if err != nil {
		// TODO:  Return an error and handle it on the reconciliation
		return nil
	}

	confMap = setCollectorEndpoint(jaeger, confMap)

	configString, _ = stringFromConfig(confMap)

	agentSpecs := jaeger.Spec.Agent
	commonSpecs := util.Merge(jaeger.Spec.JaegerCommonSpec, agentSpecs.JaegerCommonSpec)

	return &otelv1alpha1.OpenTelemetryCollector{
		ObjectMeta: metav1.ObjectMeta{
			Name:      naming.Agent(jaeger),
			Namespace: jaeger.Namespace,
		},
		Spec: otelv1alpha1.OpenTelemetryCollectorSpec{
			Image:          naming.Image(jaeger.Spec.Agent.Image, cfg.CollectorImage(), cfg.JaegerVersion()),
			Config:         configString,
			ServiceAccount: commonSpecs.ServiceAccount,
			VolumeMounts:   commonSpecs.VolumeMounts,
			Volumes:        commonSpecs.Volumes,
			Mode:           otelModeFromStrategy(agentSpecs.Strategy),
		},
	}
}
