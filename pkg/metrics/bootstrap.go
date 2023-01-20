package metrics

import (
	"context"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/exporters/prometheus"
	"go.opentelemetry.io/otel/metric/global"
	"go.opentelemetry.io/otel/sdk/metric"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/metrics"

	v1 "github.com/jaegertracing/jaeger-operator/apis/v1"
	"github.com/jaegertracing/jaeger-operator/pkg/tracing"
)

const meterName = "jaegertracing.io/jaeger"

// Bootstrap configures the OpenTelemetry meter provider with the Prometheus exporter.
func Bootstrap(ctx context.Context, namespace string, client client.Client) error {
	tracer := otel.GetTracerProvider().Tracer(v1.BootstrapTracer)
	ctx, span := tracer.Start(ctx, "bootstrap")
	defer span.End()
	tracing.SetInstanceID(ctx, namespace)

	exporter, err := prometheus.New(prometheus.WithRegisterer(metrics.Registry))
	if err != nil {
		return tracing.HandleError(err, span)
	}

	provider := metric.NewMeterProvider(metric.WithReader(exporter))

	global.SetMeterProvider(provider)

	// Create metrics
	instancesObservedValue := newInstancesMetric(client)
	err = instancesObservedValue.Setup(ctx)
	return tracing.HandleError(err, span)
}
