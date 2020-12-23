package collector

import (
	jaegertracingv2 "github.com/jaegertracing/jaeger-operator/apis/jaegertracing/v2"
	"github.com/jaegertracing/jaeger-operator/pkg/naming"
	otelv1alpha1 "github.com/open-telemetry/opentelemetry-operator/api/v1alpha1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// TODO: Better way of doing this..
func defaultConfig() string {
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
		config = defaultConfig()
	}

	return otelv1alpha1.OpenTelemetryCollector{
		ObjectMeta: metav1.ObjectMeta{
			Name:      naming.Collector(jaeger),
			Namespace: jaeger.Namespace,
		},
		Spec: otelv1alpha1.OpenTelemetryCollectorSpec{
			Image:  naming.Image(jaeger.Spec.Collector.Image, "jaeger-collector-image"),
			Config: config,
		},
	}
}
