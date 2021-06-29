package metrics

import (
	"context"

	prometheusclient "github.com/prometheus/client_golang/prometheus"
	"github.com/spf13/viper"
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
	"go.opentelemetry.io/otel/semconv"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/metrics"

	v1 "github.com/jaegertracing/jaeger-operator/pkg/apis/jaegertracing/v1"
	"github.com/jaegertracing/jaeger-operator/pkg/tracing"
	"github.com/jaegertracing/jaeger-operator/pkg/version"
)

const meterName = "jaegertracing.io/jaeger"

// Bootstrap configures the OpenTelemetry meter provides with the prometheus exporter
func Bootstrap(ctx context.Context, namespace string, client client.Client) error {
	tracer := otel.GetTracerProvider().Tracer(v1.CustomMetricsTracer)
	ctx, span := tracer.Start(ctx, "bootstrap")
	defer span.End()
	tracing.SetInstanceID(ctx, namespace)

	attr := []attribute.KeyValue{
		semconv.ServiceNameKey.String("jaeger-operator"),
		semconv.ServiceVersionKey.String(version.Get().Operator),
		semconv.ServiceNamespaceKey.String(namespace),
	}

	instanceID := viper.GetString(v1.ConfigIdentity)

	if instanceID != "" {
		attr = append(attr, semconv.ServiceInstanceIDKey.String(instanceID))
	}
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
		controller.WithResource(resource.NewWithAttributes(attr...)),
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
