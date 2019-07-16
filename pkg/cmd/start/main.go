package start

import (
	"context"
	"fmt"
	"runtime"

	"github.com/operator-framework/operator-sdk/pkg/k8sutil"
	"github.com/operator-framework/operator-sdk/pkg/leader"
	"github.com/operator-framework/operator-sdk/pkg/metrics"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"sigs.k8s.io/controller-runtime/pkg/client/config"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/runtime/signals"

	"github.com/jaegertracing/jaeger-operator/pkg/apis"
	"github.com/jaegertracing/jaeger-operator/pkg/controller"
	"github.com/jaegertracing/jaeger-operator/pkg/version"
)

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

	cmd.Flags().String("jaeger-version", version.DefaultJaeger(), "The Jaeger version to use")
	viper.BindPFlag("jaeger-version", cmd.Flags().Lookup("jaeger-version"))

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

	cmd.Flags().String("platform", "auto-detect", "The target platform the operator will run. Possible values: 'kubernetes', 'openshift', 'auto-detect'")
	viper.BindPFlag("platform", cmd.Flags().Lookup("platform"))

	cmd.Flags().String("es-provision", "auto", "Whether to auto-provision an Elasticsearch cluster for suitable Jaeger instances. Possible values: 'yes', 'no', 'auto'. When set to 'auto' and the API name 'logging.openshift.io' is available, auto-provisioning is enabled.")
	viper.BindPFlag("es-provision", cmd.Flags().Lookup("es-provision"))

	cmd.Flags().String("log-level", "info", "The log-level for the operator. Possible values: trace, debug, info, warning, error, fatal, panic")
	viper.BindPFlag("log-level", cmd.Flags().Lookup("log-level"))

	cmd.Flags().String("metrics-host", "0.0.0.0", "The host to bind the metrics port")
	viper.BindPFlag("metrics-host", cmd.Flags().Lookup("metrics-host"))

	cmd.Flags().Int32("metrics-port", 8383, "The metrics port")
	viper.BindPFlag("metrics-port", cmd.Flags().Lookup("metrics-port"))

	return cmd
}

func start(cmd *cobra.Command, args []string) {
	level, err := log.ParseLevel(viper.GetString("log-level"))
	if err != nil {
		log.SetLevel(log.InfoLevel)
	} else {
		log.SetLevel(level)
	}

	log.WithFields(log.Fields{
		"os":              runtime.GOOS,
		"arch":            runtime.GOARCH,
		"version":         runtime.Version(),
		"operator-sdk":    version.Get().OperatorSdk,
		"jaeger-operator": version.Get().Operator,
	}).Info("Versions")

	ctx := context.Background()

	namespace, err := k8sutil.GetWatchNamespace()
	if err != nil {
		log.WithError(err).Fatal("failed to get watch namespace")
	}

	// Get a config to talk to the apiserver
	cfg, err := config.GetConfig()
	if err != nil {
		log.Fatal(err)
	}

	if err := leader.Become(ctx, "jaeger-operator-lock"); err != nil {
		log.Fatal(err)
	}

	// Create a new Cmd to provide shared dependencies and start components
	mgr, err := manager.New(cfg, manager.Options{
		Namespace:          namespace,
		MetricsBindAddress: fmt.Sprintf("%s:%d", viper.GetString("metrics-host"), viper.GetInt32("metrics-port")),
	})
	if err != nil {
		log.Fatal(err)
	}

	// Setup Scheme for all resources
	if err := apis.AddToScheme(mgr.GetScheme()); err != nil {
		log.Fatal(err)
	}

	// Setup all Controllers
	if err := controller.AddToManager(mgr); err != nil {
		log.Fatal(err)
	}

	// Create Service object to expose the metrics port.
	if _, err := metrics.ExposeMetricsPort(ctx, viper.GetInt32("metrics-port")); err != nil {
		log.Fatal(err)
	}

	// Start the Cmd
	log.Fatal(mgr.Start(signals.SetupSignalHandler()))
}
