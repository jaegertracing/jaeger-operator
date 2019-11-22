package v1

const (
	// LabelOperatedBy is used as the key to the label indicating which operator is managing the instance
	LabelOperatedBy string = "jaegertracing.io/operated-by"

	// ConfigIdentity is the key to the configuration map related to the operator's identity
	ConfigIdentity string = "identity"

	// BootstrapTracer is the OpenTelemetry tracer name for the bootstrap procedure
	BootstrapTracer string = "operator/bootstrap"

	// ReconciliationTracer is the OpenTelemetry tracer name for the reconciliation loops
	ReconciliationTracer string = "operator/reconciliation"
)
