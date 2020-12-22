package instrument

const (
	// BootstrapTracer is the OpenTelemetry tracer name for the bootstrap procedure
	BootstrapTracer string = "operator/bootstrap"

	// ReconciliationTracer is the OpenTelemetry tracer name for the reconciliation loops
	ReconciliationTracer string = "operator/reconciliation"
)
