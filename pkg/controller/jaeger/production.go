package jaeger

import (
	"context"

	"github.com/sirupsen/logrus"
	"github.com/spf13/viper"
	batchv1 "k8s.io/api/batch/v1"
	"k8s.io/apimachinery/pkg/runtime"

	"github.com/jaegertracing/jaeger-operator/pkg/account"
	"github.com/jaegertracing/jaeger-operator/pkg/apis/io/v1alpha1"
	"github.com/jaegertracing/jaeger-operator/pkg/deployment"
	"github.com/jaegertracing/jaeger-operator/pkg/ingress"
	"github.com/jaegertracing/jaeger-operator/pkg/inject"
	"github.com/jaegertracing/jaeger-operator/pkg/route"
	"github.com/jaegertracing/jaeger-operator/pkg/storage"
)

type productionController struct {
	ctx    context.Context
	jaeger *v1alpha1.Jaeger
}

func newProductionController(ctx context.Context, jaeger *v1alpha1.Jaeger) *productionController {
	return &productionController{
		ctx:    ctx,
		jaeger: jaeger,
	}
}

func (c *productionController) Create() []runtime.Object {
	collector := deployment.NewCollector(c.jaeger)
	query := deployment.NewQuery(c.jaeger)
	agent := deployment.NewAgent(c.jaeger)
	os := []runtime.Object{}

	// add all service accounts
	for _, acc := range account.Get(c.jaeger) {
		os = append(os, acc)
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

	return os
}

func (c *productionController) Update() []runtime.Object {
	logrus.Debug("Update isn't yet available")
	return []runtime.Object{}
}

func (c *productionController) Dependencies() []batchv1.Job {
	return storage.Dependencies(c.jaeger)
}
