package metrics

import (
	"context"

	prometheusclient "github.com/prometheus/client_golang/prometheus"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/exporters/metric/prometheus"
	"go.opentelemetry.io/otel/metric/global"
	export "go.opentelemetry.io/otel/sdk/export/metric"
	"go.opentelemetry.io/otel/sdk/metric/aggregator/histogram"
	controller "go.opentelemetry.io/otel/sdk/metric/controller/basic"
	processor "go.opentelemetry.io/otel/sdk/metric/processor/basic"
	selector "go.opentelemetry.io/otel/sdk/metric/selector/simple"
	"go.opentelemetry.io/otel/sdk/resource"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/metrics"

	"github.com/jaegertracing/jaeger-operator/pkg/apis/jaegertracing/v1"
	"github.com/jaegertracing/jaeger-operator/pkg/tracing"
)

const meterName = "jaegertracing.io/jaeger"

// Bootstrap configures the OpenTelemetry meter provider with the Prometheus exporter.
func Bootstrap(ctx context.Context, namespace string, client client.Client) error {
	tracer := otel.GetTracerProvider().Tracer(v1.BootstrapTracer)
	ctx, span := tracer.Start(ctx, "bootstrap")
	defer span.End()
	tracing.SetInstanceID(ctx, namespace)

	config := prometheus.Config{
		Registry: metrics.Registry.(*prometheusclient.Registry),
	}
	c := controller.New(
		processor.New(
			selector.NewWithHistogramDistribution(
				histogram.WithExplicitBoundaries(config.DefaultHistogramBoundaries),
			),
			export.CumulativeExportKindSelector(),
			processor.WithMemory(true),
		),
		controller.WithResource(resource.NewWithAttributes([]attribute.KeyValue{}...)),
	)
	exporter, err := prometheus.NewExporter(config, c)
	if err != nil {
		return tracing.HandleError(err, span)
	}

	global.SetMeterProvider(exporter.MeterProvider())

	// Create metrics
	instancesObservedValue := newInstancesMetric(client)
	err = instancesObservedValue.Setup(ctx)
	return tracing.HandleError(err, span)
}
