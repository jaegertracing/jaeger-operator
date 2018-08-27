package controller

import (
	"context"

	"github.com/jaegertracing/jaeger-operator/pkg/apis/io/v1alpha1"
	"github.com/jaegertracing/jaeger-operator/pkg/deployment"

	"github.com/operator-framework/operator-sdk/pkg/sdk"
	"github.com/sirupsen/logrus"
)

type allInOneController struct {
	ctx    context.Context
	jaeger *v1alpha1.Jaeger
}

func newAllInOneController(ctx context.Context, jaeger *v1alpha1.Jaeger) *allInOneController {
	return &allInOneController{
		ctx:    ctx,
		jaeger: jaeger,
	}
}

func (c allInOneController) Create() []sdk.Object {
	logrus.Debugf("Creating all-in-one for '%v'", c.jaeger.Name)

	dep := deployment.NewAllInOne(c.jaeger)
	os := []sdk.Object{dep.Get()}
	for _, svc := range dep.Services() {
		os = append(os, svc)
	}
	for _, ingress := range dep.Ingresses() {
		os = append(os, ingress)
	}

	return os
}

func (c allInOneController) Update() []sdk.Object {
	logrus.Debug("Update isn't available for all-in-one")
	return []sdk.Object{}
}
