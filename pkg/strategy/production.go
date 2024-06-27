package strategy

import (
	"context"

	batchv1 "k8s.io/api/batch/v1"
	"k8s.io/apimachinery/pkg/runtime"

	"github.com/spf13/viper"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	appsv1 "k8s.io/api/apps/v1"
	batchv1beta1 "k8s.io/api/batch/v1beta1"
	corev1 "k8s.io/api/core/v1"

	v1 "github.com/jaegertracing/jaeger-operator/apis/v1"
	"github.com/jaegertracing/jaeger-operator/pkg/account"
	"github.com/jaegertracing/jaeger-operator/pkg/autodetect"
	crb "github.com/jaegertracing/jaeger-operator/pkg/clusterrolebinding"
	"github.com/jaegertracing/jaeger-operator/pkg/config/ca"
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
	tracer := otel.GetTracerProvider().Tracer(v1.ReconciliationTracer)
	ctx, span := tracer.Start(ctx, "newProductionStrategy") // nolint:ineffassign,staticcheck
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
	if autodetect.OperatorConfiguration.GetPlatform() == autodetect.OpenShiftPlatform {
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
	span.SetAttributes(attribute.String("Platform", autodetect.OperatorConfiguration.GetPlatform().String()))

	// add autoscalers
	c.horizontalPodAutoscalers = collector.Autoscalers()

	if isBoolTrue(jaeger.Spec.Storage.Dependencies.Enabled) {
		if cronjob.SupportedStorage(jaeger.Spec.Storage.Type) {
			c.cronJobs = append(c.cronJobs, cronjob.CreateSparkDependencies(jaeger))
		} else {
			jaeger.Logger().V(1).Info(
				"skipping spark dependencies job due to unsupported storage.",
				"type", jaeger.Spec.Storage.Type,
			)
		}
	}

	var indexCleaner runtime.Object
	if isBoolTrue(jaeger.Spec.Storage.EsIndexCleaner.Enabled) {
		if jaeger.Spec.Storage.Type == v1.JaegerESStorage {
			indexCleaner = cronjob.CreateEsIndexCleaner(jaeger)
		} else {
			jaeger.Logger().V(1).Info(
				"skipping Elasticsearch index cleaner job due to unsupported storage.",
				"type", jaeger.Spec.Storage.Type,
			)
		}
	}

	var esRollover []runtime.Object
	if storage.EnableRollover(jaeger.Spec.Storage) {
		esRollover = cronjob.CreateRollover(jaeger)
	}

	// prepare the deployments, which may get changed by the elasticsearch routine
	cDep := collector.Get()
	queryDep := inject.OAuthProxy(jaeger, query.Get())
	c.dependencies = storage.Dependencies(jaeger)

	// assembles the pieces for an elasticsearch self-provisioned deployment via the elasticsearch operator
	if v1.ShouldInjectOpenShiftElasticsearchConfiguration(jaeger.Spec.Storage) {
		var jobs []*corev1.PodSpec
		for i := range c.dependencies {
			jobs = append(jobs, &c.dependencies[i].Spec.Template.Spec)
		}
		cronjobsVersion := viper.GetString(v1.FlagCronJobsVersion)
		if indexCleaner != nil {
			if cronjobsVersion == v1.FlagCronJobsVersionBatchV1Beta1 {
				jobs = append(jobs, &indexCleaner.(*batchv1beta1.CronJob).Spec.JobTemplate.Spec.Template.Spec)
			} else {
				jobs = append(jobs, &indexCleaner.(*batchv1.CronJob).Spec.JobTemplate.Spec.Template.Spec)
			}
		}
		for i := range esRollover {
			if cronjobsVersion == v1.FlagCronJobsVersionBatchV1Beta1 {
				jobs = append(jobs, &esRollover[i].(*batchv1beta1.CronJob).Spec.JobTemplate.Spec.Template.Spec)
			} else {
				jobs = append(jobs, &esRollover[i].(*batchv1.CronJob).Spec.JobTemplate.Spec.Template.Spec)
			}
		}
		autoProvisionElasticsearch(&c, jaeger, jobs, []*appsv1.Deployment{queryDep, cDep})
	}

	// the index cleaner ES job, which may have been changed by the ES self-provisioning routine
	if indexCleaner != nil {
		c.cronJobs = append(c.cronJobs, indexCleaner)
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
	esCR := es.Elasticsearch()
	if esCR != nil {
		manifest.elasticsearches = append(manifest.elasticsearches, *esCR)
	}
}
