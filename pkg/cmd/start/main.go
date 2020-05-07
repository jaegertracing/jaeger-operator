package start

import (
	"context"
	"fmt"

	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"sigs.k8s.io/controller-runtime/pkg/manager/signals"

	"github.com/jaegertracing/jaeger-operator/pkg/version"
)

// AddGeneratorFlags adds all command line flags related to manifest
// generation. They are shared between the operator and CLI `generate`
// command.
func AddGeneratorFlags(cmd *cobra.Command) {
	cmd.Flags().String("jaeger-version", version.DefaultJaeger(), "Deprecated: the Jaeger version is now managed entirely by the operator. This option is currently no-op.")

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

	docURL := fmt.Sprintf("https://www.jaegertracing.io/docs/%s", version.DefaultJaegerMajorMinor())
	cmd.Flags().String("documentation-url", docURL, "The URL for the 'Documentation' menu item")
	viper.BindPFlag("documentation-url", cmd.Flags().Lookup("documentation-url"))

	cmd.Flags().Bool("kafka-provisioning-minimal", false, "(unsupported) Whether to provision Kafka clusters with minimal requirements, suitable for demos and tests.")
	viper.BindPFlag("kafka-provisioning-minimal", cmd.Flags().Lookup("kafka-provisioning-minimal"))
}

// NewStartCommand starts the Jaeger Operator
func NewStartCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "start",
		Short: "Starts a new Jaeger Operator",
		Long:  "Starts a new Jaeger Operator",
		Run: func(cmd *cobra.Command, args []string) {
			start(cmd, args)
		},
	}

	AddGeneratorFlags(cmd)

	// Operator specific flags here. Any flags affecting manifest generation should be added to AddGeneratorFlags instead
	cmd.Flags().String("log-level", "info", "The log-level for the operator. Possible values: trace, debug, info, warning, error, fatal, panic")
	viper.BindPFlag("log-level", cmd.Flags().Lookup("log-level"))

	cmd.Flags().String("metrics-host", "0.0.0.0", "The host to bind the metrics port")
	viper.BindPFlag("metrics-host", cmd.Flags().Lookup("metrics-host"))

	cmd.Flags().Int32("metrics-port", 8383, "The metrics port")
	viper.BindPFlag("metrics-port", cmd.Flags().Lookup("metrics-port"))

	cmd.Flags().Int32("cr-metrics-port", 8686, "The metrics port for Operator and/or Custom Resource based metrics")
	viper.BindPFlag("cr-metrics-port", cmd.Flags().Lookup("cr-metrics-port"))

	cmd.Flags().String("jaeger-agent-hostport", "localhost:6831", "The location for the Jaeger Agent")
	viper.BindPFlag("jaeger-agent-hostport", cmd.Flags().Lookup("jaeger-agent-hostport"))

	cmd.Flags().Bool("tracing-enabled", false, "Whether the Operator should report its own spans to a Jaeger instance")
	viper.BindPFlag("tracing-enabled", cmd.Flags().Lookup("tracing-enabled"))

	return cmd
}

func start(cmd *cobra.Command, args []string) {
	mgr := bootstrap(context.Background())
	// Start the Cmd
	log.Fatal(mgr.Start(signals.SetupSignalHandler()))
}
