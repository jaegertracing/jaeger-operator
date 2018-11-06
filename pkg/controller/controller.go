package controller

import (
	"context"
	"strings"

	"github.com/operator-framework/operator-sdk/pkg/sdk"
	"github.com/sirupsen/logrus"
	batchv1 "k8s.io/api/batch/v1"

	"github.com/jaegertracing/jaeger-operator/pkg/apis/io/v1alpha1"
)

// Controller knows what type of deployments to build based on a given spec
type Controller interface {
	Dependencies() []batchv1.Job
	Create() []sdk.Object
	Update() []sdk.Object
}

// NewController build a new controller object for the given spec
func NewController(ctx context.Context, jaeger *v1alpha1.Jaeger) Controller {
	normalize(jaeger)

	if jaeger.Spec.Strategy == "all-in-one" {
		logrus.Warnf("Strategy 'all-in-one' is no longer supported, please use 'allInOne'")
		jaeger.Spec.Strategy = "allInOne"
	}

	logrus.Debugf("Jaeger strategy: %s", jaeger.Spec.Strategy)
	if jaeger.Spec.Strategy == "allInOne" {
		return newAllInOneController(ctx, jaeger)
	}

	return newProductionController(ctx, jaeger)
}

// normalize changes the incoming Jaeger object so that the defaults are applied when
// needed and incompatible options are cleaned
func normalize(jaeger *v1alpha1.Jaeger) {
	// we need a name!
	if jaeger.Name == "" {
		logrus.Infof("This Jaeger instance was created without a name. Setting it to 'my-jaeger'")
		jaeger.Name = "my-jaeger"
	}

	// normalize the storage type
	if jaeger.Spec.Storage.Type == "" {
		logrus.Infof("Storage type wasn't provided for the Jaeger instance '%v'. Falling back to 'memory'", jaeger.Name)
		jaeger.Spec.Storage.Type = "memory"
	}

	if unknownStorage(jaeger.Spec.Storage.Type) {
		logrus.Infof(
			"The provided storage type for the Jaeger instance '%v' is unknown ('%v'). Falling back to 'memory'. Known options: %v",
			jaeger.Name,
			jaeger.Spec.Storage.Type,
			knownStorages(),
		)
		jaeger.Spec.Storage.Type = "memory"
	}

	// normalize the deployment strategy
	if strings.ToLower(jaeger.Spec.Strategy) != "production" {
		jaeger.Spec.Strategy = "allInOne"
	}

	// check for incompatible options
	// if the storage is `memory`, then the only possible strategy is `all-in-one`
	if strings.ToLower(jaeger.Spec.Storage.Type) == "memory" && strings.ToLower(jaeger.Spec.Strategy) != "allInOne" {
		logrus.Warnf(
			"No suitable storage was provided for the Jaeger instance '%v'. Falling back to all-in-one. Storage type: '%v'",
			jaeger.Name,
			jaeger.Spec.Storage.Type,
		)
		jaeger.Spec.Strategy = "allInOne"
	}
}

func unknownStorage(typ string) bool {
	for _, k := range knownStorages() {
		if strings.ToLower(typ) == k {
			return false
		}
	}

	return true
}

func knownStorages() []string {
	return []string{
		"memory",
		"kafka",
		"elasticsearch",
		"cassandra",
	}
}
