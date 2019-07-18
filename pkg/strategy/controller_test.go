package strategy

import (
	"context"
	"testing"

	"github.com/spf13/viper"
	"github.com/stretchr/testify/assert"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"

	v1 "github.com/jaegertracing/jaeger-operator/pkg/apis/jaegertracing/v1"
)

func TestNewControllerForAllInOneAsDefault(t *testing.T) {
	jaeger := v1.NewJaeger(types.NamespacedName{Name: "TestNewControllerForAllInOneAsDefault"})

	ctrl := For(context.TODO(), jaeger, []corev1.Secret{})
	assert.Equal(t, ctrl.Type(), AllInOne)
}

func TestNewControllerForAllInOneAsExplicitValue(t *testing.T) {
	jaeger := v1.NewJaeger(types.NamespacedName{Name: "TestNewControllerForAllInOneAsExplicitValue"})
	jaeger.Spec.Strategy = "ALL-IN-ONE" // same as 'all-in-one'

	ctrl := For(context.TODO(), jaeger, []corev1.Secret{})
	assert.Equal(t, ctrl.Type(), AllInOne)
}

func TestNewControllerForProduction(t *testing.T) {
	jaeger := v1.NewJaeger(types.NamespacedName{Name: "TestNewControllerForProduction"})
	jaeger.Spec.Strategy = "production"
	jaeger.Spec.Storage.Type = "elasticsearch"

	ctrl := For(context.TODO(), jaeger, []corev1.Secret{})
	assert.Equal(t, ctrl.Type(), Production)
}

func TestUnknownStorage(t *testing.T) {
	jaeger := v1.NewJaeger(types.NamespacedName{Name: "TestNewControllerForProduction"})
	jaeger.Spec.Storage.Type = "unknown"
	normalize(jaeger)
	assert.Equal(t, "memory", jaeger.Spec.Storage.Type)
}

func TestElasticsearchAsStorageOptions(t *testing.T) {
	jaeger := v1.NewJaeger(types.NamespacedName{Name: "TestElasticsearchAsStorageOptions"})
	jaeger.Spec.Strategy = "production"
	jaeger.Spec.Storage.Type = "elasticsearch"
	jaeger.Spec.Storage.Options = v1.NewOptions(map[string]interface{}{
		"es.server-urls": "http://elasticsearch-example-es-cluster:9200",
	})

	ctrl := For(context.TODO(), jaeger, []corev1.Secret{})
	deps := ctrl.Deployments()
	assert.Len(t, deps, 2) // query and collector, for a production setup
	counter := 0
	for _, dep := range deps {
		for _, arg := range dep.Spec.Template.Spec.Containers[0].Args {
			if arg == "--es.server-urls=http://elasticsearch-example-es-cluster:9200" {
				counter++
			}
		}
	}

	assert.Equal(t, len(deps), counter)
}

func TestDefaultName(t *testing.T) {
	jaeger := &v1.Jaeger{}
	normalize(jaeger)
	assert.NotEmpty(t, jaeger.Name)
}

func TestIncompatibleStorageForProduction(t *testing.T) {
	jaeger := &v1.Jaeger{
		Spec: v1.JaegerSpec{
			Strategy: "production",
			Storage: v1.JaegerStorageSpec{
				Type: "memory",
			},
		},
	}
	normalize(jaeger)
	assert.Equal(t, "allInOne", jaeger.Spec.Strategy)
}

func TestIncompatibleStorageForStreaming(t *testing.T) {
	jaeger := &v1.Jaeger{
		Spec: v1.JaegerSpec{
			Strategy: "streaming",
			Storage: v1.JaegerStorageSpec{
				Type: "memory",
			},
		},
	}
	normalize(jaeger)
	assert.Equal(t, "allInOne", jaeger.Spec.Strategy)
}

func TestDeprecatedAllInOneStrategy(t *testing.T) {
	jaeger := &v1.Jaeger{
		Spec: v1.JaegerSpec{
			Strategy: "all-in-one",
		},
	}
	For(context.TODO(), jaeger, []corev1.Secret{})
	assert.Equal(t, "allInOne", jaeger.Spec.Strategy)
}

func TestStorageMemoryOnlyUsedWithAllInOneStrategy(t *testing.T) {
	jaeger := &v1.Jaeger{
		Spec: v1.JaegerSpec{
			Strategy: "production",
			Storage: v1.JaegerStorageSpec{
				Type: "memory",
			},
		},
	}
	For(context.TODO(), jaeger, []corev1.Secret{})
	assert.Equal(t, "allInOne", jaeger.Spec.Strategy)
}

func TestSetSecurityToNoneByDefault(t *testing.T) {
	jaeger := v1.NewJaeger(types.NamespacedName{Name: "TestSetSecurityToNoneByDefault"})
	normalize(jaeger)
	assert.Equal(t, v1.IngressSecurityNoneExplicit, jaeger.Spec.Ingress.Security)
}

func TestSetSecurityToNoneWhenExplicitSettingToNone(t *testing.T) {
	jaeger := v1.NewJaeger(types.NamespacedName{Name: "TestSetSecurityToNoneWhenExplicitSettingToNone"})
	jaeger.Spec.Ingress.Security = v1.IngressSecurityNoneExplicit
	normalize(jaeger)
	assert.Equal(t, v1.IngressSecurityNoneExplicit, jaeger.Spec.Ingress.Security)
}

func TestSetSecurityToOAuthProxyByDefaultOnOpenShift(t *testing.T) {
	viper.Set("platform", "openshift")
	defer viper.Reset()

	jaeger := v1.NewJaeger(types.NamespacedName{Name: "TestSetSecurityToOAuthProxyByDefaultOnOpenShift"})
	normalize(jaeger)

	assert.Equal(t, v1.IngressSecurityOAuthProxy, jaeger.Spec.Ingress.Security)
}

func TestSetSecurityToNoneOnNonOpenShift(t *testing.T) {
	jaeger := v1.NewJaeger(types.NamespacedName{Name: "TestSetSecurityToNoneOnNonOpenShift"})
	jaeger.Spec.Ingress.Security = v1.IngressSecurityOAuthProxy

	normalize(jaeger)

	assert.Equal(t, v1.IngressSecurityNoneExplicit, jaeger.Spec.Ingress.Security)
}

func TestAcceptExplicitValueFromSecurityWhenOnOpenShift(t *testing.T) {
	viper.Set("platform", "openshift")
	defer viper.Reset()

	jaeger := v1.NewJaeger(types.NamespacedName{Name: "TestAcceptExplicitValueFromSecurityWhenOnOpenShift"})
	jaeger.Spec.Ingress.Security = v1.IngressSecurityNoneExplicit

	normalize(jaeger)

	assert.Equal(t, v1.IngressSecurityNoneExplicit, jaeger.Spec.Ingress.Security)
}

func TestNormalizeIndexCleaner(t *testing.T) {
	viper.Set("jaeger-es-index-cleaner-image", "foo")
	defer viper.Reset()
	trueVar := true
	falseVar := false
	days7 := 7
	days55 := 55
	tests := []struct {
		underTest v1.JaegerEsIndexCleanerSpec
		expected  v1.JaegerEsIndexCleanerSpec
	}{
		{underTest: v1.JaegerEsIndexCleanerSpec{},
			expected: v1.JaegerEsIndexCleanerSpec{Image: "foo", Schedule: "55 23 * * *", NumberOfDays: &days7, Enabled: &trueVar}},
		{underTest: v1.JaegerEsIndexCleanerSpec{Image: "bla", Schedule: "lol", NumberOfDays: &days55, Enabled: &falseVar},
			expected: v1.JaegerEsIndexCleanerSpec{Image: "bla", Schedule: "lol", NumberOfDays: &days55, Enabled: &falseVar}},
	}
	for _, test := range tests {
		normalizeIndexCleaner(&test.underTest, "elasticsearch")
		assert.Equal(t, test.expected, test.underTest)
	}
}

func TestNormalizeRollover(t *testing.T) {
	viper.Set("jaeger-es-rollover-image", "hoo")
	defer viper.Reset()
	tests := []struct {
		underTest v1.JaegerEsRolloverSpec
		expected  v1.JaegerEsRolloverSpec
	}{
		{underTest: v1.JaegerEsRolloverSpec{},
			expected: v1.JaegerEsRolloverSpec{Image: "hoo", Schedule: "*/30 * * * *"}},
		{underTest: v1.JaegerEsRolloverSpec{Image: "bla", Schedule: "lol"},
			expected: v1.JaegerEsRolloverSpec{Image: "bla", Schedule: "lol"}},
	}
	for _, test := range tests {
		normalizeRollover(&test.underTest)
		assert.Equal(t, test.expected, test.underTest)
	}
}

func TestNormalizeSparkDependencies(t *testing.T) {
	viper.Set("jaeger-spark-dependencies-image", "foo")
	defer viper.Reset()
	trueVar := true
	falseVar := false
	tests := []struct {
		underTest v1.JaegerStorageSpec
		expected  v1.JaegerStorageSpec
	}{
		{
			underTest: v1.JaegerStorageSpec{Type: "elasticsearch", Options: v1.NewOptions(map[string]interface{}{"es.server-urls": "foo"})},
			expected: v1.JaegerStorageSpec{Type: "elasticsearch", Options: v1.NewOptions(map[string]interface{}{"es.server-urls": "foo"}),
				Dependencies: v1.JaegerDependenciesSpec{Schedule: "55 23 * * *", Image: "foo", Enabled: &trueVar}},
		},
		{
			underTest: v1.JaegerStorageSpec{Type: "elasticsearch"},
			expected:  v1.JaegerStorageSpec{Type: "elasticsearch", Dependencies: v1.JaegerDependenciesSpec{Schedule: "55 23 * * *", Image: "foo"}},
		},
		{
			underTest: v1.JaegerStorageSpec{Type: "elasticsearch", Dependencies: v1.JaegerDependenciesSpec{Schedule: "foo", Image: "bla", Enabled: &falseVar}},
			expected:  v1.JaegerStorageSpec{Type: "elasticsearch", Dependencies: v1.JaegerDependenciesSpec{Schedule: "foo", Image: "bla", Enabled: &falseVar}},
		},
	}
	for _, test := range tests {
		normalizeSparkDependencies(&test.underTest)
		assert.Equal(t, test.expected, test.underTest)
	}
}

func TestNormalizeElasticsearch(t *testing.T) {
	tests := []struct {
		underTest v1.ElasticsearchSpec
		expected  v1.ElasticsearchSpec
	}{
		{underTest: v1.ElasticsearchSpec{},
			expected: v1.ElasticsearchSpec{NodeCount: 1, RedundancyPolicy: "ZeroRedundancy"}},
		{underTest: v1.ElasticsearchSpec{NodeCount: 3},
			expected: v1.ElasticsearchSpec{NodeCount: 3, RedundancyPolicy: "SingleRedundancy"}},
		{underTest: v1.ElasticsearchSpec{NodeCount: 3, RedundancyPolicy: "FullRedundancy"},
			expected: v1.ElasticsearchSpec{NodeCount: 3, RedundancyPolicy: "FullRedundancy"}},
		{underTest: v1.ElasticsearchSpec{Image: "bla", NodeCount: 150, RedundancyPolicy: "ZeroRedundancy"},
			expected: v1.ElasticsearchSpec{Image: "bla", NodeCount: 150, RedundancyPolicy: "ZeroRedundancy"}},
	}
	for _, test := range tests {
		normalizeElasticsearch(&test.underTest)
		assert.Equal(t, test.expected, test.underTest)
	}
}

func TestNormalizeUI(t *testing.T) {
	tests := []struct {
		j        *v1.JaegerSpec
		expected *v1.JaegerSpec
	}{
		{
			j:        &v1.JaegerSpec{},
			expected: &v1.JaegerSpec{UI: v1.JaegerUISpec{Options: v1.NewFreeForm(map[string]interface{}{"dependencies": map[string]interface{}{"menuEnabled": false}})}},
		},
		{
			j:        &v1.JaegerSpec{Storage: v1.JaegerStorageSpec{Type: "memory"}},
			expected: &v1.JaegerSpec{Storage: v1.JaegerStorageSpec{Type: "memory"}},
		},
		{
			j: &v1.JaegerSpec{Storage: v1.JaegerStorageSpec{Options: v1.NewOptions(map[string]interface{}{"es-archive.enabled": "true"})}},
			expected: &v1.JaegerSpec{Storage: v1.JaegerStorageSpec{Options: v1.NewOptions(map[string]interface{}{"es-archive.enabled": "true"})},
				UI: v1.JaegerUISpec{Options: v1.NewFreeForm(map[string]interface{}{"archiveEnabled": true, "dependencies": map[string]interface{}{"menuEnabled": false}})}},
		},
	}
	for _, test := range tests {
		normalizeUI(test.j)
		assert.Equal(t, test.expected, test.j)
	}
}

func TestNormalizeUIArchiveButton(t *testing.T) {
	tests := []struct {
		uiOpts   map[string]interface{}
		sOpts    map[string]string
		expected map[string]interface{}
	}{
		{},
		{
			uiOpts:   map[string]interface{}{},
			sOpts:    map[string]string{"es-archive.enabled": "false"},
			expected: map[string]interface{}{},
		},
		{
			uiOpts:   map[string]interface{}{},
			sOpts:    map[string]string{"es-archive.enabled": "true"},
			expected: map[string]interface{}{"archiveEnabled": true},
		},
		{
			uiOpts:   map[string]interface{}{},
			sOpts:    map[string]string{"cassandra-archive.enabled": "true"},
			expected: map[string]interface{}{"archiveEnabled": true},
		},
		{
			uiOpts:   map[string]interface{}{"archiveEnabled": "respectThis"},
			sOpts:    map[string]string{"es-archive.enabled": "true"},
			expected: map[string]interface{}{"archiveEnabled": "respectThis"},
		},
	}
	for _, test := range tests {
		enableArchiveButton(test.uiOpts, test.sOpts)
		assert.Equal(t, test.expected, test.uiOpts)
	}
}

func TestNormalizeUIDependenciesTab(t *testing.T) {
	falseVar := false
	tests := []struct {
		uiOpts   map[string]interface{}
		storage  string
		enabled  *bool
		expected map[string]interface{}
	}{
		{
			uiOpts:   map[string]interface{}{},
			storage:  "memory",
			expected: map[string]interface{}{},
		},
		{
			uiOpts:   map[string]interface{}{},
			storage:  "memory",
			enabled:  &falseVar,
			expected: map[string]interface{}{},
		},
		{
			uiOpts:   map[string]interface{}{},
			storage:  "whateverStorage",
			expected: map[string]interface{}{"dependencies": map[string]interface{}{"menuEnabled": false}},
		},
		{
			uiOpts:   map[string]interface{}{},
			storage:  "whateverStorage",
			enabled:  &falseVar,
			expected: map[string]interface{}{"dependencies": map[string]interface{}{"menuEnabled": false}},
		},
		{
			uiOpts:   map[string]interface{}{"dependencies": "respectThis"},
			storage:  "whateverStorage",
			expected: map[string]interface{}{"dependencies": "respectThis"},
		},
		{
			uiOpts:   map[string]interface{}{"dependencies": map[string]interface{}{"menuEnabled": "respectThis"}},
			storage:  "whateverStorage",
			expected: map[string]interface{}{"dependencies": map[string]interface{}{"menuEnabled": "respectThis"}},
		},
		{
			uiOpts:   map[string]interface{}{"dependencies": map[string]interface{}{"foo": "bar"}},
			storage:  "whateverStorage",
			expected: map[string]interface{}{"dependencies": map[string]interface{}{"foo": "bar", "menuEnabled": false}},
		},
	}
	for _, test := range tests {
		disableDependenciesTab(test.uiOpts, test.storage, test.enabled)
		assert.Equal(t, test.expected, test.uiOpts)
	}
}

func TestMenuWithSignOut(t *testing.T) {
	uiOpts := map[string]interface{}{}
	enableLogOut(uiOpts, &v1.JaegerSpec{Ingress: v1.JaegerIngressSpec{Security: v1.IngressSecurityOAuthProxy}})
	assert.Contains(t, uiOpts, "menu")

	expected := []interface{}{
		map[string]interface{}{
			"label": "About",
			"items": []interface{}{
				map[string]interface{}{
					"label": "Documentation",
					"url":   "https://www.jaegertracing.io/docs/latest",
				},
			},
		},
		map[string]interface{}{
			"label":        "Log Out",
			"url":          "/oauth/sign_in",
			"anchorTarget": "_self",
		},
	}
	assert.Equal(t, uiOpts["menu"], expected)
}

func TestMenuNoSignOutIngressSecurityNone(t *testing.T) {
	uiOpts := map[string]interface{}{}
	enableLogOut(uiOpts, &v1.JaegerSpec{Ingress: v1.JaegerIngressSpec{Security: v1.IngressSecurityNoneExplicit}})
	assert.NotContains(t, uiOpts, "menu")
}

func TestMenuNoSignOutExistingMenu(t *testing.T) {
	uiOpts := map[string]interface{}{
		"menu": []interface{}{},
	}
	enableLogOut(uiOpts, &v1.JaegerSpec{Ingress: v1.JaegerIngressSpec{Security: v1.IngressSecurityOAuthProxy}})
	assert.Contains(t, uiOpts, "menu")
	assert.Len(t, uiOpts["menu"], 0)
}

func assertHasAllObjects(t *testing.T, name string, s S, deployments map[string]bool, daemonsets map[string]bool, services map[string]bool, ingresses map[string]bool, routes map[string]bool, serviceAccounts map[string]bool, configMaps map[string]bool) {
	for _, o := range s.Deployments() {
		deployments[o.Name] = true
	}

	for _, o := range s.DaemonSets() {
		daemonsets[o.Name] = true
	}

	for _, o := range s.Services() {
		services[o.Name] = true
	}

	for _, o := range s.Ingresses() {
		ingresses[o.Name] = true
	}

	for _, o := range s.Routes() {
		routes[o.Name] = true
	}

	for _, o := range s.Accounts() {
		serviceAccounts[o.Name] = true
	}

	for _, o := range s.ConfigMaps() {
		configMaps[o.Name] = true
	}

	for k, v := range deployments {
		assert.True(t, v, "Expected %s to have been returned from the list of deployments", k)
	}

	for k, v := range daemonsets {
		assert.True(t, v, "Expected %s to have been returned from the list of daemonsets", k)
	}

	for k, v := range services {
		assert.True(t, v, "Expected %s to have been returned from the list of services", k)
	}

	for k, v := range ingresses {
		assert.True(t, v, "Expected %s to have been returned from the list of ingress rules", k)
	}

	for k, v := range routes {
		assert.True(t, v, "Expected %s to have been returned from the list of routes", k)
	}

	for k, v := range serviceAccounts {
		assert.True(t, v, "Expected %s to have been returned from the list of service accounts", k)
	}

	for k, v := range configMaps {
		assert.True(t, v, "Expected %s to have been returned from the list of config maps", k)
	}
}
