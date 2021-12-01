package start

import (
	"context"
	"flag"
	"fmt"
	"os"
	"runtime"
	"strings"

	osimagev1 "github.com/openshift/api/image/v1"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/pflag"
	"github.com/spf13/viper"
	"go.opentelemetry.io/otel"
	otelattribute "go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	corev1 "k8s.io/api/core/v1"
	k8sruntime "k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/rest"

	//  import OIDC cluster authentication plugin, e.g. for IBM Cloud
	_ "k8s.io/client-go/plugin/pkg/client/auth/oidc"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/cache"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/healthz"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"
	"sigs.k8s.io/controller-runtime/pkg/manager"

	jaegertracingv1 "github.com/jaegertracing/jaeger-operator/apis/v1"
	v1 "github.com/jaegertracing/jaeger-operator/apis/v1"
	appsv1controllers "github.com/jaegertracing/jaeger-operator/controllers/appsv1"
	jaegertracingcontrollers "github.com/jaegertracing/jaeger-operator/controllers/jaegertracing"
	"github.com/jaegertracing/jaeger-operator/pkg/autodetect"
	kafkav1beta2 "github.com/jaegertracing/jaeger-operator/pkg/kafka/v1beta2"
	opmetrics "github.com/jaegertracing/jaeger-operator/pkg/metrics"
	"github.com/jaegertracing/jaeger-operator/pkg/tracing"
	"github.com/jaegertracing/jaeger-operator/pkg/upgrade"
	"github.com/jaegertracing/jaeger-operator/pkg/util"
	"github.com/jaegertracing/jaeger-operator/pkg/version"
)

var (
	scheme   = k8sruntime.NewScheme()
	setupLog = ctrl.Log.WithName("setup")
)

func init() {
	utilruntime.Must(clientgoscheme.AddToScheme(scheme))
	utilruntime.Must(jaegertracingv1.AddToScheme(scheme))
	utilruntime.Must(kafkav1beta2.AddToScheme(scheme))
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

	log.WithFields(log.Fields{
		"os":              runtime.GOOS,
		"arch":            runtime.GOARCH,
		"version":         runtime.Version(),
		"jaeger-operator": version.Get().Operator,
		"identity":        viper.GetString(v1.ConfigIdentity),
		"jaeger":          version.Get().Jaeger,
	}).Info("Versions")

	cfg, err := ctrl.GetConfig()
	if err != nil {
		span.SetStatus(codes.Error, err.Error())
		log.Fatal(err)
	}

	span.SetAttributes(otelattribute.String("Platform", viper.GetString("platform")))
	watchNamespace, found := os.LookupEnv("WATCH_NAMESPACE")
	if found {
		setupLog.Info("watching namespace(s)", "namespaces", watchNamespace)
	} else {
		setupLog.Info("the env var WATCH_NAMESPACE isn't set, watching all namespaces")
	}

	setOperatorScope(ctx, watchNamespace)

	mgr := createManager(ctx, cfg)

	detectNamespacePermissions(ctx, mgr)
	performUpgrades(ctx, mgr)
	setupControllers(ctx, mgr)
	detectOAuthProxyImageStream(ctx, mgr)
	err = opmetrics.Bootstrap(ctx, namespace, mgr.GetClient())
	if err != nil {
		log.WithError(err).Error("failed to initialize metrics")
	}
	if d, err := autodetect.New(mgr); err != nil {
		log.WithError(err).Warn("failed to start the background process to auto-detect the operator capabilities")
	} else {
		d.Start()
	}
	return mgr
}

func detectOAuthProxyImageStream(ctx context.Context, mgr manager.Manager) {
	tracer := otel.GetTracerProvider().Tracer(v1.BootstrapTracer)
	ctx, span := tracer.Start(ctx, "detectOAuthProxyImageStream")
	defer span.End()

	if viper.GetString("platform") != v1.FlagPlatformOpenShift {
		log.Debug("Not running on OpenShift, so won't configure OAuthProxy imagestream.")
		return
	}

	imageStreamNamespace := viper.GetString("openshift-oauth-proxy-imagestream-ns")
	imageStreamName := viper.GetString("openshift-oauth-proxy-imagestream-name")
	if imageStreamNamespace == "" || imageStreamName == "" {
		log.WithFields(log.Fields{
			"namespace": imageStreamNamespace,
			"name":      imageStreamName,
		}).Info("OAuthProxy ImageStream namespace and/or name not defined")
		return
	}

	imageStream := &osimagev1.ImageStream{}
	namespacedName := types.NamespacedName{
		Name:      imageStreamName,
		Namespace: imageStreamNamespace,
	}
	if err := mgr.GetAPIReader().Get(ctx, namespacedName, imageStream); err != nil {
		log.WithFields(log.Fields{
			"namespace": imageStreamNamespace,
			"name":      imageStreamName,
		}).WithError(err).Error("Failed to obtain OAuthProxy ImageStream")
		tracing.HandleError(err, span)
		return
	}

	if len(imageStream.Status.Tags) == 0 {
		log.WithFields(log.Fields{
			"namespace": imageStreamNamespace,
			"name":      imageStreamName,
		}).Error("OAuthProxy ImageStream has no tags")
		return
	}

	if len(imageStream.Status.Tags[0].Items) == 0 {
		log.WithFields(log.Fields{
			"namespace": imageStreamNamespace,
			"name":      imageStreamName,
		}).Error("OAuthProxy ImageStream tag has no items")
		return
	}

	if len(imageStream.Status.Tags[0].Items[0].DockerImageReference) == 0 {
		log.WithFields(log.Fields{
			"namespace": imageStreamNamespace,
			"name":      imageStreamName,
		}).Error("OAuthProxy ImageStream tag has no DockerImageReference")
		return
	}

	image := imageStream.Status.Tags[0].Items[0].DockerImageReference

	viper.Set("openshift-oauth-proxy-image", image)
	log.WithFields(log.Fields{
		"image": image,
	}).Info("Updated OAuth Proxy image flag")
}

func detectNamespacePermissions(ctx context.Context, mgr manager.Manager) {
	tracer := otel.GetTracerProvider().Tracer(v1.BootstrapTracer)
	ctx, span := tracer.Start(ctx, "detectNamespacePermissions")
	defer span.End()

	namespaces := &corev1.NamespaceList{}
	opts := []client.ListOption{}
	if err := mgr.GetAPIReader().List(ctx, namespaces, opts...); err != nil {
		log.WithError(err).Trace("could not get a list of namespaces, disabling namespace controller")
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
	ctx, span := tracer.Start(ctx, "setOperatorScope")
	defer span.End()

	// set what's the namespace to watch
	viper.Set(v1.ConfigWatchNamespace, namespace)

	// for now, the logic is simple: if we are watching all namespaces, then we are cluster-wide
	if viper.GetString(v1.ConfigWatchNamespace) == v1.WatchAllNamespaces {
		span.SetAttributes(otelattribute.String(v1.ConfigOperatorScope, v1.OperatorScopeCluster))
		viper.Set(v1.ConfigOperatorScope, v1.OperatorScopeCluster)
	} else {
		log.Info("Consider running the operator in a cluster-wide scope for extra features")
		span.SetAttributes(otelattribute.String(v1.ConfigOperatorScope, v1.OperatorScopeNamespace))
		viper.Set(v1.ConfigOperatorScope, v1.OperatorScopeNamespace)
	}
}

func setLogLevel(ctx context.Context) {
	tracer := otel.GetTracerProvider().Tracer(v1.BootstrapTracer)
	ctx, span := tracer.Start(ctx, "setLogLevel")
	defer span.End()

	level, err := log.ParseLevel(viper.GetString("log-level"))
	if err != nil {
		log.SetLevel(log.InfoLevel)
	} else {
		log.SetLevel(level)
	}
}

func buildIdentity(ctx context.Context, podNamespace string) {
	tracer := otel.GetTracerProvider().Tracer(v1.BootstrapTracer)
	ctx, span := tracer.Start(ctx, "buildIdentity")
	defer span.End()

	operatorName, found := os.LookupEnv("OPERATOR_NAME")
	if !found {
		log.Warn("the OPERATOR_NAME env var isn't set, this operator's identity might clash with another operator's instance")
		operatorName = "jaeger-operator"
	}

	var identity string
	if len(podNamespace) > 0 {
		identity = fmt.Sprintf("%s.%s", podNamespace, operatorName)
	} else {
		identity = fmt.Sprintf("%s", operatorName)
	}

	span.SetAttributes(otelattribute.String(v1.ConfigIdentity, identity))
	viper.Set(v1.ConfigIdentity, identity)
}

func createManager(ctx context.Context, cfg *rest.Config) manager.Manager {
	tracer := otel.GetTracerProvider().Tracer(v1.BootstrapTracer)
	ctx, span := tracer.Start(ctx, "createManager")
	defer span.End()

	metricsHost := viper.GetString("metrics-host")
	metricsPort := viper.GetInt("metrics-port")
	metricsAddr := fmt.Sprintf("%s:%d", metricsHost, metricsPort)
	enableLeaderElection := viper.GetBool("leader-elect")
	probeAddr := viper.GetString("health-probe-bind-address")

	opts := zap.Options{
		Development: true,
	}
	opts.BindFlags(flag.CommandLine)
	pflag.CommandLine.AddGoFlagSet(flag.CommandLine)

	ctrl.SetLogger(zap.New(zap.UseFlagOptions(&opts)))

	namespace := viper.GetString(v1.ConfigWatchNamespace)

	options := ctrl.Options{
		Scheme:                 scheme,
		MetricsBindAddress:     metricsAddr,
		Port:                   9443,
		HealthProbeBindAddress: probeAddr,
		LeaderElection:         enableLeaderElection,
		LeaderElectionID:       "31e04290.jaegertracing.io",
		Namespace:              namespace,
	}

	// Add support for MultiNamespace set in WATCH_NAMESPACE (e.g ns1,ns2)
	// Note that this is not intended to be used for excluding namespaces, this is better done via a Predicate
	// Also note that you may face performance issues when using this with a high number of namespaces.
	// More Info: https://godoc.org/github.com/kubernetes-sigs/controller-runtime/pkg/cache#MultiNamespacedCacheBuilder
	if strings.Contains(namespace, ",") {
		options.Namespace = ""
		options.NewCache = cache.MultiNamespacedCacheBuilder(strings.Split(namespace, ","))
	}

	// Create a new manager to provide shared dependencies and start components
	mgr, err := ctrl.NewManager(cfg, options)
	if err != nil {
		span.SetStatus(codes.Error, err.Error())
		log.Fatal(err)
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
		log.WithError(err).Warn("failed to upgrade managed instances")
	}
}

func setupControllers(ctx context.Context, mgr manager.Manager) {
	tracer := otel.GetTracerProvider().Tracer(v1.BootstrapTracer)
	ctx, span := tracer.Start(ctx, "setupControllers")
	clientReader := mgr.GetAPIReader()
	client := mgr.GetClient()
	schema := mgr.GetScheme()
	defer span.End()

	if err := jaegertracingcontrollers.NewReconciler(client, clientReader, schema).SetupWithManager(mgr); err != nil {
		setupLog.Error(err, "unable to create controller", "controller", "Jaeger")
		os.Exit(1)
	}

	if err := appsv1controllers.NewNamespaceReconciler(client, clientReader, schema).SetupWithManager(mgr); err != nil {
		setupLog.Error(err, "unable to create controller", "controller", "Namespace")
		os.Exit(1)
	}

	if err := appsv1controllers.NewDeploymentReconciler(client, clientReader, schema).SetupWithManager(mgr); err != nil {
		setupLog.Error(err, "unable to create controller", "controller", "Deployment")
		os.Exit(1)
	}
}

func getNamespace(ctx context.Context) string {
	tracer := otel.GetTracerProvider().Tracer(v1.BootstrapTracer)
	ctx, span := tracer.Start(ctx, "getNamespace")
	defer span.End()

	podNamespace, found := os.LookupEnv("POD_NAMESPACE")
	if !found {
		log.Warn("the POD_NAMESPACE env var isn't set, trying to determine it from the service account info")
		var err error
		if podNamespace, err = util.GetOperatorNamespace(); err != nil {
			span.SetStatus(codes.Error, err.Error())
			log.WithError(err).Warn("could not read the namespace from the service account")
		}
	}

	return podNamespace
}
