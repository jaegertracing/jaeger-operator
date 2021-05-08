package tracing

import (
	"go.opentelemetry.io/otel"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
)

var (
	provider *sdktrace.TracerProvider
	closers  []func()
)

// Bootstrap prepares a new tracer to be used by the operator
func Bootstrap() {
	provider := sdktrace.NewTracerProvider(sdktrace.WithSampler(sdktrace.AlwaysSample()))
	otel.SetTracerProvider(provider)
}

// AddJaegerExporter includes the given exporter into the existing provider
/*func AddJaegerExporter(exporter *jaeger.Exporter) {
	closers = append(closers, func() {
		exporter.Flush()
	})

	ssp := sdktrace.NewSimpleSpanProcessor(exporter)
	provider.RegisterSpanProcessor(ssp)
}
*/

// Close runs the closer functions collected from all relevant exporters
func Close() {
	for _, c := range closers {
		c()
	}
}
