package metrics

import (
	"context"

	"go.opentelemetry.io/otel/codes"

	v1 "github.com/jaegertracing/jaeger-operator/pkg/apis/jaegertracing/v1"
	"github.com/jaegertracing/jaeger-operator/pkg/tracing"

	prometheusclient "github.com/prometheus/client_golang/prometheus"
	log "github.com/sirupsen/logrus"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/exporters/metric/prometheus"
	"go.opentelemetry.io/otel/metric/global"
	export "go.opentelemetry.io/otel/sdk/export/metric"
	"go.opentelemetry.io/otel/sdk/metric/aggregator/histogram"
	controller "go.opentelemetry.io/otel/sdk/metric/controller/basic"
	processor "go.opentelemetry.io/otel/sdk/metric/processor/basic"
	selector "go.opentelemetry.io/otel/sdk/metric/selector/simple"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/metrics"
)

const meterName = "jaegertracing.io/jaeger"

// Bootstrap prepares a new tracer to be used by the operator
func Bootstrap(ctx context.Context, namespace string, client client.Client) {
	tracer := otel.GetTracerProvider().Tracer(v1.CustomMetricsTracer)
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
	)
	exporter, err := prometheus.NewExporter(config, c)
	if err != nil {
		span.SetStatus(codes.Error, err.Error())
		log.WithError(err).Error("Failed to initialize metrics")

	}

	global.SetMeterProvider(exporter.MeterProvider())

	// Create metrics
	instancesObservedValue := newInstancesMetric(client)
	err = instancesObservedValue.Setup(ctx)
	if err != nil {
		span.SetStatus(codes.Error, err.Error())
		log.WithError(err).Error("Failed to initialize metrics")
	}
}
