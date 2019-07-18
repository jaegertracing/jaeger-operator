package strategy

import (
	"context"
	"encoding/json"
	"strings"

	log "github.com/sirupsen/logrus"
	"github.com/spf13/viper"
	corev1 "k8s.io/api/core/v1"

	v1 "github.com/jaegertracing/jaeger-operator/pkg/apis/jaegertracing/v1"
	"github.com/jaegertracing/jaeger-operator/pkg/cronjob"
	"github.com/jaegertracing/jaeger-operator/pkg/storage"
	esv1 "github.com/jaegertracing/jaeger-operator/pkg/storage/elasticsearch/v1"
)

const (
	esCertGenerationScript = "./scripts/cert_generation.sh"
)

// For returns the appropriate Strategy for the given Jaeger instance
func For(ctx context.Context, jaeger *v1.Jaeger, secrets []corev1.Secret) S {
	if strings.EqualFold(jaeger.Spec.Strategy, "all-in-one") {
		jaeger.Logger().Warn("Strategy 'all-in-one' is no longer supported, please use 'allInOne'")
		jaeger.Spec.Strategy = "allInOne"
	}

	normalize(jaeger)

	jaeger.Logger().WithField("strategy", jaeger.Spec.Strategy).Debug("Strategy chosen")
	if strings.EqualFold(jaeger.Spec.Strategy, "allinone") {
		return newAllInOneStrategy(jaeger)
	}

	if strings.EqualFold(jaeger.Spec.Strategy, "streaming") {
		return newStreamingStrategy(jaeger)
	}

	es := &storage.ElasticsearchDeployment{Jaeger: jaeger, CertScript: esCertGenerationScript, Secrets: secrets}
	return newProductionStrategy(jaeger, es)
}

// normalize changes the incoming Jaeger object so that the defaults are applied when
// needed and incompatible options are cleaned
func normalize(jaeger *v1.Jaeger) {
	// we need a name!
	if jaeger.Name == "" {
		jaeger.Logger().Info("This Jaeger instance was created without a name. Applying a default name.")
		jaeger.Name = "my-jaeger"
	}

	// normalize the storage type
	if jaeger.Spec.Storage.Type == "" {
		jaeger.Logger().Info("Storage type not provided. Falling back to 'memory'")
		jaeger.Spec.Storage.Type = "memory"
	}

	if unknownStorage(jaeger.Spec.Storage.Type) {
		jaeger.Logger().WithFields(log.Fields{
			"storage":       jaeger.Spec.Storage.Type,
			"known-options": storage.ValidTypes(),
		}).Info("The provided storage type is unknown. Falling back to 'memory'")
		jaeger.Spec.Storage.Type = "memory"
	}

	// normalize the deployment strategy
	if !strings.EqualFold(jaeger.Spec.Strategy, "production") && !strings.EqualFold(jaeger.Spec.Strategy, "streaming") {
		jaeger.Spec.Strategy = "allInOne"
	}

	// check for incompatible options
	// if the storage is `memory`, then the only possible strategy is `all-in-one`
	if strings.EqualFold(jaeger.Spec.Storage.Type, "memory") && !strings.EqualFold(jaeger.Spec.Strategy, "allinone") {
		jaeger.Logger().WithField("storage", jaeger.Spec.Storage.Type).Warn("No suitable storage provided. Falling back to all-in-one")
		jaeger.Spec.Strategy = "allInOne"
	}

	// we always set the value to None, except when we are on OpenShift *and* the user has not explicitly set to 'none'
	if viper.GetString("platform") == v1.FlagPlatformOpenShift && jaeger.Spec.Ingress.Security != v1.IngressSecurityNoneExplicit {
		jaeger.Spec.Ingress.Security = v1.IngressSecurityOAuthProxy
	} else {
		// cases:
		// - omitted on Kubernetes
		// - 'none' on any platform
		jaeger.Spec.Ingress.Security = v1.IngressSecurityNoneExplicit
	}

	// note that the order normalization matters - UI norm expects all normalized properties
	normalizeSparkDependencies(&jaeger.Spec.Storage)
	normalizeIndexCleaner(&jaeger.Spec.Storage.EsIndexCleaner, jaeger.Spec.Storage.Type)
	normalizeElasticsearch(&jaeger.Spec.Storage.Elasticsearch)
	normalizeRollover(&jaeger.Spec.Storage.EsRollover)
	normalizeUI(&jaeger.Spec)
}

func normalizeSparkDependencies(spec *v1.JaegerStorageSpec) {
	// auto enable only for supported storages
	if cronjob.SupportedStorage(spec.Type) &&
		spec.Dependencies.Enabled == nil &&
		!storage.ShouldDeployElasticsearch(*spec) {
		trueVar := true
		spec.Dependencies.Enabled = &trueVar
	}
	if spec.Dependencies.Image == "" {
		spec.Dependencies.Image = viper.GetString("jaeger-spark-dependencies-image")
	}
	if spec.Dependencies.Schedule == "" {
		spec.Dependencies.Schedule = "55 23 * * *"
	}
}

func normalizeIndexCleaner(spec *v1.JaegerEsIndexCleanerSpec, storage string) {
	// auto enable only for supported storages
	if storage == "elasticsearch" && spec.Enabled == nil {
		trueVar := true
		spec.Enabled = &trueVar
	}
	if spec.Image == "" {
		spec.Image = viper.GetString("jaeger-es-index-cleaner-image")
	}
	if spec.Schedule == "" {
		spec.Schedule = "55 23 * * *"
	}
	if spec.NumberOfDays == nil {
		defDays := 7
		spec.NumberOfDays = &defDays
	}
}

func normalizeElasticsearch(spec *v1.ElasticsearchSpec) {
	if spec.NodeCount == 0 {
		spec.NodeCount = 1
	}
	if spec.RedundancyPolicy == "" {
		if spec.NodeCount == 1 {
			spec.RedundancyPolicy = esv1.ZeroRedundancy
		} else {
			spec.RedundancyPolicy = esv1.SingleRedundancy
		}
	}
}

func normalizeRollover(spec *v1.JaegerEsRolloverSpec) {
	if spec.Image == "" {
		spec.Image = viper.GetString("jaeger-es-rollover-image")
	}
	if spec.Schedule == "" {
		spec.Schedule = "*/30 * * * *"
	}
}

func normalizeUI(spec *v1.JaegerSpec) {
	uiOpts := map[string]interface{}{}
	if !spec.UI.Options.IsEmpty() {
		if m, err := spec.UI.Options.GetMap(); err == nil {
			uiOpts = m
		}
	}
	enableArchiveButton(uiOpts, spec.Storage.Options.Map())
	disableDependenciesTab(uiOpts, spec.Storage.Type, spec.Storage.Dependencies.Enabled)
	enableLogOut(uiOpts, spec)
	if len(uiOpts) > 0 {
		spec.UI.Options = v1.NewFreeForm(uiOpts)
	}
}

func enableArchiveButton(uiOpts map[string]interface{}, sOpts map[string]string) {
	// respect explicit settings
	if _, ok := uiOpts["archiveEnabled"]; !ok {
		// archive tab is by default disabled
		if strings.EqualFold(sOpts["es-archive.enabled"], "true") ||
			strings.EqualFold(sOpts["cassandra-archive.enabled"], "true") {
			uiOpts["archiveEnabled"] = true
		}
	}
}

func disableDependenciesTab(uiOpts map[string]interface{}, storage string, depsEnabled *bool) {
	// dependency tab is by default enabled and memory storage support it
	if strings.EqualFold(storage, "memory") || (depsEnabled != nil && *depsEnabled == true) {
		return
	}
	deps := map[string]interface{}{}
	if val, ok := uiOpts["dependencies"]; ok {
		if val, ok := val.(map[string]interface{}); ok {
			deps = val
		} else {
			// we return as the type does not match
			return
		}
	}
	// respect explicit settings
	if _, ok := deps["menuEnabled"]; !ok {
		deps["menuEnabled"] = false
		uiOpts["dependencies"] = deps
	}
}

func enableLogOut(uiOpts map[string]interface{}, spec *v1.JaegerSpec) {
	if (spec.Ingress.Enabled != nil && *spec.Ingress.Enabled == false) ||
		spec.Ingress.Security != v1.IngressSecurityOAuthProxy {
		return
	}

	if _, ok := uiOpts["menu"]; ok {
		return
	}

	menuStr := `[
		{
		  "label": "About",
		  "items": [
			{
			  "label": "Documentation",
			  "url": "https://www.jaegertracing.io/docs/latest"
			}
		  ]
		},
		{
		  "label": "Log Out",
		  "url": "/oauth/sign_in",
		  "anchorTarget": "_self"
		}
	  ]`

	menuArray := make([]interface{}, 2)

	json.Unmarshal([]byte(menuStr), &menuArray)

	uiOpts["menu"] = menuArray
}

func unknownStorage(typ string) bool {
	for _, k := range storage.ValidTypes() {
		if strings.EqualFold(typ, k) {
			return false
		}
	}

	return true
}
