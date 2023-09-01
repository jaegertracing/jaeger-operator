package start

import (
	"context"
	"crypto/tls"
	"flag"
	"fmt"
	"os"
	"runtime"
	"strings"
	"time"

	osimagev1 "github.com/openshift/api/image/v1"
	"github.com/spf13/pflag"
	"github.com/spf13/viper"
	"go.opentelemetry.io/otel"
	otelattribute "go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.uber.org/zap/zapcore"
	corev1 "k8s.io/api/core/v1"
	k8sruntime "k8s.io/apimachinery/pkg/runtime"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/rest"
	"sigs.k8s.io/controller-runtime/pkg/log"
	metricsserver "sigs.k8s.io/controller-runtime/pkg/metrics/server"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"

	//  import OIDC cluster authentication plugin, e.g. for IBM Cloud
	_ "k8s.io/client-go/plugin/pkg/client/auth/oidc"
	k8sapiflag "k8s.io/component-base/cli/flag"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/cache"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/healthz"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/webhook"

	jaegertracingv1 "github.com/jaegertracing/jaeger-operator/apis/v1"
	v1 "github.com/jaegertracing/jaeger-operator/apis/v1"
	appsv1controllers "github.com/jaegertracing/jaeger-operator/controllers/appsv1"
	esv1controllers "github.com/jaegertracing/jaeger-operator/controllers/elasticsearch"
	jaegertracingcontrollers "github.com/jaegertracing/jaeger-operator/controllers/jaegertracing"
	"github.com/jaegertracing/jaeger-operator/pkg/autoclean"
	"github.com/jaegertracing/jaeger-operator/pkg/autodetect"
	kafkav1beta2 "github.com/jaegertracing/jaeger-operator/pkg/kafka/v1beta2"
	opmetrics "github.com/jaegertracing/jaeger-operator/pkg/metrics"
	"github.com/jaegertracing/jaeger-operator/pkg/tracing"
	"github.com/jaegertracing/jaeger-operator/pkg/upgrade"
	"github.com/jaegertracing/jaeger-operator/pkg/util"
	"github.com/jaegertracing/jaeger-operator/pkg/version"

	consolev1 "github.com/openshift/api/console/v1"
	routev1 "github.com/openshift/api/route/v1"
	esv1 "github.com/openshift/elasticsearch-operator/apis/logging/v1"
)

var (
	scheme   = k8sruntime.NewScheme()
	setupLog = ctrl.Log.WithName("setup")
)

type tlsConfig struct {
	minVersion   string
	cipherSuites []string
}

func init() {
	utilruntime.Must(clientgoscheme.AddToScheme(scheme))
	utilruntime.Must(jaegertracingv1.AddToScheme(scheme))
	utilruntime.Must(kafkav1beta2.AddToScheme(scheme))
	utilruntime.Must(routev1.Install(scheme))
	utilruntime.Must(osimagev1.Install(scheme))
	utilruntime.Must(consolev1.Install(scheme))
	utilruntime.Must(esv1.AddToScheme(scheme))
	//+kubebuilder:scaffold:scheme
}

func bootstrap(ctx context.Context) manager.Manager {
	namespace := getNamespace(ctx)
	tracing.Bootstrap(ctx, namespace)

	tracer := otel.GetTracerProvider().Tracer(v1.BootstrapTracer)
	ctx, span := tracer.Start(ctx, "bootstrap")
	defer span.End()

	setLogLevel(ctx)

	buildIdentity(ctx, namespace)
	tracing.SetInstanceID(ctx, namespace)

	ctrl.Log.Info("Versions",
		"os", runtime.GOOS,
		"arch", runtime.GOARCH,
		"version", runtime.Version(),
		"jaeger-operator", version.Get().Operator,
		"identity", viper.GetString(v1.ConfigIdentity),
		"jaeger", version.Get().Jaeger,
	)

	cfg, err := ctrl.GetConfig()
	if err != nil {
		span.SetStatus(codes.Error, err.Error())
		log.Log.V(6).Info("%s", err)
	}

	span.SetAttributes(otelattribute.String("Platform", autodetect.OperatorConfiguration.GetPlatform().String()))
	watchNamespace, found := os.LookupEnv("WATCH_NAMESPACE")
	if found {
		setupLog.Info("watching namespace(s)", "namespaces", watchNamespace)
	} else {
		setupLog.Info("the env var WATCH_NAMESPACE isn't set, watching all namespaces")
	}

	setOperatorScope(ctx, watchNamespace)

	mgr := createManager(ctx, cfg)

	if d, err := autodetect.New(mgr); err != nil {
		log.Log.Error(
			err,
			"failed to start the background process to auto-detect the operator capabilities",
		)
	} else {
		d.Start()
	}

	if c, err := autoclean.New(mgr); err != nil {
		log.Log.Error(
			err,
			"failed to start the background process to auto-clean the operator objects",
		)
	} else {
		c.Start()
	}

	detectNamespacePermissions(ctx, mgr)
	performUpgrades(ctx, mgr)
	setupControllers(ctx, mgr)
	setupWebhooks(ctx, mgr)
	err = opmetrics.Bootstrap(ctx, namespace, mgr.GetClient())
	if err != nil {
		log.Log.Error(err, "failed to initialize metrics")
	}
	return mgr
}

func detectNamespacePermissions(ctx context.Context, mgr manager.Manager) {
	tracer := otel.GetTracerProvider().Tracer(v1.BootstrapTracer)
	ctx, span := tracer.Start(ctx, "detectNamespacePermissions")
	defer span.End()

	namespaces := &corev1.NamespaceList{}
	opts := []client.ListOption{}
	if err := mgr.GetAPIReader().List(ctx, namespaces, opts...); err != nil {
		log.Log.V(-1).Info(
			fmt.Sprintf(
				"could not get a list of namespaces, disabling namespace controller. reason: %s",
				err,
			),
		)
		tracing.HandleError(err, span)
		span.SetAttributes(otelattribute.Bool(v1.ConfigEnableNamespaceController, false))
		viper.Set(v1.ConfigEnableNamespaceController, false)
	} else {
		span.SetAttributes(otelattribute.Bool(v1.ConfigEnableNamespaceController, true))
		viper.Set(v1.ConfigEnableNamespaceController, true)
	}
}

func setOperatorScope(ctx context.Context, namespace string) {
	tracer := otel.GetTracerProvider().Tracer(v1.BootstrapTracer)
	ctx, span := tracer.Start(ctx, "setOperatorScope") // nolint:ineffassign,staticcheck
	defer span.End()

	// set what's the namespace to watch
	viper.Set(v1.ConfigWatchNamespace, namespace)

	// for now, the logic is simple: if we are watching all namespaces, then we are cluster-wide
	if viper.GetString(v1.ConfigWatchNamespace) == v1.WatchAllNamespaces {
		span.SetAttributes(otelattribute.String(v1.ConfigOperatorScope, v1.OperatorScopeCluster))
		viper.Set(v1.ConfigOperatorScope, v1.OperatorScopeCluster)
	} else {
		log.Log.Info("Consider running the operator in a cluster-wide scope for extra features")
		span.SetAttributes(otelattribute.String(v1.ConfigOperatorScope, v1.OperatorScopeNamespace))
		viper.Set(v1.ConfigOperatorScope, v1.OperatorScopeNamespace)
	}
}

func setLogLevel(ctx context.Context) {
	tracer := otel.GetTracerProvider().Tracer(v1.BootstrapTracer)
	ctx, span := tracer.Start(ctx, "setLogLevel") // nolint:ineffassign,staticcheck
	defer span.End()

	var loggingLevel zapcore.Level
	switch strings.ToLower(viper.GetString("log-level")) {
	case "panic":
		loggingLevel = zapcore.PanicLevel
	case "fatal":
		loggingLevel = zapcore.FatalLevel
	case "error":
		loggingLevel = zapcore.ErrorLevel
	case "warn", "warning":
		loggingLevel = zapcore.WarnLevel
	case "info":
		loggingLevel = zapcore.InfoLevel
	case "debug":
		loggingLevel = zapcore.DebugLevel
	}

	opts := zap.Options{
		Development: true,
		Level:       loggingLevel,
	}

	opts.BindFlags(flag.CommandLine)
	pflag.CommandLine.AddGoFlagSet(flag.CommandLine)

	ctrl.SetLogger(zap.New(zap.UseFlagOptions(&opts)))
}

func buildIdentity(ctx context.Context, podNamespace string) {
	tracer := otel.GetTracerProvider().Tracer(v1.BootstrapTracer)
	ctx, span := tracer.Start(ctx, "buildIdentity") // nolint:ineffassign,staticcheck
	defer span.End()

	operatorName, found := os.LookupEnv("OPERATOR_NAME")
	if !found {
		log.Log.V(1).Info(
			"the OPERATOR_NAME env var isn't set, this operator's identity might clash with another operator's instance",
		)
		operatorName = "jaeger-operator"
	}

	var identity string
	if len(podNamespace) > 0 {
		identity = fmt.Sprintf("%s.%s", podNamespace, operatorName)
	} else {
		identity = operatorName
	}

	span.SetAttributes(otelattribute.String(v1.ConfigIdentity, identity))
	viper.Set(v1.ConfigIdentity, identity)
}

func createManager(ctx context.Context, cfg *rest.Config) manager.Manager {
	tracer := otel.GetTracerProvider().Tracer(v1.BootstrapTracer)
	ctx, span := tracer.Start(ctx, "createManager") // nolint:ineffassign,staticcheck
	defer span.End()

	metricsHost := viper.GetString("metrics-host")
	metricsPort := viper.GetInt("metrics-port")
	metricsAddr := fmt.Sprintf("%s:%d", metricsHost, metricsPort)
	enableLeaderElection := viper.GetBool("leader-elect")
	probeAddr := viper.GetString("health-probe-bind-address")
	webhookPort := viper.GetInt("webhook-bind-port")

	var tlsOpt tlsConfig
	tlsOpt.minVersion = viper.GetString("tls-min-version")
	tlsOpt.cipherSuites = viper.GetStringSlice("tls-cipher-suites")

	// see https://github.com/openshift/library-go/blob/4362aa519714a4b62b00ab8318197ba2bba51cb7/pkg/config/leaderelection/leaderelection.go#L104
	leaseDuration := time.Second * 137
	renewDeadline := time.Second * 107
	retryPeriod := time.Second * 26

	optionsTlSOptsFuncs := []func(*tls.Config){
		func(config *tls.Config) { tlsConfigSetting(config, tlsOpt) },
	}

	// Add support for MultiNamespace set in WATCH_NAMESPACE (e.g ns1,ns2)
	// Note that this is not intended to be used for excluding namespaces, this is better done via a Predicate
	// Also note that you may face performance issues when using this with a high number of namespaces.
	// More Info: https://godoc.org/github.com/kubernetes-sigs/controller-runtime/pkg/cache#MultiNamespacedCacheBuilder
	namespace := viper.GetString(v1.ConfigWatchNamespace)
	var namespaces map[string]cache.Config
	if namespace != "" {
		namespaces = map[string]cache.Config{}
		for _, ns := range strings.Split(namespace, ",") {
			namespaces[ns] = cache.Config{}
		}
	}

	options := ctrl.Options{
		Scheme: scheme,
		Metrics: metricsserver.Options{
			BindAddress: metricsAddr,
		},
		WebhookServer: webhook.NewServer(webhook.Options{
			Port:    webhookPort,
			TLSOpts: optionsTlSOptsFuncs,
		}),
		HealthProbeBindAddress: probeAddr,
		LeaderElection:         enableLeaderElection,
		LeaderElectionID:       "31e04290.jaegertracing.io",
		LeaseDuration:          &leaseDuration,
		RenewDeadline:          &renewDeadline,
		RetryPeriod:            &retryPeriod,
		Cache: cache.Options{
			DefaultNamespaces: namespaces,
		},
	}

	// Create a new manager to provide shared dependencies and start components
	mgr, err := ctrl.NewManager(cfg, options)
	if err != nil {
		span.SetStatus(codes.Error, err.Error())
		log.Log.V(6).Info(fmt.Sprintf("%s", err))
	}

	if err := mgr.AddHealthzCheck("healthz", healthz.Ping); err != nil {
		span.SetStatus(codes.Error, err.Error())
		setupLog.Error(err, "unable to set up health check")
		os.Exit(1)
	}
	if err := mgr.AddReadyzCheck("readyz", healthz.Ping); err != nil {
		span.SetStatus(codes.Error, err.Error())
		setupLog.Error(err, "unable to set up ready check")
		os.Exit(1)
	}

	return mgr
}

func performUpgrades(ctx context.Context, mgr manager.Manager) {
	tracer := otel.GetTracerProvider().Tracer(v1.BootstrapTracer)
	ctx, span := tracer.Start(ctx, "performUpgrades")
	defer span.End()

	// upgrades all the instances managed by this operator
	if err := upgrade.ManagedInstances(ctx, mgr.GetClient(), mgr.GetAPIReader(), version.Get().Jaeger); err != nil {
		log.Log.Error(err, "failed to upgrade managed instances")
	}
}

func setupControllers(ctx context.Context, mgr manager.Manager) {
	tracer := otel.GetTracerProvider().Tracer(v1.BootstrapTracer)
	ctx, span := tracer.Start(ctx, "setupControllers") // nolint:ineffassign,staticcheck
	clientReader := mgr.GetAPIReader()
	client := mgr.GetClient()
	schema := mgr.GetScheme()
	defer span.End()

	if err := jaegertracingcontrollers.NewReconciler(client, clientReader, schema).SetupWithManager(mgr); err != nil {
		setupLog.Error(err, "unable to create controller", "controller", "Jaeger")
		os.Exit(1)
	}

	if viper.GetBool(v1.ConfigEnableNamespaceController) {
		if err := appsv1controllers.NewNamespaceReconciler(client, clientReader, schema).SetupWithManager(mgr); err != nil {
			setupLog.Error(err, "unable to create controller", "controller", "Namespace")
			os.Exit(1)
		}
	} else {
		log.Log.V(1).Info("skipping reconciliation for namespaces, do not have permissions to list and watch namespaces")
	}

	if err := esv1controllers.NewReconciler(client, clientReader).SetupWithManager(mgr); err != nil {
		setupLog.Error(err, "unable to create controller", "controller", "Elasticsearch")
		os.Exit(1)
	}
}

func setupWebhooks(_ context.Context, mgr manager.Manager) {
	if err := (&v1.Jaeger{}).SetupWebhookWithManager(mgr); err != nil {
		setupLog.Error(err, "unable to create webhook", "webhook", "Jaeger")
		os.Exit(1)
	}

	// register webhook
	srv := mgr.GetWebhookServer()
	decoder := admission.NewDecoder(mgr.GetScheme())
	srv.Register("/mutate-v1-deployment", &webhook.Admission{
		Handler: appsv1controllers.NewDeploymentInterceptorWebhook(mgr.GetClient(), decoder),
	})
}

func getNamespace(ctx context.Context) string {
	tracer := otel.GetTracerProvider().Tracer(v1.BootstrapTracer)
	ctx, span := tracer.Start(ctx, "getNamespace") // nolint:ineffassign,staticcheck
	defer span.End()

	podNamespace, found := os.LookupEnv("POD_NAMESPACE")
	if !found {
		log.Log.V(1).Info(
			"the POD_NAMESPACE env var isn't set, trying to determine it from the service account info",
		)
		var err error
		if podNamespace, err = util.GetOperatorNamespace(); err != nil {
			span.SetStatus(codes.Error, err.Error())
			log.Log.Error(err, "could not read the namespace from the service account")
		}
	}

	return podNamespace
}

func tlsConfigSetting(cfg *tls.Config, tlsOpt tlsConfig) {
	version, err := k8sapiflag.TLSVersion(tlsOpt.minVersion)
	if err != nil {
		setupLog.Error(err, "TLS version invalid")
	}
	cfg.MinVersion = version

	cipherSuiteIDs, err := k8sapiflag.TLSCipherSuites(tlsOpt.cipherSuites)
	if err != nil {
		setupLog.Error(err, "Failed to convert TLS cipher suite name to ID")
	}
	cfg.CipherSuites = cipherSuiteIDs
}
