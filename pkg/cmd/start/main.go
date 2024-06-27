package start

import (
	"context"
	"fmt"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/log"

	v1 "github.com/jaegertracing/jaeger-operator/apis/v1"
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
	cmd.Flags().String("jaeger-spark-dependencies-image", "ghcr.io/jaegertracing/spark-dependencies/spark-dependencies", "The Docker image for the Spark Dependencies Job")
	cmd.Flags().String("jaeger-es-index-cleaner-image", "jaegertracing/jaeger-es-index-cleaner", "The Docker image for the Jaeger Elasticsearch Index Cleaner")
	cmd.Flags().String("jaeger-es-rollover-image", "jaegertracing/jaeger-es-rollover", "The Docker image for the Jaeger Elasticsearch Rollover")
	cmd.Flags().String(v1.FlagOpenShiftOauthProxyImage, "quay.io/openshift/origin-oauth-proxy:4.12", "The Docker image location definition for the OpenShift OAuth Proxy")
	cmd.Flags().String("openshift-oauth-proxy-imagestream-ns", "", "The namespace for the OpenShift OAuth Proxy imagestream")
	cmd.Flags().String("openshift-oauth-proxy-imagestream-name", "", "The name for the OpenShift OAuth Proxy imagestream")
	cmd.Flags().String("platform", v1.FlagPlatformAutoDetect, "The target platform the operator will run. Possible values: 'kubernetes', 'openshift', 'auto-detect'")
	cmd.Flags().String("es-provision", v1.FlagProvisionElasticsearchAuto, "Whether to auto-provision an Elasticsearch cluster for suitable Jaeger instances. Possible values: 'yes', 'no', 'auto'. When set to 'auto' and the API name 'logging.openshift.io' is available, auto-provisioning is enabled.")
	cmd.Flags().String("kafka-provision", "auto", "Whether to auto-provision a Kafka cluster for suitable Jaeger instances. Possible values: 'yes', 'no', 'auto'. When set to 'auto' and the API name 'kafka.strimzi.io' is available, auto-provisioning is enabled.")
	cmd.Flags().Bool("kafka-provisioning-minimal", false, "(unsupported) Whether to provision Kafka clusters with minimal requirements, suitable for demos and tests.")
	cmd.Flags().String("secure-listen-address", "", "")
	cmd.Flags().String("health-probe-bind-address", ":8081", "The address the probe endpoint binds to.")
	cmd.Flags().Int("webhook-bind-port", 9443, "The address webhooks expose.")
	cmd.Flags().String("tls-min-version", "VersionTLS12", "Minimum TLS version supported. Value must match version names from https://golang.org/pkg/crypto/tls/#pkg-constants.")
	cmd.Flags().StringSlice("tls-cipher-suites", nil, "Comma-separated list of cipher suites for the server. Values are from tls package constants (https://golang.org/pkg/crypto/tls/#pkg-constants). If omitted, the default Go cipher suites will be used")
	cmd.Flags().Bool("leader-elect", false, "Enable leader election for controller manager. "+
		"Enabling this will ensure there is only one active controller manager.")

	_ = viper.BindEnv("jaeger-agent-image", "RELATED_IMAGE_JAEGER_AGENT")
	_ = viper.BindEnv("jaeger-query-image", "RELATED_IMAGE_JAEGER_QUERY")
	_ = viper.BindEnv("jaeger-collector-image", "RELATED_IMAGE_JAEGER_COLLECTOR")
	_ = viper.BindEnv("jaeger-ingester-image", "RELATED_IMAGE_JAEGER_INGESTER")
	_ = viper.BindEnv("jaeger-all-in-one-image", "RELATED_IMAGE_JAEGER_ALL_IN_ONE")
	_ = viper.BindEnv("jaeger-cassandra-schema-image", "RELATED_IMAGE_CASSANDRA_SCHEMA")
	_ = viper.BindEnv("jaeger-spark-dependencies-image", "RELATED_IMAGE_SPARK_DEPENDENCIES")
	_ = viper.BindEnv("jaeger-es-index-cleaner-image", "RELATED_IMAGE_JAEGER_ES_INDEX_CLEANER")
	_ = viper.BindEnv("jaeger-es-rollover-image", "RELATED_IMAGE_JAEGER_ES_ROLLOVER")
	_ = viper.BindEnv(v1.FlagOpenShiftOauthProxyImage, "RELATED_IMAGE_OPENSHIFT_OAUTH_PROXY")

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
	cmd.Flags().String("jaeger-agent-hostport", "localhost:6831", "The location for the Jaeger Agent")
	cmd.Flags().Bool("tracing-enabled", false, "Whether the Operator should report its own spans to a Jaeger instance")

	return cmd
}

func start(cmd *cobra.Command, args []string) error {
	mgr := bootstrap(context.Background())

	// Start the Cmd
	if err := mgr.Start(ctrl.SetupSignalHandler()); err != nil {
		log.Log.Error(err, "Manager exited non-zero")
		return err
	}

	return nil
}
