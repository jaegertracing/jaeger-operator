package account

// Component represents type of jaeger component
type Component string

const (
	// CollectorComponent represents the value for the Component type for Jaeger Collector
	CollectorComponent Component = "collector"

	// QueryComponent represents the value for the Component type for Jaeger Query
	QueryComponent Component = "query"

	// IngesterComponent represents the value for the Component type for Jaeger Ingester
	IngesterComponent Component = "ingester"

	// AllInOneComponent represents the value for the Component type for Jaeger All-In-One
	AllInOneComponent Component = "all-in-one"

	// AgentComponent epresents the value for the Component type for Jaeger Agent
	AgentComponent Component = "agent"
)
