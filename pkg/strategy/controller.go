package strategy

import (
	"context"
	"fmt"
	"strings"

	log "github.com/sirupsen/logrus"
	"github.com/spf13/viper"
	"go.opentelemetry.io/otel"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"

	v1 "github.com/jaegertracing/jaeger-operator/pkg/apis/jaegertracing/v1"
	"github.com/jaegertracing/jaeger-operator/pkg/cronjob"
	"github.com/jaegertracing/jaeger-operator/pkg/storage"
	esv1 "github.com/jaegertracing/jaeger-operator/pkg/storage/elasticsearch/v1"
)

const (
	esCertGenerationScript = "./scripts/cert_generation.sh"
)

var (
	defaultEsMemory     = resource.MustParse("16Gi")
	defaultEsCPURequest = resource.MustParse("1")
)

// For returns the appropriate Strategy for the given Jaeger instance
func For(ctx context.Context, jaeger *v1.Jaeger) S {
	tracer := otel.GetTracerProvider().Tracer(v1.ReconciliationTracer)
	ctx, span := tracer.Start(ctx, "strategy.For")
	defer span.End()

	if jaeger.Spec.Strategy == v1.DeploymentStrategyDeprecatedAllInOne {
		jaeger.Logger().Warn("Strategy 'all-in-one' is no longer supported, please use 'allInOne'")
		jaeger.Spec.Strategy = v1.DeploymentStrategyAllInOne
	}

	normalize(ctx, jaeger)

	jaeger.Logger().WithField("strategy", jaeger.Spec.Strategy).Debug("Strategy chosen")
	if jaeger.Spec.Strategy == v1.DeploymentStrategyAllInOne {
		return newAllInOneStrategy(ctx, jaeger)
	}

	if jaeger.Spec.Strategy == v1.DeploymentStrategyStreaming {
		return newStreamingStrategy(ctx, jaeger)
	}

	return newProductionStrategy(ctx, jaeger)
}

// normalize changes the incoming Jaeger object so that the defaults are applied when
// needed and incompatible options are cleaned
func normalize(ctx context.Context, jaeger *v1.Jaeger) {
	tracer := otel.GetTracerProvider().Tracer(v1.ReconciliationTracer)
	ctx, span := tracer.Start(ctx, "normalize")
	defer span.End()

	// we need a name!
	if jaeger.Name == "" {
		jaeger.Logger().Info("This Jaeger instance was created without a name. Applying a default name.")
		jaeger.Name = "my-jaeger"
	}

	// normalize the storage type
	if jaeger.Spec.Storage.Type == "" {
		jaeger.Logger().Info("Storage type not provided. Falling back to 'memory'")
		jaeger.Spec.Storage.Type = v1.JaegerMemoryStorage
	}

	if unknownStorage(jaeger.Spec.Storage.Type) {
		jaeger.Logger().WithFields(log.Fields{
			"storage":       jaeger.Spec.Storage.Type,
			"known-options": v1.ValidStorageTypes(),
		}).Info("The provided storage type is unknown. Falling back to 'memory'")
		jaeger.Spec.Storage.Type = v1.JaegerMemoryStorage
	}

	// remove reserved labels
	for _, labels := range []map[string]string{
		jaeger.Spec.JaegerCommonSpec.Labels,
		jaeger.Spec.AllInOne.JaegerCommonSpec.Labels,
		jaeger.Spec.Query.JaegerCommonSpec.Labels,
	} {
		if _, ok := labels["app.kubernetes.io/instance"]; ok {
			span.AddEvent(fmt.Sprintf("the reserved label 'app.kubernetes.io/instance' is overwritten, falling back to %s", jaeger.Name))
			delete(labels, "app.kubernetes.io/instance")
		}
		if _, ok := labels["app.kubernetes.io/managed-by"]; ok {
			span.AddEvent("the reserved label 'app.kubernetes.io/managed-by' is overwritten, falling back to jaeger-operator")
			delete(labels, "app.kubernetes.io/managed-by")
		}
	}

	// normalize the deployment strategy
	if jaeger.Spec.Strategy != v1.DeploymentStrategyProduction && jaeger.Spec.Strategy != v1.DeploymentStrategyStreaming {
		jaeger.Spec.Strategy = v1.DeploymentStrategyAllInOne
	}

	// check for incompatible options
	// if the storage is `memory`, then the only possible strategy is `all-in-one`
	if !distributedStorage(jaeger.Spec.Storage.Type) && jaeger.Spec.Strategy != v1.DeploymentStrategyAllInOne {
		jaeger.Logger().WithField("storage", jaeger.Spec.Storage.Type).Warn("No suitable storage provided. Falling back to allInOne")
		jaeger.Spec.Strategy = v1.DeploymentStrategyAllInOne
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

func distributedStorage(storage v1.JaegerStorageType) bool {
	return (storage != v1.JaegerMemoryStorage) && (storage != v1.JaegerBadgerStorage)
}

func normalizeSparkDependencies(spec *v1.JaegerStorageSpec) {
	sFlagsMap := spec.Options.Map()
	tlsEnabled := sFlagsMap["es.tls"]
	tlsSkipHost := sFlagsMap["es.tls.skip-host-verify"]
	tlsCa := sFlagsMap["es.tls.ca"]
	tlsIsNotEnabled := !strings.EqualFold(tlsEnabled, "true") &&
		!strings.EqualFold(tlsSkipHost, "true") &&
		strings.EqualFold(tlsCa, "")
	// auto enable only for supported storages
	if cronjob.SupportedStorage(spec.Type) &&
		spec.Dependencies.Enabled == nil &&
		!storage.ShouldDeployElasticsearch(*spec) &&
		tlsIsNotEnabled {
		trueVar := true
		spec.Dependencies.Enabled = &trueVar
	}
	if spec.Dependencies.Schedule == "" {
		spec.Dependencies.Schedule = "55 23 * * *"
	}
}

func normalizeIndexCleaner(spec *v1.JaegerEsIndexCleanerSpec, storage v1.JaegerStorageType) {
	// auto enable only for supported storages
	if storage == v1.JaegerESStorage && spec.Enabled == nil {
		trueVar := true
		spec.Enabled = &trueVar
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
		spec.NodeCount = 3
	}
	if spec.RedundancyPolicy == "" {
		if spec.NodeCount == 1 {
			spec.RedundancyPolicy = esv1.ZeroRedundancy
		} else {
			spec.RedundancyPolicy = esv1.SingleRedundancy
		}
	}
	if spec.Resources == nil {
		spec.Resources = &corev1.ResourceRequirements{
			Limits: corev1.ResourceList{
				corev1.ResourceMemory: defaultEsMemory,
			},
			Requests: corev1.ResourceList{
				corev1.ResourceMemory: defaultEsMemory,
				corev1.ResourceCPU:    defaultEsCPURequest,
			},
		}
	}
}

func normalizeRollover(spec *v1.JaegerEsRolloverSpec) {
	if spec.Schedule == "" {
		spec.Schedule = "0 0 * * *"
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
	enableDocumentationLink(uiOpts, spec)
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

func disableDependenciesTab(uiOpts map[string]interface{}, storage v1.JaegerStorageType, depsEnabled *bool) {
	// dependency tab is by default enabled and memory storage support it
	if (storage == v1.JaegerMemoryStorage) || (depsEnabled != nil && *depsEnabled == true) {
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

func hasDocumentationLink(menus []interface{}) (bool, int) {
	// Verify if a documentation entry exists.
	// for now the only way we have to see if a documentation link exists is comparing labels
	for menuIndex, menuItem := range menus {
		converted, ok := menuItem.(map[string]interface{})
		if !ok {
			return false, 0
		}
		if label, ok := converted["label"]; ok && label == "About" {
			items := converted["items"]
			convertedItems, ok := items.([]interface{})
			if !ok {
				return false, 0
			}
			for _, item := range convertedItems {
				itemMap, ok := item.(map[string]interface{})
				if !ok {
					return false, 0
				}
				if itemLabel, ok := itemMap["label"]; ok && itemLabel == "Documentation" {
					return true, menuIndex
				}
			}
		}
	}
	return false, 0
}

func enableDocumentationLink(uiOpts map[string]interface{}, spec *v1.JaegerSpec) {
	if !viper.IsSet("documentation-url") {
		return
	}

	if menus, ok := uiOpts["menu"]; ok {
		menuArray := menus.([]interface{})
		// If a documentation menu exists, update it, otherwise is a custom menu, don't touch it.
		if hasDocLink, menuIndex := hasDocumentationLink(menuArray); hasDocLink {
			// Update menu link
			e := map[string]interface{}{
				"label": "About",
				"items": []interface{}{map[string]interface{}{
					"label": "Documentation",
					"url":   viper.GetString("documentation-url"),
				}},
			}
			menuArray[menuIndex] = e
		}
		return
	}

	e := map[string]interface{}{
		"label": "About",
		"items": []interface{}{map[string]interface{}{
			"label": "Documentation",
			"url":   viper.GetString("documentation-url"),
		}},
	}
	uiOpts["menu"] = []interface{}{e}
}

func enableLogOut(uiOpts map[string]interface{}, spec *v1.JaegerSpec) {
	if (spec.Ingress.Enabled != nil && *spec.Ingress.Enabled == false) ||
		spec.Ingress.Security != v1.IngressSecurityOAuthProxy {
		return
	}

	if spec.Ingress.Openshift.SkipLogout != nil && *spec.Ingress.Openshift.SkipLogout == true {
		return
	}

	var menuArray []interface{}
	if m, ok := uiOpts["menu"]; ok {
		menuArray = m.([]interface{})
	}

	for _, v := range menuArray {
		converted, ok := v.(map[string]interface{})
		if !ok {
			// not a map, skip
			return
		}

		// if it has a URL entry, and if the entry contains "/oauth/sign_in", skip
		url := fmt.Sprintf("%v", converted["url"])
		// this is very naive, but will work for most cases, as that's how the OpenShift OAuth Proxy
		// build the URL. If needed, this can be a list of patterns in the future
		if strings.Contains(url, "/oauth/sign_in") {
			return
		}
	}

	logout := map[string]interface{}{
		"label":        "Log Out",
		"url":          "/oauth/sign_in",
		"anchorTarget": "_self",
	}

	uiOpts["menu"] = append(menuArray, logout)
}

func unknownStorage(typ v1.JaegerStorageType) bool {
	for _, k := range v1.ValidStorageTypes() {
		if typ == k {
			return false
		}
	}

	return true
}
