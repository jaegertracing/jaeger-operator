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
	otelconfig "github.com/jaegertracing/jaeger-operator/pkg/opentelemetry/config"

	jaegertracingv2 "github.com/jaegertracing/jaeger-operator/apis/jaegertracing/v2"
	"github.com/jaegertracing/jaeger-operator/pkg/naming"
	"github.com/jaegertracing/jaeger-operator/pkg/util"
)

func defaultConfig() *otelconfig.Configuration {
	return otelconfig.NewConfiguration(
		otelconfig.WithExporter(&otelconfig.JaegerExporterConfig{
			GRPCSettings: otelconfig.GRPCSettings{
				TLSConfig: otelconfig.TLSConfig{
					Insecure: true,
				},
			},
		}),
		otelconfig.WithReceiver(
			&otelconfig.JaegerReceiverConfig{
				Protocols: otelconfig.Protocols{
					GRPC:          &otelconfig.GRPCSettings{},
					ThriftCompact: &otelconfig.UDPSettings{},
					ThriftHTTP:    &otelconfig.HttpSettings{},
				},
			},
		),
		otelconfig.WithReceiver(
			&otelconfig.OTLPReceiver{
				Protocols: otelconfig.Protocols{
					GRPC: &otelconfig.GRPCSettings{},
				},
			},
		),
	)
}

func otelModeFromStrategy(strategy jaegertracingv2.AgentStrategy) otelv1alpha1.Mode {
	if strategy == jaegertracingv2.AgentDaemonSet {
		return otelv1alpha1.ModeDaemonSet
	}
	return otelv1alpha1.ModeSidecar
}

func Get(jaeger jaegertracingv2.Jaeger, cfg config.Config) *otelv1alpha1.OpenTelemetryCollector {

	configuration := defaultConfig()
	jaegerExporter := configuration.GetJaegerExporter()
	if jaegerExporter != nil {
		jaegerExporter.Endpoint = fmt.Sprintf("%s.%s.svc:14250", naming.CollectorHeadlessService(jaeger), jaeger.Namespace)
	}

	configString, _ := configuration.String()

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
