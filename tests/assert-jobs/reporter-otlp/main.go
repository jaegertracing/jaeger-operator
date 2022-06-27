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
	"go.opentelemetry.io/otel/exporters/otlp"
	"go.opentelemetry.io/otel/exporters/otlp/otlpgrpc"
	"go.opentelemetry.io/otel/exporters/otlp/otlphttp"
	"go.opentelemetry.io/otel/propagation"
	controller "go.opentelemetry.io/otel/sdk/metric/controller/basic"
	processor "go.opentelemetry.io/otel/sdk/metric/processor/basic"
	"go.opentelemetry.io/otel/sdk/metric/selector/simple"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/semconv"
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

// Get the endpoint where the collector is listening
func getCollector() string {
	reportingProtocol := viper.GetString(flagReportingProtocol)
	jaegerEndpoint := viper.GetString(otlpExporterEndpoint)

	var endpoint string
	switch reportingProtocol {
	case "grpc":
		endpoint = fmt.Sprintf("%s:4317", jaegerEndpoint)
	case "http":
		endpoint = fmt.Sprintf("%s:4318", jaegerEndpoint)
	default:
		logrus.Fatalln("Reporting protocol", reportingProtocol, "not recognized")
	}
	return endpoint
}

// Initializes an OTLP exporter and configure the traces provider
func initProvider(serviceName string) func() {
	logrus.Debugln("Initializing the OTLP exporter")
	ctx := context.Background()

	collector := getCollector()

	reportingProtocol := viper.GetString(flagReportingProtocol)

	logrus.Debugln("Using", reportingProtocol, "to report the traces")

	var driver otlp.ProtocolDriver

	switch reportingProtocol {
	case "grpc":
		driver = otlpgrpc.NewDriver(
			otlpgrpc.WithInsecure(),
			otlpgrpc.WithEndpoint(collector),
			otlpgrpc.WithDialOption(grpc.WithBlock()),
		)
	case "http":
		driver = otlphttp.NewDriver(
			otlphttp.WithInsecure(),
			otlphttp.WithEndpoint(collector),
		)
	default:
		logrus.Fatalln("Reporting protocol", reportingProtocol, "not recognized")
	}

	exp, err := otlp.NewExporter(ctx, driver)
	if err != nil {
		logrus.Fatalln("error creating the exporter", err)
	}

	res, err := resource.New(ctx,
		resource.WithAttributes(
			semconv.ServiceNameKey.String(serviceName),
		),
	)
	if err != nil {
		logrus.Fatalln("error creating the resource", err)
	}

	bsp := sdktrace.NewBatchSpanProcessor(exp)
	tracerProvider := sdktrace.NewTracerProvider(
		sdktrace.WithSampler(sdktrace.AlwaysSample()),
		sdktrace.WithResource(res),
		sdktrace.WithSpanProcessor(bsp),
	)

	cont := controller.New(
		processor.New(
			simple.NewWithExactDistribution(),
			exp,
		),
		controller.WithExporter(exp),
		controller.WithCollectPeriod(2*time.Second),
	)

	otel.SetTextMapPropagator(propagation.TraceContext{})
	otel.SetTracerProvider(tracerProvider)

	err = cont.Start(context.Background())
	if err != nil {
		logrus.Fatalln("error while starting the controller", err)
	}

	// This function should be called when the tracing features will not be
	// used anymore
	return func() {
		err1 := cont.Stop(context.Background())
		err2 := tracerProvider.Shutdown(ctx)

		// The errors are checked later to try to run all the "closing" tasks
		if err1 != nil {
			logrus.Fatalln("failed while stopping the controller", err1)
		}
		if err2 != nil {
			logrus.Fatalln("failed to shutting down the tracer provider", err2)
		}
	}
}

// Generate substans inside a span
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
// jaegerEnpoint: where Jaeger endpoint is located
// serviceName: service to use for the span
// operationName: operation described in the span
func generateTraces(jaegerEndpoint string, serviceName string, operationName string) {
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
	if viper.GetBool(flagVerbose) == true {
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
		err = utils.WaitUntilRestAPIAvailable(fmt.Sprintf("http://%s", getCollector()))
		if err != nil {
			logrus.Fatalln(err)
		}
	default:
		logrus.Fatalln("Protocol not recognized:", reportingProtocol)
	}

	generateTraces(
		viper.GetString(otlpExporterEndpoint),
		viper.GetString(flagJaegerServiceName),
		viper.GetString(flagJaegerOperationName),
	)
}
