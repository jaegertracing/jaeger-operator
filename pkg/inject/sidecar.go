package inject

var (
	// AnnotationRev is the annotation name to look for when deciding whether or not to inject
	AnnotationRev = "sidecar.jaegertracing.io/revision"
	// Annotation is the annotation name to look for when deciding whether or not to inject
	Annotation = "sidecar.jaegertracing.io/inject"
	// AnnotationLegacy holds the annotation name we had in the past, which we keep for backwards compatibility
	AnnotationLegacy = "inject-jaeger-agent"
	// PrometheusDefaultAnnotations is a map containing annotations for prometheus to be inserted at sidecar in case it doesn't have any
	PrometheusDefaultAnnotations = map[string]string{
		"prometheus.io/scrape": "true",
		"prometheus.io/port":   "14271",
	}
)
