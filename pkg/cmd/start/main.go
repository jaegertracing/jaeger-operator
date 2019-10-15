package start

import (
	"context"
	"fmt"
	"os"
	"runtime"

	"github.com/operator-framework/operator-sdk/pkg/k8sutil"
	kubemetrics "github.com/operator-framework/operator-sdk/pkg/kube-metrics"
	"github.com/operator-framework/operator-sdk/pkg/leader"
	"github.com/operator-framework/operator-sdk/pkg/metrics"
	"github.com/operator-framework/operator-sdk/pkg/restmapper"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
	"k8s.io/client-go/rest"
	"sigs.k8s.io/controller-runtime/pkg/client/config"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/manager/signals"

	"github.com/jaegertracing/jaeger-operator/pkg/apis"
	v1 "github.com/jaegertracing/jaeger-operator/pkg/apis/jaegertracing/v1"
	"github.com/jaegertracing/jaeger-operator/pkg/controller"
	"github.com/jaegertracing/jaeger-operator/pkg/upgrade"
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

	cmd.Flags().String("platform", "auto-detect", "The target platform the operator will run. Possible values: 'kubernetes', 'openshift', 'auto-detect'")
	viper.BindPFlag("platform", cmd.Flags().Lookup("platform"))

	cmd.Flags().String("es-provision", "auto", "Whether to auto-provision an Elasticsearch cluster for suitable Jaeger instances. Possible values: 'yes', 'no', 'auto'. When set to 'auto' and the API name 'logging.openshift.io' is available, auto-provisioning is enabled.")
	viper.BindPFlag("es-provision", cmd.Flags().Lookup("es-provision"))

	cmd.Flags().String("kafka-provision", "auto", "Whether to auto-provision a Kafka cluster for suitable Jaeger instances. Possible values: 'yes', 'no', 'auto'. When set to 'auto' and the API name 'kafka.strimzi.io' is available, auto-provisioning is enabled.")
	viper.BindPFlag("kafka-provision", cmd.Flags().Lookup("kafka-provision"))

	cmd.Flags().String("log-level", "info", "The log-level for the operator. Possible values: trace, debug, info, warning, error, fatal, panic")
	viper.BindPFlag("log-level", cmd.Flags().Lookup("log-level"))

	cmd.Flags().String("metrics-host", "0.0.0.0", "The host to bind the metrics port")
	viper.BindPFlag("metrics-host", cmd.Flags().Lookup("metrics-host"))

	cmd.Flags().Int32("metrics-port", 8383, "The metrics port")
	viper.BindPFlag("metrics-port", cmd.Flags().Lookup("metrics-port"))

	cmd.Flags().Int32("cr-metrics-port", 8686, "The metrics port for Operator and/or Custom Resource based metrics")
	viper.BindPFlag("cr-metrics-port", cmd.Flags().Lookup("cr-metrics-port"))

	docURL := fmt.Sprintf("https://www.jaegertracing.io/docs/%s", version.DefaultJaegerMajorMinor())
	cmd.Flags().String("documentation-url", docURL, "The URL for the 'Documentation' menu item")
	viper.BindPFlag("documentation-url", cmd.Flags().Lookup("documentation-url"))

	return cmd
}

func start(cmd *cobra.Command, args []string) {
	level, err := log.ParseLevel(viper.GetString("log-level"))
	if err != nil {
		log.SetLevel(log.InfoLevel)
	} else {
		log.SetLevel(level)
	}

	podNamespace, found := os.LookupEnv("POD_NAMESPACE")
	if !found {
		log.Warn("the POD_NAMESPACE env var isn't set, this operator's identity might clash with another operator's instance")
	}

	operatorName, found := os.LookupEnv("OPERATOR_NAME")
	if !found {
		log.Warn("the OPERATOR_NAME env var isn't set, this operator's identity might clash with another operator's instance")
	}

	identity := fmt.Sprintf("%s.%s", podNamespace, operatorName)
	viper.Set(v1.ConfigIdentity, identity)

	log.WithFields(log.Fields{
		"os":              runtime.GOOS,
		"arch":            runtime.GOARCH,
		"version":         runtime.Version(),
		"operator-sdk":    version.Get().OperatorSdk,
		"jaeger-operator": version.Get().Operator,
		"identity":        identity,
		"jaeger":          version.Get().Jaeger,
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
		MapperProvider:     restmapper.NewDynamicRESTMapper,
		MetricsBindAddress: fmt.Sprintf("%s:%d", viper.GetString("metrics-host"), viper.GetInt32("metrics-port")),
	})
	if err != nil {
		log.Fatal(err)
	}

	// Setup Scheme for all resources
	if err := apis.AddToScheme(mgr.GetScheme()); err != nil {
		log.Fatal(err)
	}

	// upgrades all the instances managed by this operator
	if err := upgrade.ManagedInstances(mgr.GetClient()); err != nil {
		log.WithError(err).Warn("failed to upgrade managed instances")
	}

	// Setup all Controllers
	if err := controller.AddToManager(mgr); err != nil {
		log.Fatal(err)
	}

	// Create Service object to expose the metrics port.
	if err := serveCRMetrics(cfg); err != nil {
		log.WithField("error", err.Error()).Warn("could not generate and serve custom resource metrics")
	}

	metricsPort := viper.GetInt32("metrics-port")
	operatorMetricsPort := viper.GetInt32("cr-metrics-port")

	// Add to the below struct any other metrics ports you want to expose.
	servicePorts := []corev1.ServicePort{
		{Port: metricsPort, Name: metrics.OperatorPortName, Protocol: corev1.ProtocolTCP, TargetPort: intstr.IntOrString{Type: intstr.Int, IntVal: metricsPort}},
		{Port: operatorMetricsPort, Name: metrics.CRPortName, Protocol: corev1.ProtocolTCP, TargetPort: intstr.IntOrString{Type: intstr.Int, IntVal: operatorMetricsPort}},
	}
	// Create Service object to expose the metrics port(s).
	service, err := metrics.CreateMetricsService(ctx, cfg, servicePorts)
	if err != nil {
		log.WithField("error", err.Error()).Warn("could not create metrics Service")
	}

	// CreateServiceMonitors will automatically create the prometheus-operator ServiceMonitor resources
	// necessary to configure Prometheus to scrape metrics from this operator.
	services := []*corev1.Service{service}
	_, err = metrics.CreateServiceMonitors(cfg, namespace, services)
	if err != nil {
		log.WithField("error", err.Error()).Warn("Could not create ServiceMonitor object")
		// If this operator is deployed to a cluster without the prometheus-operator running, it will return
		// ErrServiceMonitorNotPresent, which can be used to safely skip ServiceMonitor creation.
		if err == metrics.ErrServiceMonitorNotPresent {
			log.WithField("error", err.Error()).Warn("Install prometheus-operator in your cluster to create ServiceMonitor objects")
		}
	}

	// Start the Cmd
	log.Fatal(mgr.Start(signals.SetupSignalHandler()))
}

// serveCRMetrics gets the Operator/CustomResource GVKs and generates metrics based on those types.
// It serves those metrics on "http://metricsHost:operatorMetricsPort".
func serveCRMetrics(cfg *rest.Config) error {
	// Below function returns filtered operator/CustomResource specific GVKs.
	// For more control override the below GVK list with your own custom logic.
	filteredGVK, err := k8sutil.GetGVKsFromAddToScheme(apis.AddToScheme)
	if err != nil {
		return err
	}
	// Get the namespace the operator is currently deployed in.
	operatorNs, err := k8sutil.GetOperatorNamespace()
	if err != nil {
		return err
	}
	// To generate metrics in other namespaces, add the values below.
	ns := []string{operatorNs}
	// Generate and serve custom resource specific metrics.
	err = kubemetrics.GenerateAndServeCRMetrics(cfg, ns, filteredGVK, viper.GetString("metrics-host"), viper.GetInt32("cr-metrics-port"))
	if err != nil {
		return err
	}
	return nil
}
