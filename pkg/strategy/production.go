package strategy

import (
	"context"
	"strings"

	"github.com/sirupsen/logrus"
	"github.com/spf13/viper"
	batchv1 "k8s.io/api/batch/v1"
	batchv1beta1 "k8s.io/api/batch/v1beta1"
	"k8s.io/apimachinery/pkg/runtime"

	"github.com/jaegertracing/jaeger-operator/pkg/account"
	"github.com/jaegertracing/jaeger-operator/pkg/apis/io/v1alpha1"
	"github.com/jaegertracing/jaeger-operator/pkg/config/sampling"
	"github.com/jaegertracing/jaeger-operator/pkg/config/ui"
	"github.com/jaegertracing/jaeger-operator/pkg/cronjob"
	"github.com/jaegertracing/jaeger-operator/pkg/deployment"
	"github.com/jaegertracing/jaeger-operator/pkg/ingress"
	"github.com/jaegertracing/jaeger-operator/pkg/inject"
	"github.com/jaegertracing/jaeger-operator/pkg/route"
	"github.com/jaegertracing/jaeger-operator/pkg/storage"
)

type productionStrategy struct {
	ctx    context.Context
	jaeger *v1alpha1.Jaeger
}

func newProductionStrategy(ctx context.Context, jaeger *v1alpha1.Jaeger) *productionStrategy {
	return &productionStrategy{
		ctx:    ctx,
		jaeger: jaeger,
	}
}

func (c *productionStrategy) Create() []runtime.Object {
	collector := deployment.NewCollector(c.jaeger)
	query := deployment.NewQuery(c.jaeger)
	agent := deployment.NewAgent(c.jaeger)
	os := []runtime.Object{}

	// add all service accounts
	for _, acc := range account.Get(c.jaeger) {
		os = append(os, acc)
	}

	// add the config map
	cm := configmap.NewUIConfig(c.jaeger).Get()
	if nil != cm {
		os = append(os, cm)
	}

	// add the Sampling config map
	scmp := sampling.NewConfig(c.jaeger).Get()
	if nil != scmp {
		os = append(os, scmp)
	}

	cDep := collector.Get()
	queryDep := inject.OAuthProxy(c.jaeger, query.Get())

	// add the deployments
	os = append(os, cDep, queryDep)

	if ds := agent.Get(); nil != ds {
		os = append(os, ds)
	}

	// add the services
	for _, svc := range collector.Services() {
		os = append(os, svc)
	}

	for _, svc := range query.Services() {
		os = append(os, svc)
	}

	// add the routes/ingresses
	if viper.GetString("platform") == v1alpha1.FlagPlatformOpenShift {
		if q := route.NewQueryRoute(c.jaeger).Get(); nil != q {
			os = append(os, q)
		}
	} else {
		if q := ingress.NewQueryIngress(c.jaeger).Get(); nil != q {
			os = append(os, q)
		}
	}

	if isBoolTrue(c.jaeger.Spec.Storage.SparkDependencies.Enabled) {
		if cronjob.SupportedStorage(c.jaeger.Spec.Storage.Type) {
			os = append(os, cronjob.CreateSparkDependencies(c.jaeger))
		} else {
			logrus.WithField("type", c.jaeger.Spec.Storage.Type).Warn("Skipping spark dependencies job due to unsupported storage.")
		}
	}

	var indexCleaner *batchv1beta1.CronJob
	if isBoolTrue(c.jaeger.Spec.Storage.EsIndexCleaner.Enabled) {
		if strings.EqualFold(c.jaeger.Spec.Storage.Type, "elasticsearch") {
			indexCleaner = cronjob.CreateEsIndexCleaner(c.jaeger)
			os = append(os, indexCleaner)
		} else {
			logrus.WithField("type", c.jaeger.Spec.Storage.Type).Warn("Skipping Elasticsearch index cleaner job due to unsupported storage.")
		}
	}

	if storage.ShouldDeployElasticsearch(c.jaeger.Spec.Storage) {
		es := &storage.ElasticsearchDeployment{
			Jaeger: c.jaeger,
		}
		objs, err := es.CreateElasticsearchObjects(cDep.Spec.Template.Spec.ServiceAccountName, queryDep.Spec.Template.Spec.ServiceAccountName)
		if err != nil {
			logrus.Error("Could not create Elasticsearch objects, Elasticsearch will not be deployed", err)
		} else {
			os = append(os, objs...)
			es.InjectStorageConfiguration(&queryDep.Spec.Template.Spec)
			es.InjectStorageConfiguration(&cDep.Spec.Template.Spec)
			if indexCleaner != nil {
				es.InjectIndexCleanerConfiguration(&indexCleaner.Spec.JobTemplate.Spec.Template.Spec)
			}
		}
	}

	return os
}

func (c *productionStrategy) Update() []runtime.Object {
	logrus.Debug("Update isn't yet available")
	return []runtime.Object{}
}

func (c *productionStrategy) Dependencies() []batchv1.Job {
	return storage.Dependencies(c.jaeger)
}
