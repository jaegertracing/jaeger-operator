module github.com/jaegertracing/jaeger-operator

go 1.16

require (
	github.com/Masterminds/semver v1.5.0
	github.com/googleapis/gnostic v0.5.5
	github.com/mitchellh/go-homedir v1.1.0
	github.com/openshift/api v0.0.0-20210713130143-be21c6cb1bea
	github.com/openshift/elasticsearch-operator v0.0.0-20210921091239-caf25067d56d
	github.com/opentracing/opentracing-go v1.1.0
	github.com/operator-framework/operator-lib v0.9.0
	github.com/pkg/errors v0.9.1
	github.com/prometheus/client_golang v1.11.0
	github.com/sirupsen/logrus v1.8.1
	github.com/spf13/cobra v1.3.0
	github.com/spf13/pflag v1.0.5
	github.com/spf13/viper v1.10.1
	github.com/stretchr/testify v1.7.0
	github.com/uber/jaeger-client-go v2.20.1+incompatible
	go.opentelemetry.io/otel v0.20.0
	go.opentelemetry.io/otel/exporters/metric/prometheus v0.20.0
	go.opentelemetry.io/otel/exporters/trace/jaeger v0.20.0
	go.opentelemetry.io/otel/metric v0.20.0
	go.opentelemetry.io/otel/oteltest v0.20.0
	go.opentelemetry.io/otel/sdk v0.20.0
	go.opentelemetry.io/otel/sdk/export/metric v0.20.0
	go.opentelemetry.io/otel/sdk/metric v0.20.0
	go.opentelemetry.io/otel/trace v0.20.0
	k8s.io/api v0.21.3
	k8s.io/apimachinery v0.21.3
	k8s.io/client-go v12.0.0+incompatible
	sigs.k8s.io/controller-runtime v0.9.6
)

replace k8s.io/client-go => k8s.io/client-go v0.21.2 // Required by prometheus-operator
