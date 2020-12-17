module github.com/jaegertracing/jaeger-operator

go 1.14

require (
	github.com/Masterminds/semver v1.5.0
	github.com/ghodss/yaml v1.0.1-0.20190212211648-25d852aebe32
	github.com/go-openapi/spec v0.19.8
	github.com/googleapis/gnostic v0.5.3 // indirect
	github.com/kr/pretty v0.2.0 // indirect
	github.com/mitchellh/go-homedir v1.1.0
	github.com/openshift/api v0.0.0-20200701144905-de5b010b2b38
	github.com/opentracing/opentracing-go v1.1.1-0.20190913142402-a7454ce5950e
	github.com/operator-framework/operator-sdk v0.18.2
	github.com/pkg/errors v0.9.1
	github.com/sirupsen/logrus v1.5.0
	github.com/spf13/cobra v1.0.0
	github.com/spf13/viper v1.7.1
	github.com/stretchr/testify v1.5.1
	github.com/uber/jaeger-client-go v2.20.1+incompatible
	go.opentelemetry.io/otel v0.1.2
	go.opentelemetry.io/otel/exporter/trace/jaeger v0.1.2
	golang.org/x/crypto v0.0.0-20200709230013-948cd5f35899 // indirect
	google.golang.org/grpc v1.32.0
	gopkg.in/yaml.v2 v2.3.0
	k8s.io/api v0.18.6
	k8s.io/apimachinery v0.18.6
	k8s.io/client-go v12.0.0+incompatible
	k8s.io/kube-openapi v0.0.0-20200410145947-61e04a5be9a6
	sigs.k8s.io/controller-runtime v0.6.3
)

replace (
	github.com/Azure/go-autorest => github.com/Azure/go-autorest v13.3.2+incompatible // Required by OLM
	k8s.io/client-go => k8s.io/client-go v0.18.2 // Required by prometheus-operator
	vbom.ml/util => github.com/fvbommel/util v0.0.0-20180919145318-efcd4e0f9787
)
