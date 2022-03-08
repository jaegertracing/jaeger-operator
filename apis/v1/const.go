package v1

const (
	// LabelOperatedBy is used as the key to the label indicating which operator is managing the instance
	LabelOperatedBy string = "jaegertracing.io/operated-by"

	// ConfigIdentity is the key to the configuration map related to the operator's identity
	ConfigIdentity string = "identity"

	// ConfigWatchNamespace is the key to the configuration map related to the namespace the operator should watch
	ConfigWatchNamespace string = "watch-namespace"

	// ConfigEnableNamespaceController is the key to the configuration map related to the boolean, determining whether the namespace controller is enabled
	ConfigEnableNamespaceController string = "enable-namespace-controller"

	// ConfigOperatorScope is the configuration key holding the scope of the operator
	ConfigOperatorScope string = "operator-scope"

	// WatchAllNamespaces is the value that the ConfigWatchNamespace holds to represent "all namespaces".
	WatchAllNamespaces string = ""

	// OperatorScopeCluster signals that the operator's instance is installed cluster-wide
	OperatorScopeCluster string = "cluster"

	// OperatorScopeNamespace signals that the operator's instance is working on a single namespace
	OperatorScopeNamespace string = "namespace"

	// BootstrapTracer is the OpenTelemetry tracer name for the bootstrap procedure
	BootstrapTracer string = "operator/bootstrap"

	// ReconciliationTracer is the OpenTelemetry tracer name for the reconciliation loops
	ReconciliationTracer string = "operator/reconciliation"
)
