package generate

import (
	"context"
	"fmt"
	"os"

	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"

	v1 "github.com/jaegertracing/jaeger-operator/pkg/apis/jaegertracing/v1"
	"github.com/jaegertracing/jaeger-operator/pkg/strategy"
	"github.com/jaegertracing/jaeger-operator/pkg/version"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	k8s_json "k8s.io/apimachinery/pkg/runtime/serializer/json"
	"k8s.io/apimachinery/pkg/util/yaml"
)

// NewGenerateCommand starts the Jaeger Operator
func NewGenerateCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "generate",
		Short: "Generate YAML manifests from Jaeger CRD",
		Long: `Generate YAML manifests from Jaeger CRD.

Defaults to reading Jaeger CRD from standard input and writing it to standard output, override with --cr <filename> and --output <filename>.`,
		RunE: generate,
	}

	cmd.Flags().String("cr", "/dev/stdin", "Input Jaeger CRD")
	viper.BindPFlag("cr", cmd.Flags().Lookup("cr"))

	cmd.Flags().String("output", "/dev/stdout", "Where to print the generated YAML documents")
	viper.BindPFlag("output", cmd.Flags().Lookup("output"))

	// TODO: jaeger-version -- now deprecated/nop. How to handle that. OK if running from a container with the config file? TODO test that
	// TODO: Test with podman run

	// --- 8< ---
	// TODO: These options must be shared with start/bootstrap.go Where (which package) do we put them?

	// TODO: Did I get all the relevant options? Too many?

	cmd.Flags().String("jaeger-agent-image", "jaegertracing/jaeger-agent", "The Docker image for the Jaeger Agent")
	viper.BindPFlag("jaeger-agent-image", cmd.Flags().Lookup("jaeger-agent-image"))

	cmd.Flags().String("jaeger-query-image", "jaegertracing/jaeger-query", "The Docker image for the Jaeger Query")
	viper.BindPFlag("jaeger-query-image", cmd.Flags().Lookup("jaeger-query-image"))

	cmd.Flags().String("jaeger-collector-image", "jaegertracing/jaeger-collector", "The Docker image for the Jaeger Collector")
	viper.BindPFlag("jaeger-collector-image", cmd.Flags().Lookup("jaeger-collector-image"))

	cmd.Flags().String("jaeger-ingester-image", "jaegertracing/jaeger-ingester", "The Docker image for the Jaeger Ingester")
	viper.BindPFlag("jaeger-ingester-image", cmd.Flags().Lookup("jaeger-ingester-image"))

	cmd.Flags().String("jaeger-all-in-one-image", "jaegertracing/all-in-one", "The Docker image for the Jaeger all-in-one")
	viper.BindPFlag("jaeger-all-in-one-image", cmd.Flags().Lookup("jaeger-all-in-one-image"))

	cmd.Flags().String("jaeger-cassandra-schema-image", "jaegertracing/jaeger-cassandra-schema", "The Docker image for the Jaeger Cassandra Schema")
	viper.BindPFlag("jaeger-cassandra-schema-image", cmd.Flags().Lookup("jaeger-cassandra-schema-image"))

	cmd.Flags().String("jaeger-spark-dependencies-image", "jaegertracing/spark-dependencies", "The Docker image for the Spark Dependencies Job")
	viper.BindPFlag("jaeger-spark-dependencies-image", cmd.Flags().Lookup("jaeger-spark-dependencies-image"))

	cmd.Flags().String("jaeger-es-index-cleaner-image", "jaegertracing/jaeger-es-index-cleaner", "The Docker image for the Jaeger Elasticsearch Index Cleaner")
	viper.BindPFlag("jaeger-es-index-cleaner-image", cmd.Flags().Lookup("jaeger-es-index-cleaner-image"))

	cmd.Flags().String("jaeger-es-rollover-image", "jaegertracing/jaeger-es-rollover", "The Docker image for the Jaeger Elasticsearch Rollover")
	viper.BindPFlag("jaeger-es-rollover-image", cmd.Flags().Lookup("jaeger-es-rollover-image"))

	cmd.Flags().String("openshift-oauth-proxy-image", "openshift/oauth-proxy:latest", "The Docker image location definition for the OpenShift OAuth Proxy")
	viper.BindPFlag("openshift-oauth-proxy-image", cmd.Flags().Lookup("openshift-oauth-proxy-image"))

	cmd.Flags().String("openshift-oauth-proxy-imagestream-ns", "", "The namespace for the OpenShift OAuth Proxy imagestream")
	viper.BindPFlag("openshift-oauth-proxy-imagestream-ns", cmd.Flags().Lookup("openshift-oauth-proxy-imagestream-ns"))

	cmd.Flags().String("openshift-oauth-proxy-imagestream-name", "", "The name for the OpenShift OAuth Proxy imagestream")
	viper.BindPFlag("openshift-oauth-proxy-imagestream-name", cmd.Flags().Lookup("openshift-oauth-proxy-imagestream-name"))

	cmd.Flags().String("platform", "auto-detect", "The target platform the operator will run. Possible values: 'kubernetes', 'openshift', 'auto-detect'")
	viper.BindPFlag("platform", cmd.Flags().Lookup("platform"))

	cmd.Flags().String("es-provision", "auto", "Whether to auto-provision an Elasticsearch cluster for suitable Jaeger instances. Possible values: 'yes', 'no', 'auto'. When set to 'auto' and the API name 'logging.openshift.io' is available, auto-provisioning is enabled.")
	viper.BindPFlag("es-provision", cmd.Flags().Lookup("es-provision"))

	cmd.Flags().String("kafka-provision", "auto", "Whether to auto-provision a Kafka cluster for suitable Jaeger instances. Possible values: 'yes', 'no', 'auto'. When set to 'auto' and the API name 'kafka.strimzi.io' is available, auto-provisioning is enabled.")
	viper.BindPFlag("kafka-provision", cmd.Flags().Lookup("kafka-provision"))

	cmd.Flags().String("log-level", "info", "The log-level for the operator. Possible values: trace, debug, info, warning, error, fatal, panic")
	viper.BindPFlag("log-level", cmd.Flags().Lookup("log-level"))

	docURL := fmt.Sprintf("https://www.jaegertracing.io/docs/%s", version.DefaultJaegerMajorMinor())
	cmd.Flags().String("documentation-url", docURL, "The URL for the 'Documentation' menu item")
	viper.BindPFlag("documentation-url", cmd.Flags().Lookup("documentation-url"))

	cmd.Flags().String("jaeger-agent-hostport", "localhost:6831", "The location for the Jaeger Agent")
	viper.BindPFlag("jaeger-agent-hostport", cmd.Flags().Lookup("jaeger-agent-hostport"))

	cmd.Flags().Bool("kafka-provisioning-minimal", false, "(unsupported) Whether to provision Kafka clusters with minimal requirements, suitable for demos and tests.")
	viper.BindPFlag("kafka-provisioning-minimal", cmd.Flags().Lookup("kafka-provisioning-minimal"))

	// --- 8< ---

	return cmd
}

func createSpecFromYAML(filename string) (*v1.Jaeger, error) {
	// #nosec   G304: Potential file inclusion via variable
	f, err := os.Open(filename)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	var spec v1.Jaeger
	decoder := yaml.NewYAMLOrJSONDecoder(f, 8192)
	if err := decoder.Decode(&spec); err != nil {
		return nil, err
	}

	return &spec, nil
}

func generate(cmd *cobra.Command, args []string) error {
	level, err := log.ParseLevel(viper.GetString("log-level"))
	if err != nil {
		log.SetLevel(log.InfoLevel)
	} else {
		log.SetLevel(level)
	}

	input := viper.GetString("cr")
	if input == "/dev/stdin" {
		// Reading from stdin by default is neat when running as a
		// container instead of a binary, but is confusing when no
		// input is sent and the program just hangs
		log.Info("Reading Jaeger CRD from standard input (use --cr <filename> to override)")
	}

	spec, err := createSpecFromYAML(input)
	if err != nil {
		return err
	}

	s := strategy.For(context.Background(), spec)

	outputName := viper.GetString("output")
	out, err := os.OpenFile(outputName, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, 0600)
	if err != nil {
		return err
	}

	defer out.Close()

	encoder := k8s_json.NewYAMLSerializer(k8s_json.DefaultMetaFactory, nil, nil)
	for _, obj := range s.All() {
		// OwnerReferences normally references the CR, but it is not a
		// resource in the cluster so we must remove it

		type f interface {
			SetOwnerReferences(references []metav1.OwnerReference)
		}

		meta := obj.(f)
		meta.SetOwnerReferences(nil)

		fmt.Fprintln(out, "---")
		if err := encoder.Encode(obj, out); err != nil {
			log.Fatal(err)
		}
	}

	return nil
}
