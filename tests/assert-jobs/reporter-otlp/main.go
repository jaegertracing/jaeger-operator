package main

import (
	"context"
	"flag"
	"fmt"
	"strings"
	"time"

	"github.com/sirupsen/logrus"
	"google.golang.org/grpc"

	"github.com/jaegertracing/jaeger-operator/tests/assert-jobs/utils"

	"github.com/spf13/pflag"
	"github.com/spf13/viper"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracehttp"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.12.0"
)

const (
	tracerName              = "E2E-TRACER"
	flagJaegerServiceName   = "jaeger-service-name"
	flagJaegerOperationName = "operation-name"
	flagVerbose             = "verbose"
	flagReportingProtocol   = "reporting-protocol"
	otlpExporterEndpoint    = "OTEL_EXPORTER_OTLP_ENDPOINT"
)

// Init the CMD and return error if something didn't go properly
func initCmd() error {
	viper.SetEnvKeyReplacer(strings.NewReplacer("-", "_"))
	viper.AutomaticEnv()

	viper.SetDefault(otlpExporterEndpoint, "localhost")
	flag.String(otlpExporterEndpoint, "", "OTL Exporter endpoint")

	viper.SetDefault(flagJaegerServiceName, "jaeger-service")
	flag.String(flagJaegerServiceName, "", "Jaeger service name")

	viper.SetDefault(flagReportingProtocol, "http")
	flag.String(flagReportingProtocol, "", "Protocol to report traces (http|grpc)")

	viper.SetDefault(flagVerbose, false)
	flag.Bool(flagVerbose, false, "Enable verbosity")

	viper.SetDefault(flagJaegerOperationName, "jaeger-operation")
	flag.String(flagJaegerOperationName, "", "Jaeger operation name")

	pflag.CommandLine.AddGoFlagSet(flag.CommandLine)
	pflag.Parse()

	err := viper.BindPFlags(pflag.CommandLine)
	return err
}

// Initializes an OTLP exporter and configure the traces provider
func initProvider(serviceName string) func() {
	logrus.Debugln("Initializing the OTLP exporter")
	ctx := context.Background()

	reportingProtocol := viper.GetString(flagReportingProtocol)

	logrus.Debugln("Using", reportingProtocol, "to report the traces")

	var exp *otlptrace.Exporter
	var err error
	switch reportingProtocol {
	case "grpc":
		exp, err = otlptracegrpc.New(ctx, otlptracegrpc.WithDialOption(grpc.WithBlock()))
	case "http":
		exp, err = otlptracehttp.New(ctx)
	default:
		logrus.Fatalln("Reporting protocol", reportingProtocol, "not recognized")
	}

	if err != nil {
		logrus.Fatalln("error creating", reportingProtocol, "exporter", err)
	}

	tracerProvider := sdktrace.NewTracerProvider(
		sdktrace.WithSampler(sdktrace.AlwaysSample()),
		sdktrace.WithResource(resource.NewWithAttributes(semconv.SchemaURL, semconv.ServiceNameKey.String(serviceName))),
		sdktrace.WithSpanProcessor(sdktrace.NewBatchSpanProcessor(exp)),
	)

	otel.SetTextMapPropagator(propagation.TraceContext{})
	otel.SetTracerProvider(tracerProvider)

	// This function should be called when the tracing features will not be
	// used anymore
	return func() {
		if err := tracerProvider.Shutdown(ctx); err != nil {
			logrus.Fatalln("failed to shutting down the tracer provider", err)
		}
	}
}

// Generate subspans inside a span
// ctx: context for the program
// depth: how many spans should be created as child spans of this one
func generateSubSpans(ctx context.Context, depth int) {
	if depth == 0 {
		return
	}
	tracer := otel.Tracer(tracerName)
	ctx, span := tracer.Start(ctx, fmt.Sprintf("subspan-%d", depth))
	defer span.End()
	logrus.Debugln("\tGenerating subspan", depth)
	time.Sleep(time.Millisecond * 30)
	generateSubSpans(ctx, depth-1)
}

// Generate some traces
// serviceName: service to use for the span
// operationName: operation described in the span
func generateTraces(serviceName string, operationName string) {
	logrus.Debugln("Trying to generate traces")
	shutdown := initProvider(serviceName)
	defer shutdown()

	tracer := otel.Tracer(tracerName)

	logrus.Debugln("Generating traces!")

	i := 0
	for {
		logrus.Debugf("Generating trace %d", i)
		ctx, iSpan := tracer.Start(context.Background(), operationName)
		generateSubSpans(ctx, 5)
		iSpan.End()
		i++
		time.Sleep(time.Millisecond * 100)
	}
}

func main() {
	err := initCmd()
	if err != nil {
		logrus.Fatal(err)
	}
	if viper.GetBool(flagVerbose) {
		logrus.SetLevel(logrus.DebugLevel)
	}

	// Sometimes, Kubernetes reports the Jaeger service is there but there is
	// an interval where the service is up but the REST API is not operative yet
	reportingProtocol := viper.GetString(flagReportingProtocol)

	switch reportingProtocol {
	case "grpc":
		// To avoid creating all the files for gRPC, we just wait some time
		time.Sleep(time.Second * 5)
	case "http":
		jaegerEndpoint := viper.GetString(otlpExporterEndpoint)
		if err := utils.WaitUntilRestAPIAvailable(jaegerEndpoint); err != nil {
			logrus.Fatalln(err)
		}
	default:
		logrus.Fatalln("Protocol not recognized:", reportingProtocol)
	}

	generateTraces(
		viper.GetString(flagJaegerServiceName),
		viper.GetString(flagJaegerOperationName),
	)
}
