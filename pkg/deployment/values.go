package deployment

// Package level constants
const (

	// images
	allInOneValue = "jaegertracing/all-in-one"
	collector     = "jaeger-collector-image"
	// jaeger
	jaeger          = "jaeger"
	jaegerComponent = "jaeger-component"
	// version
	versionLabel = "jaeger-version"
	versionValue = "1.6"
	// prometheus
	prometheusScrapeLabel = "prometheus.io/scrape"
	prometheusScrapeValue = "true"
	prometheusPortLabel   = "prometheus.io/port"
	// meta
	metaAPIVersion = "apps/v1"
	metaDeployment = "Deployment"
	// containers
	zkCompactTrft   = "zk-compact-trft"
	configRest      = "config-rest"
	jgCompactTrft   = "jg-compact-trft"
	jgBinaryTrft    = "jg-binary-trft"
	spanStorageType = "SPAN_STORAGE_TYPE"
	zipkin          = "zipkin"
	cTchanTrft      = "c-tchan-trft"
	cBinaryTrft     = "c-binary-trft"
	query           = "query"
	// tests
	ingressEnabledDefault = "IngressEnabledDefault"
	ingressEnabledFalse   = "IngressEnabledFalse"
	ingressEnabledTrue    = "IngressEnabledTrue"
	// misc
	app = "app"
)
