package controller

import (
	"context"
	"strings"

	"github.com/operator-framework/operator-sdk/pkg/sdk"
	"github.com/sirupsen/logrus"
	"github.com/spf13/viper"
	batchv1 "k8s.io/api/batch/v1"

	"github.com/jaegertracing/jaeger-operator/pkg/apis/io/v1alpha1"
	"github.com/jaegertracing/jaeger-operator/pkg/storage"
)

// Controller knows what type of deployments to build based on a given spec
type Controller interface {
	Dependencies() []batchv1.Job
	Create() []sdk.Object
	Update() []sdk.Object
}

// NewController build a new controller object for the given spec
func NewController(ctx context.Context, jaeger *v1alpha1.Jaeger) Controller {
	if strings.ToLower(jaeger.Spec.Strategy) == "all-in-one" {
		logrus.Warnf("Strategy 'all-in-one' is no longer supported, please use 'allInOne'")
		jaeger.Spec.Strategy = "allInOne"
	}

	normalize(jaeger)

	logrus.Debugf("Jaeger strategy: %s", jaeger.Spec.Strategy)
	if strings.ToLower(jaeger.Spec.Strategy) == "allinone" {
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
			storage.ValidTypes(),
		)
		jaeger.Spec.Storage.Type = "memory"
	}

	// normalize the deployment strategy
	if strings.ToLower(jaeger.Spec.Strategy) != "production" {
		jaeger.Spec.Strategy = "allInOne"
	}

	// check for incompatible options
	// if the storage is `memory`, then the only possible strategy is `all-in-one`
	if strings.ToLower(jaeger.Spec.Storage.Type) == "memory" && strings.ToLower(jaeger.Spec.Strategy) != "allinone" {
		logrus.Warnf(
			"No suitable storage was provided for the Jaeger instance '%v'. Falling back to all-in-one. Storage type: '%v'",
			jaeger.Name,
			jaeger.Spec.Storage.Type,
		)
		jaeger.Spec.Strategy = "allInOne"
	}

	if viper.GetString("platform") != v1alpha1.FlagPlatformOpenShift {
		// for now, we only know about the OpenShift OAuth Proxy, so, disable it if
		// the platform isn't OpenShift
		b := false
		jaeger.Spec.Ingress.OAuthProxy = &b
	} else if jaeger.Spec.Ingress.OAuthProxy == nil {
		// we are on openshift, so, set it to true if omitted
		b := true
		jaeger.Spec.Ingress.OAuthProxy = &b
	}
}

func unknownStorage(typ string) bool {
	for _, k := range storage.ValidTypes() {
		if strings.ToLower(typ) == k {
			return false
		}
	}

	return true
}
