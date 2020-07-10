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

// AddFlags adds all command line flags related to manifest
// generation. They are shared between the operator and CLI `generate`
// command.
func AddFlags(cmd *cobra.Command) {
	cmd.Flags().String("log-level", "info", "The log-level for the operator. Possible values: trace, debug, info, warning, error, fatal, panic")
	cmd.Flags().String("jaeger-version", version.DefaultJaeger(), "Deprecated: the Jaeger version is now managed entirely by the operator. This option is currently no-op.")
	cmd.Flags().String("jaeger-agent-image", "jaegertracing/jaeger-agent", "The Docker image for the Jaeger Agent")
	cmd.Flags().String("jaeger-query-image", "jaegertracing/jaeger-query", "The Docker image for the Jaeger Query")
	cmd.Flags().String("jaeger-collector-image", "jaegertracing/jaeger-collector", "The Docker image for the Jaeger Collector")
	cmd.Flags().String("jaeger-ingester-image", "jaegertracing/jaeger-ingester", "The Docker image for the Jaeger Ingester")
	cmd.Flags().String("jaeger-all-in-one-image", "jaegertracing/all-in-one", "The Docker image for the Jaeger all-in-one")
	cmd.Flags().String("jaeger-cassandra-schema-image", "jaegertracing/jaeger-cassandra-schema", "The Docker image for the Jaeger Cassandra Schema")
	cmd.Flags().String("jaeger-spark-dependencies-image", "jaegertracing/spark-dependencies", "The Docker image for the Spark Dependencies Job")
	cmd.Flags().String("jaeger-es-index-cleaner-image", "jaegertracing/jaeger-es-index-cleaner", "The Docker image for the Jaeger Elasticsearch Index Cleaner")
	cmd.Flags().String("jaeger-es-rollover-image", "jaegertracing/jaeger-es-rollover", "The Docker image for the Jaeger Elasticsearch Rollover")
	cmd.Flags().String("openshift-oauth-proxy-image", "openshift/oauth-proxy:latest", "The Docker image location definition for the OpenShift OAuth Proxy")
	cmd.Flags().String("openshift-oauth-proxy-imagestream-ns", "", "The namespace for the OpenShift OAuth Proxy imagestream")
	cmd.Flags().String("openshift-oauth-proxy-imagestream-name", "", "The name for the OpenShift OAuth Proxy imagestream")
	cmd.Flags().String("platform", "auto-detect", "The target platform the operator will run. Possible values: 'kubernetes', 'openshift', 'auto-detect'")
	cmd.Flags().String("es-provision", "auto", "Whether to auto-provision an Elasticsearch cluster for suitable Jaeger instances. Possible values: 'yes', 'no', 'auto'. When set to 'auto' and the API name 'logging.openshift.io' is available, auto-provisioning is enabled.")
	cmd.Flags().String("kafka-provision", "auto", "Whether to auto-provision a Kafka cluster for suitable Jaeger instances. Possible values: 'yes', 'no', 'auto'. When set to 'auto' and the API name 'kafka.strimzi.io' is available, auto-provisioning is enabled.")
	cmd.Flags().Bool("kafka-provisioning-minimal", false, "(unsupported) Whether to provision Kafka clusters with minimal requirements, suitable for demos and tests.")

	docURL := fmt.Sprintf("https://www.jaegertracing.io/docs/%s", version.DefaultJaegerMajorMinor())
	cmd.Flags().String("documentation-url", docURL, "The URL for the 'Documentation' menu item")
}

// NewStartCommand starts the Jaeger Operator
func NewStartCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "start",
		Short: "Starts a new Jaeger Operator",
		Long:  "Starts a new Jaeger Operator",
		PreRun: func(cmd *cobra.Command, args []string) {
			viper.BindPFlags(cmd.Flags())
		},
		RunE: start,
	}

	AddFlags(cmd)
	cmd.Flags().String("metrics-host", "0.0.0.0", "The host to bind the metrics port")
	cmd.Flags().Int32("metrics-port", 8383, "The metrics port")
	cmd.Flags().Int32("cr-metrics-port", 8686, "The metrics port for Operator and/or Custom Resource based metrics")
	cmd.Flags().String("jaeger-agent-hostport", "localhost:6831", "The location for the Jaeger Agent")
	cmd.Flags().Bool("tracing-enabled", false, "Whether the Operator should report its own spans to a Jaeger instance")

	return cmd
}

func start(cmd *cobra.Command, args []string) error {
	mgr := bootstrap(context.Background())

	// Start the Cmd
	if err := mgr.Start(signals.SetupSignalHandler()); err != nil {
		log.Fatal(err, "Manager exited non-zero")
		return err
	}

	return nil
}
