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
	jaeger := v1.NewJaeger(types.NamespacedName{Name: "my-instance"})

	ctrl := For(context.TODO(), jaeger)
	assert.Equal(t, ctrl.Type(), v1.DeploymentStrategyAllInOne)
}

func TestNewControllerForAllInOneAsExplicitValue(t *testing.T) {
	jaeger := v1.NewJaeger(types.NamespacedName{Name: "my-instance"})
	jaeger.Spec.Strategy = v1.DeploymentStrategyDeprecatedAllInOne // same as 'all-in-one'

	ctrl := For(context.TODO(), jaeger)
	assert.Equal(t, ctrl.Type(), v1.DeploymentStrategyAllInOne)
}

func TestNewControllerForProduction(t *testing.T) {
	jaeger := v1.NewJaeger(types.NamespacedName{Name: "my-instance"})
	jaeger.Spec.Strategy = v1.DeploymentStrategyProduction
	jaeger.Spec.Storage.Type = v1.JaegerESStorage

	ctrl := For(context.TODO(), jaeger)
	assert.Equal(t, ctrl.Type(), v1.DeploymentStrategyProduction)
}

func TestUnknownStorage(t *testing.T) {
	jaeger := v1.NewJaeger(types.NamespacedName{Name: "my-instance"})
	jaeger.Spec.Storage.Type = "unknown"
	normalize(context.Background(), jaeger)
	assert.Equal(t, v1.JaegerMemoryStorage, jaeger.Spec.Storage.Type)
}

func TestElasticsearchAsStorageOptions(t *testing.T) {
	jaeger := v1.NewJaeger(types.NamespacedName{Name: "my-instance"})
	jaeger.Spec.Strategy = v1.DeploymentStrategyProduction
	jaeger.Spec.Storage.Type = v1.JaegerESStorage
	jaeger.Spec.Storage.Options = v1.NewOptions(map[string]interface{}{
		"es.server-urls": "http://elasticsearch-example-es-cluster:9200",
	})

	ctrl := For(context.TODO(), jaeger)
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
	normalize(context.Background(), jaeger)
	assert.NotEmpty(t, jaeger.Name)
}

func TestIncompatibleMemoryStorageForProduction(t *testing.T) {
	jaeger := &v1.Jaeger{
		Spec: v1.JaegerSpec{
			Strategy: v1.DeploymentStrategyProduction,
			Storage: v1.JaegerStorageSpec{
				Type: v1.JaegerMemoryStorage,
			},
		},
	}
	normalize(context.Background(), jaeger)
	assert.Equal(t, v1.DeploymentStrategyAllInOne, jaeger.Spec.Strategy)
}

func TestIncompatibleBadgerStorageForProduction(t *testing.T) {
	jaeger := &v1.Jaeger{
		Spec: v1.JaegerSpec{
			Strategy: v1.DeploymentStrategyProduction,
			Storage: v1.JaegerStorageSpec{
				Type: v1.JaegerBadgerStorage,
			},
		},
	}
	normalize(context.Background(), jaeger)
	assert.Equal(t, v1.DeploymentStrategyAllInOne, jaeger.Spec.Strategy)
}

func TestIncompatibleStorageForStreaming(t *testing.T) {
	jaeger := &v1.Jaeger{
		Spec: v1.JaegerSpec{
			Strategy: v1.DeploymentStrategyStreaming,
			Storage: v1.JaegerStorageSpec{
				Type: v1.JaegerMemoryStorage,
			},
		},
	}
	normalize(context.Background(), jaeger)
	assert.Equal(t, v1.DeploymentStrategyAllInOne, jaeger.Spec.Strategy)
}

func TestDeprecatedAllInOneStrategy(t *testing.T) {
	jaeger := &v1.Jaeger{
		Spec: v1.JaegerSpec{
			Strategy: v1.DeploymentStrategyDeprecatedAllInOne,
		},
	}
	For(context.TODO(), jaeger)
	assert.Equal(t, v1.DeploymentStrategyAllInOne, jaeger.Spec.Strategy)
}

func TestStorageMemoryOnlyUsedWithAllInOneStrategy(t *testing.T) {
	jaeger := &v1.Jaeger{
		Spec: v1.JaegerSpec{
			Strategy: v1.DeploymentStrategyProduction,
			Storage: v1.JaegerStorageSpec{
				Type: v1.JaegerMemoryStorage,
			},
		},
	}
	For(context.TODO(), jaeger)
	assert.Equal(t, v1.DeploymentStrategyAllInOne, jaeger.Spec.Strategy)
}

func TestSetSecurityToNoneByDefault(t *testing.T) {
	jaeger := v1.NewJaeger(types.NamespacedName{Name: "my-instance"})
	normalize(context.Background(), jaeger)
	assert.Equal(t, v1.IngressSecurityNoneExplicit, jaeger.Spec.Ingress.Security)
}

func TestSetSecurityToNoneWhenExplicitSettingToNone(t *testing.T) {
	jaeger := v1.NewJaeger(types.NamespacedName{Name: "my-instance"})
	jaeger.Spec.Ingress.Security = v1.IngressSecurityNoneExplicit
	normalize(context.Background(), jaeger)
	assert.Equal(t, v1.IngressSecurityNoneExplicit, jaeger.Spec.Ingress.Security)
}

func TestSetSecurityToOAuthProxyByDefaultOnOpenShift(t *testing.T) {
	viper.Set("platform", "openshift")
	defer viper.Reset()

	jaeger := v1.NewJaeger(types.NamespacedName{Name: "my-instance"})
	normalize(context.Background(), jaeger)

	assert.Equal(t, v1.IngressSecurityOAuthProxy, jaeger.Spec.Ingress.Security)
}

func TestSetSecurityToNoneOnNonOpenShift(t *testing.T) {
	jaeger := v1.NewJaeger(types.NamespacedName{Name: "my-instance"})
	jaeger.Spec.Ingress.Security = v1.IngressSecurityOAuthProxy

	normalize(context.Background(), jaeger)

	assert.Equal(t, v1.IngressSecurityNoneExplicit, jaeger.Spec.Ingress.Security)
}

func TestAcceptExplicitValueFromSecurityWhenOnOpenShift(t *testing.T) {
	viper.Set("platform", "openshift")
	defer viper.Reset()

	jaeger := v1.NewJaeger(types.NamespacedName{Name: "my-instance"})
	jaeger.Spec.Ingress.Security = v1.IngressSecurityNoneExplicit

	normalize(context.Background(), jaeger)

	assert.Equal(t, v1.IngressSecurityNoneExplicit, jaeger.Spec.Ingress.Security)
}

func TestRemoveReservedLabels(t *testing.T) {
	jaeger := v1.NewJaeger(types.NamespacedName{Name: "TestRemoveReservedLabels"})
	jaeger.Spec.Labels = map[string]string{
		"app.kubernetes.io/instance":   "custom-instance",
		"app.kubernetes.io/managed-by": "custom-managed-by",
	}

	normalize(context.Background(), jaeger)

	assert.NotContains(t, jaeger.Spec.Labels, "app.kubernetes.io/instance")
	assert.NotContains(t, jaeger.Spec.Labels, "app.kubernetes.io/managed-by")
}

func TestNormalizeIndexCleaner(t *testing.T) {
	trueVar := true
	falseVar := false
	days7 := 7
	days55 := 55
	tests := []struct {
		underTest v1.JaegerEsIndexCleanerSpec
		expected  v1.JaegerEsIndexCleanerSpec
	}{
		{underTest: v1.JaegerEsIndexCleanerSpec{},
			expected: v1.JaegerEsIndexCleanerSpec{Schedule: "55 23 * * *", NumberOfDays: &days7, Enabled: &trueVar}},
		{underTest: v1.JaegerEsIndexCleanerSpec{Image: "bla", Schedule: "lol", NumberOfDays: &days55, Enabled: &falseVar},
			expected: v1.JaegerEsIndexCleanerSpec{Image: "bla", Schedule: "lol", NumberOfDays: &days55, Enabled: &falseVar}},
	}
	for _, test := range tests {
		normalizeIndexCleaner(&test.underTest, v1.JaegerESStorage)
		assert.Equal(t, test.expected, test.underTest)
	}
}

func TestNormalizeRollover(t *testing.T) {
	tests := []struct {
		underTest v1.JaegerEsRolloverSpec
		expected  v1.JaegerEsRolloverSpec
	}{
		{underTest: v1.JaegerEsRolloverSpec{},
			expected: v1.JaegerEsRolloverSpec{Schedule: "0 0 * * *"}},
		{underTest: v1.JaegerEsRolloverSpec{Image: "bla", Schedule: "lol"},
			expected: v1.JaegerEsRolloverSpec{Image: "bla", Schedule: "lol"}},
	}
	for _, test := range tests {
		normalizeRollover(&test.underTest)
		assert.Equal(t, test.expected, test.underTest)
	}
}

func TestNormalizeSparkDependencies(t *testing.T) {
	trueVar := true
	falseVar := false
	tests := []struct {
		underTest v1.JaegerStorageSpec
		expected  v1.JaegerStorageSpec
	}{
		{
			underTest: v1.JaegerStorageSpec{Type: v1.JaegerESStorage, Options: v1.NewOptions(map[string]interface{}{"es.server-urls": "foo"})},
			expected: v1.JaegerStorageSpec{Type: v1.JaegerESStorage, Options: v1.NewOptions(map[string]interface{}{"es.server-urls": "foo"}),
				Dependencies: v1.JaegerDependenciesSpec{Schedule: "55 23 * * *", Enabled: &trueVar}},
		},
		{
			underTest: v1.JaegerStorageSpec{Type: v1.JaegerESStorage},
			expected:  v1.JaegerStorageSpec{Type: v1.JaegerESStorage, Dependencies: v1.JaegerDependenciesSpec{Schedule: "55 23 * * *"}},
		},
		{
			underTest: v1.JaegerStorageSpec{Type: v1.JaegerESStorage, Dependencies: v1.JaegerDependenciesSpec{},
				Options: v1.NewOptions(map[string]interface{}{"es.server-urls": "local", "es.tls": true})},
			expected: v1.JaegerStorageSpec{Type: v1.JaegerESStorage, Dependencies: v1.JaegerDependenciesSpec{Schedule: "55 23 * * *", Enabled: nil},
				Options: v1.NewOptions(map[string]interface{}{"es.server-urls": "local", "es.tls": true}),
			},
		},
		{
			underTest: v1.JaegerStorageSpec{Type: v1.JaegerESStorage, Dependencies: v1.JaegerDependenciesSpec{},
				Options: v1.NewOptions(map[string]interface{}{"es.server-urls": "local", "es.skip-host-verify": false})},
			expected: v1.JaegerStorageSpec{Type: v1.JaegerESStorage, Dependencies: v1.JaegerDependenciesSpec{Schedule: "55 23 * * *", Enabled: &trueVar},
				Options: v1.NewOptions(map[string]interface{}{"es.server-urls": "local", "es.skip-host-verify": false}),
			},
		},
		{
			underTest: v1.JaegerStorageSpec{Type: v1.JaegerESStorage, Dependencies: v1.JaegerDependenciesSpec{},
				Options: v1.NewOptions(map[string]interface{}{"es.server-urls": "local", "es.skip-host-verify": false, "es.tls.ca": "rr"})},
			expected: v1.JaegerStorageSpec{Type: v1.JaegerESStorage, Dependencies: v1.JaegerDependenciesSpec{Schedule: "55 23 * * *", Enabled: nil},
				Options: v1.NewOptions(map[string]interface{}{"es.server-urls": "local", "es.skip-host-verify": false, "es.tls.ca": "rr"}),
			},
		},
		{
			underTest: v1.JaegerStorageSpec{Type: v1.JaegerESStorage, Dependencies: v1.JaegerDependenciesSpec{Schedule: "foo", Image: "bla", Enabled: &falseVar}},
			expected:  v1.JaegerStorageSpec{Type: v1.JaegerESStorage, Dependencies: v1.JaegerDependenciesSpec{Schedule: "foo", Image: "bla", Enabled: &falseVar}},
		},
	}
	for _, test := range tests {
		normalizeSparkDependencies(&test.underTest)
		assert.Equal(t, test.expected, test.underTest)
	}
}

func TestNormalizeElasticsearch(t *testing.T) {
	defResources := &corev1.ResourceRequirements{
		Limits:   corev1.ResourceList{corev1.ResourceMemory: defaultEsMemory},
		Requests: corev1.ResourceList{corev1.ResourceMemory: defaultEsMemory, corev1.ResourceCPU: defaultEsCPURequest},
	}
	tests := []struct {
		underTest v1.ElasticsearchSpec
		expected  v1.ElasticsearchSpec
	}{
		{underTest: v1.ElasticsearchSpec{},
			expected: v1.ElasticsearchSpec{NodeCount: 3, RedundancyPolicy: "SingleRedundancy", Resources: defResources},
		},
		{underTest: v1.ElasticsearchSpec{NodeCount: 1},
			expected: v1.ElasticsearchSpec{NodeCount: 1, RedundancyPolicy: "ZeroRedundancy", Resources: defResources}},
		{underTest: v1.ElasticsearchSpec{NodeCount: 3, RedundancyPolicy: "FullRedundancy"},
			expected: v1.ElasticsearchSpec{NodeCount: 3, RedundancyPolicy: "FullRedundancy", Resources: defResources}},
		{underTest: v1.ElasticsearchSpec{Image: "bla", NodeCount: 150, RedundancyPolicy: "ZeroRedundancy", Resources: &corev1.ResourceRequirements{}},
			expected: v1.ElasticsearchSpec{Image: "bla", NodeCount: 150, RedundancyPolicy: "ZeroRedundancy", Resources: &corev1.ResourceRequirements{}}},
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
			j:        &v1.JaegerSpec{Storage: v1.JaegerStorageSpec{Type: v1.JaegerMemoryStorage}},
			expected: &v1.JaegerSpec{Storage: v1.JaegerStorageSpec{Type: v1.JaegerMemoryStorage}},
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
		storage  v1.JaegerStorageType
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

func TestMenuWithLogOut(t *testing.T) {
	spec := &v1.JaegerSpec{Ingress: v1.JaegerIngressSpec{Security: v1.IngressSecurityOAuthProxy}}
	uiOpts := map[string]interface{}{}
	enableLogOut(uiOpts, spec)
	assert.Contains(t, uiOpts, "menu")

	expected := []interface{}{
		map[string]interface{}{
			"label":        "Log Out",
			"url":          "/oauth/sign_in",
			"anchorTarget": "_self",
		},
	}
	assert.Equal(t, expected, uiOpts["menu"])
}

func TestMenuWithCustomDocURL(t *testing.T) {
	docURL := "http://test/doc/url"

	viper.Set("documentation-url", docURL)
	defer viper.Reset()

	uiOpts := map[string]interface{}{}
	spec := &v1.JaegerSpec{Ingress: v1.JaegerIngressSpec{Security: v1.IngressSecurityOAuthProxy}}
	enableDocumentationLink(uiOpts, spec)
	assert.Contains(t, uiOpts, "menu")

	expected := []interface{}{
		map[string]interface{}{
			"label": "About",
			"items": []interface{}{
				map[string]interface{}{
					"label": "Documentation",
					"url":   docURL,
				},
			},
		},
	}
	assert.Equal(t, expected, uiOpts["menu"])
}

func TestUpdateMenuDocURL(t *testing.T) {
	docURLv1 := "http://testv1/doc/url"
	docURLv2 := "http://testv2/doc/url"

	viper.Set("documentation-url", docURLv1)
	defer viper.Reset()

	uiOpts := map[string]interface{}{}

	spec := &v1.JaegerSpec{Ingress: v1.JaegerIngressSpec{Security: v1.IngressSecurityOAuthProxy}}
	enableDocumentationLink(uiOpts, spec)
	assert.Contains(t, uiOpts, "menu")

	expected := []interface{}{
		map[string]interface{}{
			"label": "About",
			"items": []interface{}{
				map[string]interface{}{
					"label": "Documentation",
					"url":   docURLv1,
				},
			},
		},
	}
	assert.Equal(t, expected, uiOpts["menu"])

	viper.Set("documentation-url", docURLv2)
	enableDocumentationLink(uiOpts, spec)
	assert.Contains(t, uiOpts, "menu")

	expected = []interface{}{
		map[string]interface{}{
			"label": "About",
			"items": []interface{}{
				map[string]interface{}{
					"label": "Documentation",
					"url":   docURLv2,
				},
			},
		},
	}
	assert.Equal(t, expected, uiOpts["menu"])

}

func TestNoDocWithCustomMenu(t *testing.T) {
	viper.Set("documentation-url", "http://testv1/doc/url")
	defer viper.Reset()

	internalLink := map[string]interface{}{
		"label": "Some internal links",
		"items": []interface{}{
			map[string]interface{}{
				"label": "The internal link",
				"url":   "http://example.com/internal",
			},
		},
	}
	uiOpts := map[string]interface{}{
		"menu": []interface{}{internalLink},
	}

	spec := &v1.JaegerSpec{Ingress: v1.JaegerIngressSpec{Security: v1.IngressSecurityOAuthProxy}}
	enableDocumentationLink(uiOpts, spec)
	assert.Equal(t, uiOpts, uiOpts)

}

func TestMenuNoLogOutIngressSecurityNone(t *testing.T) {
	uiOpts := map[string]interface{}{}
	spec := &v1.JaegerSpec{Ingress: v1.JaegerIngressSpec{Security: v1.IngressSecurityNoneExplicit}}
	enableLogOut(uiOpts, spec)
	assert.NotContains(t, uiOpts, "menu")
}

func TestMenuNoLogOutExistingMenuWithSkipOption(t *testing.T) {
	// prepare
	internalLink := map[string]interface{}{
		"label": "Some internal links",
		"items": []interface{}{
			map[string]interface{}{
				"label": "The internal link",
				"url":   "http://example.com/internal",
			},
		},
	}
	uiOpts := map[string]interface{}{
		"menu": []interface{}{internalLink},
	}
	trueVar := true
	spec := &v1.JaegerSpec{
		Ingress: v1.JaegerIngressSpec{
			Security: v1.IngressSecurityOAuthProxy,
			Openshift: v1.JaegerIngressOpenShiftSpec{
				SkipLogout: &trueVar,
			},
		},
	}

	// test
	enableLogOut(uiOpts, spec)

	// verify
	assert.Len(t, uiOpts["menu"], 1)
	expected := []interface{}{internalLink}
	assert.Equal(t, expected, uiOpts["menu"])
}

func TestCustomMenuGetsLogOutAdded(t *testing.T) {
	// prepare
	internalLink := map[string]interface{}{
		"label": "Some internal links",
		"items": []interface{}{
			map[string]interface{}{
				"label": "The internal link",
				"url":   "http://example.com/internal",
			},
		},
	}
	uiOpts := map[string]interface{}{
		"menu": []interface{}{internalLink},
	}
	spec := &v1.JaegerSpec{
		Ingress: v1.JaegerIngressSpec{
			Security: v1.IngressSecurityOAuthProxy,
		},
	}

	// test
	enableLogOut(uiOpts, spec)

	// verify
	expected := []interface{}{
		internalLink,
		map[string]interface{}{
			"label":        "Log Out",
			"url":          "/oauth/sign_in",
			"anchorTarget": "_self",
		},
	}
	assert.Len(t, uiOpts["menu"], 2)
	assert.Equal(t, expected, uiOpts["menu"])
}

func TestCustomMenuGetsLogOutSkipped(t *testing.T) {
	// prepare
	internalLink := map[string]interface{}{
		"label": "Some internal links",
		"items": []interface{}{
			map[string]interface{}{
				"label": "The internal link",
				"url":   "http://example.com/internal",
			},
		},
	}
	logout := map[string]interface{}{
		"label":        "Custom Log Out",
		"url":          "https://example.com/custom/path/to/oauth/sign_in",
		"anchorTarget": "_self",
	}
	uiOpts := map[string]interface{}{
		"menu": []interface{}{
			internalLink,
			logout,
		},
	}
	spec := &v1.JaegerSpec{
		Ingress: v1.JaegerIngressSpec{
			Security: v1.IngressSecurityOAuthProxy,
		},
	}

	// test
	enableLogOut(uiOpts, spec)

	// verify
	expected := []interface{}{
		internalLink,
		logout,
	}
	assert.Len(t, uiOpts["menu"], 2)
	assert.Equal(t, expected, uiOpts["menu"])
}

func assertHasAllObjects(t *testing.T, name string, s S, deployments map[string]bool, daemonsets map[string]bool, services map[string]bool, ingresses map[string]bool, routes map[string]bool, serviceAccounts map[string]bool, configMaps map[string]bool, consoleLinks map[string]bool) {
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

	for _, o := range s.ConsoleLinks(s.routes) {
		consoleLinks[o.Name] = true
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

	for k, v := range consoleLinks {
		assert.True(t, v, "Expected %s to have been returned from the list of console links", k)
	}
}
