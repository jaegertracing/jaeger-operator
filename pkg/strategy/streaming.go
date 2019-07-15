package strategy

import (
	"strings"

	"github.com/spf13/viper"
	appsv1 "k8s.io/api/apps/v1"

	"github.com/jaegertracing/jaeger-operator/pkg/account"
	v1 "github.com/jaegertracing/jaeger-operator/pkg/apis/jaegertracing/v1"
	crb "github.com/jaegertracing/jaeger-operator/pkg/clusterrolebinding"
	"github.com/jaegertracing/jaeger-operator/pkg/config/sampling"
	configmap "github.com/jaegertracing/jaeger-operator/pkg/config/ui"
	"github.com/jaegertracing/jaeger-operator/pkg/cronjob"
	"github.com/jaegertracing/jaeger-operator/pkg/deployment"
	"github.com/jaegertracing/jaeger-operator/pkg/ingress"
	"github.com/jaegertracing/jaeger-operator/pkg/inject"
	"github.com/jaegertracing/jaeger-operator/pkg/route"
	"github.com/jaegertracing/jaeger-operator/pkg/storage"
)

func newStreamingStrategy(jaeger *v1.Jaeger) S {
	c := S{typ: Streaming}
	collector := deployment.NewCollector(jaeger)
	query := deployment.NewQuery(jaeger)
	agent := deployment.NewAgent(jaeger)
	ingester := deployment.NewIngester(jaeger)

	// add all service accounts
	for _, acc := range account.Get(jaeger) {
		c.accounts = append(c.accounts, *acc)
	}

	// add all cluster role bindings
	c.clusterRoleBindings = crb.Get(jaeger)

	// add the config map
	if cm := configmap.NewUIConfig(jaeger).Get(); cm != nil {
		c.configMaps = append(c.configMaps, *cm)
	}

	// add the Sampling config map
	if cm := sampling.NewConfig(jaeger).Get(); cm != nil {
		c.configMaps = append(c.configMaps, *cm)
	}

	// add the deployments
	c.deployments = []appsv1.Deployment{*collector.Get(), *inject.Sidecar(jaeger, inject.OAuthProxy(jaeger, query.Get()))}

	if d := ingester.Get(); d != nil {
		c.deployments = append(c.deployments, *d)
	}

	// add the daemonsets
	if ds := agent.Get(); ds != nil {
		c.daemonSets = []appsv1.DaemonSet{*ds}
	}

	// add the services
	for _, svc := range collector.Services() {
		c.services = append(c.services, *svc)
	}

	for _, svc := range query.Services() {
		c.services = append(c.services, *svc)
	}

	// add the routes/ingresses
	if viper.GetString("platform") == v1.FlagPlatformOpenShift {
		if q := route.NewQueryRoute(jaeger).Get(); nil != q {
			c.routes = append(c.routes, *q)
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

	c.dependencies = storage.Dependencies(jaeger)

	return c
}
