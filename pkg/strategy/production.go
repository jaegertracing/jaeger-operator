package strategy

import (
	"context"
	"strings"

	corev1 "k8s.io/api/core/v1"

	"github.com/spf13/viper"
	"go.opentelemetry.io/otel/api/key"
	"go.opentelemetry.io/otel/global"
	appsv1 "k8s.io/api/apps/v1"
	batchv1beta1 "k8s.io/api/batch/v1beta1"

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

func newProductionStrategy(ctx context.Context, jaeger *v1.Jaeger) S {
	tracer := global.TraceProvider().GetTracer(v1.ReconciliationTracer)
	ctx, span := tracer.Start(ctx, "newProductionStrategy")
	defer span.End()

	c := S{typ: v1.DeploymentStrategyProduction}
	collector := deployment.NewCollector(jaeger)
	query := deployment.NewQuery(jaeger)
	agent := deployment.NewAgent(jaeger)

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
			if link := consolelink.Get(jaeger, q); link != nil {
				c.consoleLinks = append(c.consoleLinks, *link)
			}
		}
	} else {
		span.SetAttribute(key.String("Platform", v1.FlagPlatformKubernetes))
		if q := ingress.NewQueryIngress(jaeger).Get(); nil != q {
			c.ingresses = append(c.ingresses, *q)
		}
	}

	// add autoscalers
	c.horizontalPodAutoscalers = collector.Autoscalers()

	if isBoolTrue(jaeger.Spec.Storage.Dependencies.Enabled) {
		if cronjob.SupportedStorage(jaeger.Spec.Storage.Type) {
			c.cronJobs = append(c.cronJobs, *cronjob.CreateSparkDependencies(jaeger))
		} else {
			jaeger.Logger().WithField("type", jaeger.Spec.Storage.Type).Warn("Skipping spark dependencies job due to unsupported storage.")
		}
	}

	var indexCleaner *batchv1beta1.CronJob
	if isBoolTrue(jaeger.Spec.Storage.EsIndexCleaner.Enabled) {
		if strings.EqualFold(jaeger.Spec.Storage.Type, "elasticsearch") {
			indexCleaner = cronjob.CreateEsIndexCleaner(jaeger)
		} else {
			jaeger.Logger().WithField("type", jaeger.Spec.Storage.Type).Warn("Skipping Elasticsearch index cleaner job due to unsupported storage.")
		}
	}

	var esRollover []batchv1beta1.CronJob
	if storage.EnableRollover(jaeger.Spec.Storage) {
		esRollover = cronjob.CreateRollover(jaeger)
	}

	// prepare the deployments, which may get changed by the elasticsearch routine
	cDep := collector.Get()
	queryDep := inject.Sidecar(jaeger, inject.OAuthProxy(jaeger, query.Get()))
	c.dependencies = storage.Dependencies(jaeger)

	// assembles the pieces for an elasticsearch self-provisioned deployment via the elasticsearch operator
	if storage.ShouldDeployElasticsearch(jaeger.Spec.Storage) {
		var jobs []*corev1.PodSpec
		for i := range c.dependencies {
			jobs = append(jobs, &c.dependencies[i].Spec.Template.Spec)
		}
		if indexCleaner != nil {
			jobs = append(jobs, &indexCleaner.Spec.JobTemplate.Spec.Template.Spec)
		}
		for i := range esRollover {
			jobs = append(jobs, &esRollover[i].Spec.JobTemplate.Spec.Template.Spec)
		}
		autoProvisionElasticsearch(&c, jaeger, jobs, []*appsv1.Deployment{queryDep, cDep})
	}

	// the index cleaner ES job, which may have been changed by the ES self-provisioning routine
	if indexCleaner != nil {
		c.cronJobs = append(c.cronJobs, *indexCleaner)
	}
	if len(esRollover) > 0 {
		c.cronJobs = append(c.cronJobs, esRollover...)
	}

	// add the deployments, which may have been changed by the ES self-provisioning routine
	c.deployments = []appsv1.Deployment{*cDep, *queryDep}

	return c
}

func autoProvisionElasticsearch(manifest *S, jaeger *v1.Jaeger, curatorPods []*corev1.PodSpec, deployments []*appsv1.Deployment) {
	es := &storage.ElasticsearchDeployment{Jaeger: jaeger}
	for i := range deployments {
		es.InjectStorageConfiguration(&deployments[i].Spec.Template.Spec)
	}
	for _, pod := range curatorPods {
		es.InjectSecretsConfiguration(pod)
	}
	manifest.elasticsearches = append(manifest.elasticsearches, *es.Elasticsearch())
}
