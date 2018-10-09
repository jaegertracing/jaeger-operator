package controller

import (
	"context"

	"github.com/operator-framework/operator-sdk/pkg/sdk"
	"github.com/sirupsen/logrus"

	"github.com/jaegertracing/jaeger-operator/pkg/apis/io/v1alpha1"
)

// Controller knows what type of deployments to build based on a given spec
type Controller interface {
	Create() []sdk.Object
	Update() []sdk.Object
}

// NewController build a new controller object for the given spec
func NewController(ctx context.Context, jaeger *v1alpha1.Jaeger) Controller {
	logrus.Debugf("Jaeger strategy: %s", jaeger.Spec.Strategy)
	if jaeger.Spec.Strategy == "all-in-one" {
		return newAllInOneController(ctx, jaeger)
	}

	return newProductionController(ctx, jaeger)
}
