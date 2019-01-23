package strategy

import (
	"context"
	"fmt"
	"strings"

	"github.com/sirupsen/logrus"
	"github.com/spf13/viper"
	batchv1 "k8s.io/api/batch/v1"
	"k8s.io/apimachinery/pkg/runtime"

	"github.com/jaegertracing/jaeger-operator/pkg/apis/io/v1alpha1"
	"github.com/jaegertracing/jaeger-operator/pkg/cronjob"
	"github.com/jaegertracing/jaeger-operator/pkg/storage"
)

// S knows what type of deployments to build based on a given spec
type S interface {
	Dependencies() []batchv1.Job
	Create() []runtime.Object
	Update() []runtime.Object
}

// For returns the appropriate Strategy for the given Jaeger instance
func For(ctx context.Context, jaeger *v1alpha1.Jaeger) S {
	if strings.ToLower(jaeger.Spec.Strategy) == "all-in-one" {
		logrus.Warnf("Strategy 'all-in-one' is no longer supported, please use 'allInOne'")
		jaeger.Spec.Strategy = "allInOne"
	}

	normalize(jaeger)

	logrus.Debugf("Jaeger strategy: %s", jaeger.Spec.Strategy)
	if strings.ToLower(jaeger.Spec.Strategy) == "allinone" {
		return newAllInOneStrategy(ctx, jaeger)
	}

	return newProductionStrategy(ctx, jaeger)
}

// normalize changes the incoming Jaeger object so that the defaults are applied when
// needed and incompatible options are cleaned
func normalize(jaeger *v1alpha1.Jaeger) {
	// we need a name!
	if jaeger.Name == "" {
		logrus.Infof("This Jaeger instance was created without a name. Setting it to 'my-jaeger'")
		jaeger.Name = "my-jaeger"
	}

	// normalize the version
	if jaeger.Spec.Version == "" {
		jaegerVersion := viper.GetString("jaeger-version")
		logrus.Infof("Version wasn't provided for the Jaeger instance '%v'. Falling back to '%v'", jaeger.Name, jaegerVersion)
		jaeger.Spec.Version = jaegerVersion
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

	// we always set the value to None, except when we are on OpenShift *and* the user has not explicitly set to 'none'
	if viper.GetString("platform") == v1alpha1.FlagPlatformOpenShift && jaeger.Spec.Ingress.Security != v1alpha1.IngressSecurityNoneExplicit {
		jaeger.Spec.Ingress.Security = v1alpha1.IngressSecurityOAuthProxy
	} else {
		// cases:
		// - omitted on Kubernetes
		// - 'none' on any platform
		jaeger.Spec.Ingress.Security = v1alpha1.IngressSecurityNone
	}

	normalizeSparkDependencies(&jaeger.Spec.Storage.SparkDependencies, jaeger.Spec.Storage.Type)
	normalizeIndexCleaner(&jaeger.Spec.Storage.EsIndexCleaner, jaeger.Spec.Storage.Type)
}

func normalizeSparkDependencies(spec *v1alpha1.JaegerDependenciesSpec, storage string) {
	// auto enable only for supported storages
	if cronjob.SupportedStorage(storage) && spec.Enabled == nil {
		trueVar := true
		spec.Enabled = &trueVar
	}
	if spec.Image == "" {
		spec.Image = fmt.Sprintf("%s", viper.GetString("jaeger-spark-dependencies-image"))
	}
	if spec.Schedule == "" {
		spec.Schedule = "55 23 * * *"
	}
}

func normalizeIndexCleaner(spec *v1alpha1.JaegerEsIndexCleanerSpec, storage string) {
	// auto enable only for supported storages
	if storage == "elasticsearch" && spec.Enabled == nil {
		trueVar := true
		spec.Enabled = &trueVar
	}
	if spec.Image == "" {
		spec.Image = fmt.Sprintf("%s", viper.GetString("jaeger-es-index-cleaner-image"))
	}
	if spec.Schedule == "" {
		spec.Schedule = "55 23 * * *"
	}
	if spec.NumberOfDays == 0 {
		spec.NumberOfDays = 7
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
