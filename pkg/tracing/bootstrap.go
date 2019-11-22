package tracing

import (
	"log"

	"go.opentelemetry.io/otel/exporter/trace/jaeger"
	"go.opentelemetry.io/otel/global"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
)

var (
	provider *sdktrace.Provider
	closers  []func()
)

// Bootstrap prepares a new tracer to be used by the operator
func Bootstrap() {
	sampling := sdktrace.Config{DefaultSampler: sdktrace.AlwaysSample()}

	var err error
	provider, err = sdktrace.NewProvider(sdktrace.WithConfig(sampling))
	if err != nil {
		log.Fatal(err)
	}

	global.SetTraceProvider(provider)
}

// AddJaegerExporter includes the given exporter into the existing provider
func AddJaegerExporter(exporter *jaeger.Exporter) {
	closers = append(closers, func() {
		exporter.Flush()
	})

	ssp := sdktrace.NewSimpleSpanProcessor(exporter)
	provider.RegisterSpanProcessor(ssp)
}

// Close runs the closer functions collected from all relevant exporters
func Close() {
	for _, c := range closers {
		c()
	}
}
