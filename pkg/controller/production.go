package controller

import (
	"context"

	"github.com/jaegertracing/jaeger-operator/pkg/apis/io/v1alpha1"
	"github.com/jaegertracing/jaeger-operator/pkg/deployment"
	"github.com/sirupsen/logrus"

	"github.com/operator-framework/operator-sdk/pkg/sdk"
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
		agent.InjectSidecar(*query.Get()),
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
