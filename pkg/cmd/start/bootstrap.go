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
	"github.com/spf13/viper"
	"go.opentelemetry.io/otel/api/core"
	"go.opentelemetry.io/otel/api/key"
	"go.opentelemetry.io/otel/exporter/trace/jaeger"
	"go.opentelemetry.io/otel/global"
	"google.golang.org/grpc/codes"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
	"k8s.io/client-go/rest"
	"sigs.k8s.io/controller-runtime/pkg/client/config"
	"sigs.k8s.io/controller-runtime/pkg/manager"

	"github.com/jaegertracing/jaeger-operator/pkg/apis"
	v1 "github.com/jaegertracing/jaeger-operator/pkg/apis/jaegertracing/v1"
	"github.com/jaegertracing/jaeger-operator/pkg/controller"
	"github.com/jaegertracing/jaeger-operator/pkg/tracing"
	"github.com/jaegertracing/jaeger-operator/pkg/upgrade"
	"github.com/jaegertracing/jaeger-operator/pkg/version"
)

func bootstrap(ctx context.Context) manager.Manager {
	tracing.Bootstrap()

	tracer := global.TraceProvider().GetTracer(v1.BootstrapTracer)
	ctx, span := tracer.Start(ctx, "bootstrap")
	defer span.End()

	setLogLevel(ctx)

	namespace := getNamespace(ctx)

	buildIdentity(ctx, namespace)

	if viper.GetBool("tracing-enabled") {
		buildJaegerExporter(ctx)
	}

	log.WithFields(log.Fields{
		"os":              runtime.GOOS,
		"arch":            runtime.GOARCH,
		"version":         runtime.Version(),
		"operator-sdk":    version.Get().OperatorSdk,
		"jaeger-operator": version.Get().Operator,
		"identity":        viper.GetString(v1.ConfigIdentity),
		"jaeger":          version.Get().Jaeger,
	}).Info("Versions")

	if err := leader.Become(ctx, "jaeger-operator-lock"); err != nil {
		log.Fatal(err)
	}

	cfg, err := config.GetConfig()
	if err != nil {
		span.SetStatus(codes.Internal)
		span.SetAttribute(key.String("error", err.Error()))
		log.Fatal(err)
	}

	mgr := createManager(ctx, cfg)

	performUpgrades(ctx, mgr)
	setupControllers(ctx, mgr)
	serveCRMetrics(ctx, cfg, namespace)
	createMetricsService(ctx, cfg, namespace)

	if viper.GetBool("provision-own-instance") {
		provisionOwnJaeger(ctx, mgr.GetClient(), namespace)
	}

	return mgr
}

func setLogLevel(ctx context.Context) {
	tracer := global.TraceProvider().GetTracer(v1.BootstrapTracer)
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
	tracer := global.TraceProvider().GetTracer(v1.BootstrapTracer)
	ctx, span := tracer.Start(ctx, "buildIdentity")
	defer span.End()

	operatorName, found := os.LookupEnv("OPERATOR_NAME")
	if !found {
		log.Warn("the OPERATOR_NAME env var isn't set, this operator's identity might clash with another operator's instance")
	}

	var identity string
	if len(podNamespace) > 0 {
		identity = fmt.Sprintf("%s.%s", podNamespace, operatorName)
	} else {
		identity = fmt.Sprintf("%s", operatorName)
	}
	viper.Set(v1.ConfigIdentity, identity)
}

func buildJaegerExporter(ctx context.Context) {
	tracer := global.TraceProvider().GetTracer(v1.BootstrapTracer)
	ctx, span := tracer.Start(ctx, "buildJaegerExporter")
	defer span.End()

	agentHostPort := viper.GetString("jaeger-agent-hostport")
	jaegerExporter, err := jaeger.NewExporter(
		jaeger.WithAgentEndpoint(agentHostPort),
		jaeger.WithProcess(jaeger.Process{
			ServiceName: "jaeger-operator",
			Tags: []core.KeyValue{
				key.String("operator.identity", viper.GetString(v1.ConfigIdentity)),
				key.String("operator.version", version.Get().Operator),
			},
		}),
		jaeger.WithOnError(func(err error) {
			log.WithError(err).Warn("failed to setup the Jaeger exporter")
		}),
	)
	if err == nil {
		tracing.AddJaegerExporter(jaegerExporter)
	} else {
		span.SetStatus(codes.Internal)
		log.WithError(err).Warn("could not configure a Jaeger tracer for the operator")
	}
}

func createManager(ctx context.Context, cfg *rest.Config) manager.Manager {
	tracer := global.TraceProvider().GetTracer(v1.BootstrapTracer)
	ctx, span := tracer.Start(ctx, "createManager")
	defer span.End()

	namespace, err := k8sutil.GetWatchNamespace()
	if err != nil {
		span.SetStatus(codes.Internal)
		span.SetAttribute(key.String("error", err.Error()))
		log.WithError(err).Fatal("failed to get watch namespace")
	}

	mgr, err := manager.New(cfg, manager.Options{
		Namespace:          namespace,
		MapperProvider:     restmapper.NewDynamicRESTMapper,
		MetricsBindAddress: fmt.Sprintf("%s:%d", viper.GetString("metrics-host"), viper.GetInt32("metrics-port")),
	})
	if err != nil {
		span.SetStatus(codes.Internal)
		span.SetAttribute(key.String("error", err.Error()))
		log.Fatal(err)
	}

	// Setup Scheme for all resources
	if err := apis.AddToScheme(mgr.GetScheme()); err != nil {
		span.SetStatus(codes.Internal)
		span.SetAttribute(key.String("error", err.Error()))
		log.Fatal(err)
	}

	return mgr
}

// serveCRMetrics gets the Operator/CustomResource GVKs and generates metrics based on those types.
// It serves those metrics on "http://metricsHost:operatorMetricsPort".
func serveCRMetrics(ctx context.Context, cfg *rest.Config, operatorNs string) {
	tracer := global.TraceProvider().GetTracer(v1.BootstrapTracer)
	ctx, span := tracer.Start(ctx, "serveCRMetrics")
	defer span.End()

	// Below function returns filtered operator/CustomResource specific GVKs.
	// For more control override the below GVK list with your own custom logic.
	filteredGVK, err := k8sutil.GetGVKsFromAddToScheme(apis.AddToScheme)
	if err != nil {
		span.SetStatus(codes.Internal)
		log.WithError(err).Warn("could not generate and serve custom resource metrics")
		return
	}

	ns := []string{operatorNs}
	err = kubemetrics.GenerateAndServeCRMetrics(cfg, ns, filteredGVK, viper.GetString("metrics-host"), viper.GetInt32("cr-metrics-port"))
	if err != nil {
		span.SetStatus(codes.Internal)
		span.SetAttribute(key.String("error", err.Error()))
		log.WithError(err).Warn("could not generate and serve custom resource metrics")
	}
}

func performUpgrades(ctx context.Context, mgr manager.Manager) {
	tracer := global.TraceProvider().GetTracer(v1.BootstrapTracer)
	ctx, span := tracer.Start(ctx, "performUpgrades")
	defer span.End()

	// upgrades all the instances managed by this operator
	if err := upgrade.ManagedInstances(ctx, mgr.GetClient()); err != nil {
		log.WithError(err).Warn("failed to upgrade managed instances")
	}
}

func setupControllers(ctx context.Context, mgr manager.Manager) {
	tracer := global.TraceProvider().GetTracer(v1.BootstrapTracer)
	ctx, span := tracer.Start(ctx, "setupControllers")
	defer span.End()

	if err := controller.AddToManager(mgr); err != nil {
		log.Fatal(err)
	}
}

func getNamespace(ctx context.Context) string {
	tracer := global.TraceProvider().GetTracer(v1.BootstrapTracer)
	ctx, span := tracer.Start(ctx, "getNamespace")
	defer span.End()

	podNamespace, found := os.LookupEnv("POD_NAMESPACE")
	if !found {
		log.Warn("the POD_NAMESPACE env var isn't set, trying to determine it from the service account info")

		var err error
		if podNamespace, err = k8sutil.GetOperatorNamespace(); err != nil {
			span.SetStatus(codes.Internal)
			span.SetAttribute(key.String("error", err.Error()))
			log.WithError(err).Warn("could not read the namespace from the service account")
		}
	}

	return podNamespace
}

func createMetricsService(ctx context.Context, cfg *rest.Config, namespace string) {
	tracer := global.TraceProvider().GetTracer(v1.BootstrapTracer)
	ctx, span := tracer.Start(ctx, "createMetricsService")
	defer span.End()

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
		span.SetStatus(codes.Internal)
		span.SetAttribute(key.String("error", err.Error()))
		log.WithError(err).Warn("could not create metrics Service")
	}

	createServiceMonitor(ctx, cfg, namespace, service)
}

func createServiceMonitor(ctx context.Context, cfg *rest.Config, namespace string, service *corev1.Service) {
	tracer := global.TraceProvider().GetTracer(v1.BootstrapTracer)
	ctx, span := tracer.Start(ctx, "createServiceMonitor")
	defer span.End()

	// CreateServiceMonitors will automatically create the prometheus-operator ServiceMonitor resources
	// necessary to configure Prometheus to scrape metrics from this operator.
	services := []*corev1.Service{service}
	_, err := metrics.CreateServiceMonitors(cfg, namespace, services)
	if err != nil {
		if err == metrics.ErrServiceMonitorNotPresent {
			log.WithError(err).Info("Install prometheus-operator in your cluster to create ServiceMonitor objects")
		} else {
			span.SetStatus(codes.Internal)
			span.SetAttribute(key.String("error", err.Error()))
			log.WithError(err).Warn("could not create ServiceMonitor object")
		}
	}
}
