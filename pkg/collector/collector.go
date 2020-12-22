package collector

import (
	jaegertracingv2 "github.com/jaegertracing/jaeger-operator/apis/jaegertracing/v2"
	"github.com/jaegertracing/jaeger-operator/pkg/naming"
	otelv1alpha1 "github.com/open-telemetry/opentelemetry-operator/api/v1alpha1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func Get(jaeger jaegertracingv2.Jaeger) otelv1alpha1.OpenTelemetryCollector {

	otelCollector := otelv1alpha1.OpenTelemetryCollector{
		ObjectMeta: metav1.ObjectMeta{
			Name:      naming.Collector(jaeger),
			Namespace: jaeger.Namespace,
		},
	}
	collectorSpec := jaeger.Spec.Collector

	if collectorSpec.Image != "" {
		otelCollector.Spec.Image = collectorSpec.Image
	}

	if collectorSpec.Replicas != nil {
		otelCollector.Spec.Replicas = collectorSpec.Replicas
	}

	return otelCollector
}
