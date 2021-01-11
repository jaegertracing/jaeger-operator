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

package collector

import (
	otelv1alpha1 "github.com/open-telemetry/opentelemetry-operator/api/v1alpha1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	jaegertracingv2 "github.com/jaegertracing/jaeger-operator/apis/jaegertracing/v2"
	"github.com/jaegertracing/jaeger-operator/pkg/naming"
	"github.com/jaegertracing/jaeger-operator/pkg/util"
)

// TODO: Better way of doing this..
func DefaultConfig() string {
	return `
    receivers:
      jaeger:
        protocols:
          grpc:
    processors:
      queued_retry:

    exporters:
      logging:

    service:
      pipelines:
        traces:
          receivers: [jaeger]
          processors: [queued_retry]
          exporters: [logging]`
}

func Get(jaeger jaegertracingv2.Jaeger) otelv1alpha1.OpenTelemetryCollector {

	config := jaeger.Spec.Collector.Config
	if config == "" {
		config = DefaultConfig()
	}

	collectorSpecs := jaeger.Spec.Collector
	commonSpecs := util.Merge(jaeger.Spec.JaegerCommonSpec, collectorSpecs.JaegerCommonSpec)

	return otelv1alpha1.OpenTelemetryCollector{
		ObjectMeta: metav1.ObjectMeta{
			Name:      jaeger.Name,
			Namespace: jaeger.Namespace,
		},
		Spec: otelv1alpha1.OpenTelemetryCollectorSpec{
			Image:          naming.Image(jaeger.Spec.Collector.Image, "jaeger-collector-image"),
			Config:         config,
			Replicas:       collectorSpecs.Replicas,
			ServiceAccount: commonSpecs.ServiceAccount,
			VolumeMounts:   commonSpecs.VolumeMounts,
			Volumes:        commonSpecs.Volumes,
		},
	}
}
