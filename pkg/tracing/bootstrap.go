package tracing

import (
	"context"
	"strings"

	log "github.com/sirupsen/logrus"
	"github.com/spf13/viper"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/exporters/trace/jaeger"
	"go.opentelemetry.io/otel/sdk/resource"
	tracesdk "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/semconv"

	v1 "github.com/jaegertracing/jaeger-operator/pkg/apis/jaegertracing/v1"
	"github.com/jaegertracing/jaeger-operator/pkg/version"
)

var (
	processor tracesdk.SpanProcessor
)

// Bootstrap prepares a new tracer to be used by the operator
func Bootstrap(ctx context.Context, namespace string) {
	if viper.GetBool("tracing-enabled") {
		err := buildSpanProcessor()
		if err != nil {
			log.WithError(err).Warn("could not configure a Jaeger tracer for the operator")
		} else {
			buildJaegerExporter(ctx, namespace, "")
		}
	}
}

//SetInstanceID set the computed instance id on the tracing provider
func SetInstanceID(ctx context.Context, namespace string) {
	if viper.GetBool("tracing-enabled") {
		// Rebuild the provider with the same exporter
		buildJaegerExporter(ctx, namespace, viper.GetString(v1.ConfigIdentity))
	}
}

func buildSpanProcessor() error {
	agentHostPort := viper.GetString("jaeger-agent-hostport")
	hostPort := strings.Split(agentHostPort, ":")

	var endpoint jaeger.EndpointOption
	if len(hostPort) >= 2 {
		endpoint = jaeger.WithAgentEndpoint(
			jaeger.WithAgentHost(hostPort[0]),
			jaeger.WithAgentPort(hostPort[1]),
		)
	} else {
		endpoint = jaeger.WithAgentEndpoint(
			jaeger.WithAgentHost(hostPort[0]),
		)
	}

	jexporter, err := jaeger.NewRawExporter(endpoint)

	if err != nil {
		return err
	}
	processor = tracesdk.NewBatchSpanProcessor(jexporter)
	return nil
}

func buildJaegerExporter(ctx context.Context, namespace string, instanceID string) {
	tracer := otel.GetTracerProvider().Tracer(v1.BootstrapTracer)
	ctx, span := tracer.Start(ctx, "buildJaegerExporter")
	defer span.End()
	attr := []attribute.KeyValue{
		semconv.ServiceNameKey.String("jaeger-operator"),
		semconv.ServiceVersionKey.String(version.Get().Operator),
		semconv.ServiceNamespaceKey.String(namespace),
	}

	if instanceID != "" {
		attr = append(attr, semconv.ServiceInstanceIDKey.String(instanceID))
	}
	if processor != nil {
		traceProvider := tracesdk.NewTracerProvider(
			tracesdk.WithSpanProcessor(processor),
			tracesdk.WithResource(resource.NewWithAttributes(attr...)),
		)
		otel.SetTracerProvider(traceProvider)
	}
}
