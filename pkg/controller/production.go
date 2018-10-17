package controller

import (
	"context"

	"github.com/operator-framework/operator-sdk/pkg/sdk"
	"github.com/sirupsen/logrus"
	batchv1 "k8s.io/api/batch/v1"

	"github.com/jaegertracing/jaeger-operator/pkg/apis/io/v1alpha1"
	"github.com/jaegertracing/jaeger-operator/pkg/deployment"
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

func (c *productionController) Create() []sdk.Object {
	collector := deployment.NewCollector(c.jaeger)
	query := deployment.NewQuery(c.jaeger)
	agent := deployment.NewAgent(c.jaeger)

	components := []sdk.Object{
		collector.Get(),
		query.Get(),
	}

	ds := agent.Get()
	if nil != ds {
		components = append(components, ds)
	}

	for _, svc := range collector.Services() {
		components = append(components, svc)
	}

	for _, svc := range query.Services() {
		components = append(components, svc)
	}

	for _, ingress := range query.Ingresses() {
		components = append(components, ingress)
	}

	return components
}

func (c *productionController) Update() []sdk.Object {
	logrus.Debug("Update isn't yet available")
	return []sdk.Object{}
}

func (c *productionController) Dependencies() []batchv1.Job {
	return storage.Dependencies(c.jaeger)
}
