package strategy

import (
	"context"

	"github.com/sirupsen/logrus"
	"github.com/spf13/viper"
	batchv1 "k8s.io/api/batch/v1"
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

	// add the deployments
	os = append(os,
		collector.Get(),
		inject.OAuthProxy(c.jaeger, query.Get()),
	)

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

	if cronjob.SupportedStorage(c.jaeger.Spec.Storage.Type) {
		if c.jaeger.Spec.Storage.SparkDependencies.Enabled {
			os = append(os, cronjob.CreateSparkDependencies(c.jaeger))
		} else {
			logrus.Info("Do not installing spark dependencies - need to be enabled explicitly")
		}
	}

	if c.jaeger.Spec.Storage.EsIndexCleaner.Enabled {
		if c.jaeger.Spec.Storage.Type == "elasticsearch" {
			os = append(os, cronjob.CreateEsIndexCleaner(c.jaeger))
		} else {
			logrus.WithField("type", c.jaeger.Spec.Storage.Type).Warn("Elasticsearch index cleaner is enabled for unsupported storage")
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
