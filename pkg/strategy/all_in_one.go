package strategy

import (
	"context"
	"strings"

	"github.com/spf13/viper"
	"go.opentelemetry.io/otel/global"
	appsv1 "k8s.io/api/apps/v1"

	"github.com/jaegertracing/jaeger-operator/pkg/account"
	v1 "github.com/jaegertracing/jaeger-operator/pkg/apis/jaegertracing/v1"
	crb "github.com/jaegertracing/jaeger-operator/pkg/clusterrolebinding"
	"github.com/jaegertracing/jaeger-operator/pkg/config/ca"
	"github.com/jaegertracing/jaeger-operator/pkg/config/otelconfig"
	"github.com/jaegertracing/jaeger-operator/pkg/config/sampling"
	configmap "github.com/jaegertracing/jaeger-operator/pkg/config/ui"
	"github.com/jaegertracing/jaeger-operator/pkg/consolelink"
	"github.com/jaegertracing/jaeger-operator/pkg/cronjob"
	"github.com/jaegertracing/jaeger-operator/pkg/deployment"
	"github.com/jaegertracing/jaeger-operator/pkg/ingress"
	"github.com/jaegertracing/jaeger-operator/pkg/inject"
	"github.com/jaegertracing/jaeger-operator/pkg/route"
	"github.com/jaegertracing/jaeger-operator/pkg/storage"
)

func newAllInOneStrategy(ctx context.Context, jaeger *v1.Jaeger) S {
	tracer := global.TraceProvider().GetTracer(v1.ReconciliationTracer)
	ctx, span := tracer.Start(ctx, "newAllInOneStrategy")
	defer span.End()

	c := S{typ: v1.DeploymentStrategyAllInOne}
	jaeger.Logger().Debug("Creating all-in-one deployment")

	dep := deployment.NewAllInOne(jaeger)

	// add all service accounts
	for _, acc := range account.Get(jaeger) {
		c.accounts = append(c.accounts, *acc)
	}

	// add all cluster role bindings
	c.clusterRoleBindings = crb.Get(jaeger)

	// add the UI config map
	if cm := configmap.NewUIConfig(jaeger).Get(); cm != nil {
		c.configMaps = append(c.configMaps, *cm)
	}

	// add the Sampling config map
	if cm := sampling.NewConfig(jaeger).Get(); cm != nil {
		c.configMaps = append(c.configMaps, *cm)
	}

	// add the optional OpenShift trusted CA config map
	if cm := ca.GetTrustedCABundle(jaeger); cm != nil {
		c.configMaps = append(c.configMaps, *cm)
	}

	// add the service CA config map
	if cm := ca.GetServiceCABundle(jaeger); cm != nil {
		c.configMaps = append(c.configMaps, *cm)
	}

	if cm := otelconfig.Get(jaeger); len(cm) > 0 {
		c.configMaps = append(c.configMaps, cm...)
	}

	// add the deployments
	c.deployments = []appsv1.Deployment{*inject.OAuthProxy(jaeger, dep.Get())}

	// add the daemonsets
	if ds := deployment.NewAgent(jaeger).Get(); ds != nil {
		c.daemonSets = []appsv1.DaemonSet{*ds}
	}

	// add the services
	for _, svc := range dep.Services() {
		c.services = append(c.services, *svc)
	}

	// add the routes/ingresses
	if viper.GetString("platform") == v1.FlagPlatformOpenShift {
		if q := route.NewQueryRoute(jaeger).Get(); nil != q {
			c.routes = append(c.routes, *q)
			if link := consolelink.Get(jaeger, q); link != nil {
				c.consoleLinks = append(c.consoleLinks, *link)
			}
		}
	} else {
		if q := ingress.NewQueryIngress(jaeger).Get(); nil != q {
			c.ingresses = append(c.ingresses, *q)
		}
	}

	if isBoolTrue(jaeger.Spec.Storage.Dependencies.Enabled) {
		if cronjob.SupportedStorage(jaeger.Spec.Storage.Type) {
			c.cronJobs = append(c.cronJobs, *cronjob.CreateSparkDependencies(jaeger))
		} else {
			jaeger.Logger().WithField("type", jaeger.Spec.Storage.Type).Warn("Skipping spark dependencies job due to unsupported storage.")
		}
	}

	if isBoolTrue(jaeger.Spec.Storage.EsIndexCleaner.Enabled) {
		if strings.EqualFold(jaeger.Spec.Storage.Type, "elasticsearch") {
			c.cronJobs = append(c.cronJobs, *cronjob.CreateEsIndexCleaner(jaeger))
		} else {
			jaeger.Logger().WithField("type", jaeger.Spec.Storage.Type).Warn("Skipping Elasticsearch index cleaner job due to unsupported storage.")
		}
	}

	if storage.EnableRollover(jaeger.Spec.Storage) {
		c.cronJobs = append(c.cronJobs, cronjob.CreateRollover(jaeger)...)
	}

	c.dependencies = storage.Dependencies(jaeger)

	return c
}

func isBoolTrue(b *bool) bool {
	return b != nil && *b
}
