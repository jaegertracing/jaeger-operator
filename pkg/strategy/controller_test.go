package strategy

import (
	"context"
	"reflect"
	"testing"

	"github.com/spf13/viper"
	"github.com/stretchr/testify/assert"
	appsv1 "k8s.io/api/apps/v1"
	"k8s.io/apimachinery/pkg/runtime"

	"github.com/jaegertracing/jaeger-operator/pkg/apis/io/v1alpha1"
)

func TestNewControllerForAllInOneAsDefault(t *testing.T) {
	jaeger := v1alpha1.NewJaeger("TestNewControllerForAllInOneAsDefault")

	ctrl := For(context.TODO(), jaeger)
	assert.Equal(t, ctrl.Type(), AllInOne)
}

func TestNewControllerForAllInOneAsExplicitValue(t *testing.T) {
	jaeger := v1alpha1.NewJaeger("TestNewControllerForAllInOneAsExplicitValue")
	jaeger.Spec.Strategy = "ALL-IN-ONE" // same as 'all-in-one'

	ctrl := For(context.TODO(), jaeger)
	assert.Equal(t, ctrl.Type(), AllInOne)
}

func TestNewControllerForProduction(t *testing.T) {
	jaeger := v1alpha1.NewJaeger("TestNewControllerForProduction")
	jaeger.Spec.Strategy = "production"
	jaeger.Spec.Storage.Type = "elasticsearch"

	ctrl := For(context.TODO(), jaeger)
	assert.Len(t, ctrl.Deployments(), 2)
}

func TestUnknownStorage(t *testing.T) {
	jaeger := v1alpha1.NewJaeger("TestNewControllerForProduction")
	jaeger.Spec.Storage.Type = "unknown"
	normalize(jaeger)
	assert.Equal(t, "memory", jaeger.Spec.Storage.Type)
}

func TestElasticsearchAsStorageOptions(t *testing.T) {
	jaeger := v1alpha1.NewJaeger("TestElasticsearchAsStorageOptions")
	jaeger.Spec.Strategy = "production"
	jaeger.Spec.Storage.Type = "elasticsearch"
	jaeger.Spec.Storage.Options = v1alpha1.NewOptions(map[string]interface{}{
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
	jaeger := &v1alpha1.Jaeger{}
	normalize(jaeger)
	assert.NotEmpty(t, jaeger.Name)
}

func TestIncompatibleStorageForProduction(t *testing.T) {
	jaeger := &v1alpha1.Jaeger{
		Spec: v1alpha1.JaegerSpec{
			Strategy: "production",
			Storage: v1alpha1.JaegerStorageSpec{
				Type: "memory",
			},
		},
	}
	normalize(jaeger)
	assert.Equal(t, "allInOne", jaeger.Spec.Strategy)
}

func TestIncompatibleStorageForStreaming(t *testing.T) {
	jaeger := &v1alpha1.Jaeger{
		Spec: v1alpha1.JaegerSpec{
			Strategy: "streaming",
			Storage: v1alpha1.JaegerStorageSpec{
				Type: "memory",
			},
		},
	}
	normalize(jaeger)
	assert.Equal(t, "allInOne", jaeger.Spec.Strategy)
}

func TestDeprecatedAllInOneStrategy(t *testing.T) {
	jaeger := &v1alpha1.Jaeger{
		Spec: v1alpha1.JaegerSpec{
			Strategy: "all-in-one",
		},
	}
	For(context.TODO(), jaeger)
	assert.Equal(t, "allInOne", jaeger.Spec.Strategy)
}

func TestStorageMemoryOnlyUsedWithAllInOneStrategy(t *testing.T) {
	jaeger := &v1alpha1.Jaeger{
		Spec: v1alpha1.JaegerSpec{
			Strategy: "production",
			Storage: v1alpha1.JaegerStorageSpec{
				Type: "memory",
			},
		},
	}
	For(context.TODO(), jaeger)
	assert.Equal(t, "allInOne", jaeger.Spec.Strategy)
}

func TestSetSecurityToNoneByDefault(t *testing.T) {
	jaeger := v1alpha1.NewJaeger("TestSetSecurityToNoneByDefault")
	normalize(jaeger)
	assert.Equal(t, v1alpha1.IngressSecurityNone, jaeger.Spec.Ingress.Security)
}

func TestSetSecurityToNoneWhenExplicitSettingToNone(t *testing.T) {
	jaeger := v1alpha1.NewJaeger("TestSetSecurityToNoneWhenExplicitSettingToNone")
	jaeger.Spec.Ingress.Security = v1alpha1.IngressSecurityNoneExplicit
	normalize(jaeger)
	assert.Equal(t, v1alpha1.IngressSecurityNone, jaeger.Spec.Ingress.Security)
}

func TestSetSecurityToOAuthProxyByDefaultOnOpenShift(t *testing.T) {
	viper.Set("platform", "openshift")
	defer viper.Reset()

	jaeger := v1alpha1.NewJaeger("TestSetSecurityToOAuthProxyByDefaultOnOpenShift")
	normalize(jaeger)

	assert.Equal(t, v1alpha1.IngressSecurityOAuthProxy, jaeger.Spec.Ingress.Security)
}

func TestSetSecurityToNoneOnNonOpenShift(t *testing.T) {
	jaeger := v1alpha1.NewJaeger("TestSetSecurityToNoneOnNonOpenShift")
	jaeger.Spec.Ingress.Security = v1alpha1.IngressSecurityOAuthProxy

	normalize(jaeger)

	assert.Equal(t, v1alpha1.IngressSecurityNone, jaeger.Spec.Ingress.Security)
}

func TestAcceptExplicitValueFromSecurityWhenOnOpenShift(t *testing.T) {
	viper.Set("platform", "openshift")
	defer viper.Reset()

	jaeger := v1alpha1.NewJaeger("TestAcceptExplicitValueFromSecurityWhenOnOpenShift")
	jaeger.Spec.Ingress.Security = v1alpha1.IngressSecurityNoneExplicit

	normalize(jaeger)

	assert.Equal(t, v1alpha1.IngressSecurityNone, jaeger.Spec.Ingress.Security)
}

func TestNormalizeIndexCleaner(t *testing.T) {
	viper.Set("jaeger-es-index-cleaner-image", "foo")
	defer viper.Reset()
	trueVar := true
	falseVar := false
	tests := []struct {
		underTest v1alpha1.JaegerEsIndexCleanerSpec
		expected  v1alpha1.JaegerEsIndexCleanerSpec
	}{
		{underTest: v1alpha1.JaegerEsIndexCleanerSpec{},
			expected: v1alpha1.JaegerEsIndexCleanerSpec{Image: "foo", Schedule: "55 23 * * *", NumberOfDays: 7, Enabled: &trueVar}},
		{underTest: v1alpha1.JaegerEsIndexCleanerSpec{Image: "bla", Schedule: "lol", NumberOfDays: 55, Enabled: &falseVar},
			expected: v1alpha1.JaegerEsIndexCleanerSpec{Image: "bla", Schedule: "lol", NumberOfDays: 55, Enabled: &falseVar}},
	}
	for _, test := range tests {
		normalizeIndexCleaner(&test.underTest, "elasticsearch")
		assert.Equal(t, test.expected, test.underTest)
	}
}

func TestNormalizeSparkDependencies(t *testing.T) {
	viper.Set("jaeger-spark-dependencies-image", "foo")
	defer viper.Reset()
	trueVar := true
	falseVar := false
	tests := []struct {
		underTest v1alpha1.JaegerDependenciesSpec
		expected  v1alpha1.JaegerDependenciesSpec
	}{
		{underTest: v1alpha1.JaegerDependenciesSpec{},
			expected: v1alpha1.JaegerDependenciesSpec{Schedule: "55 23 * * *", Image: "foo", Enabled: &trueVar}},
		{underTest: v1alpha1.JaegerDependenciesSpec{Schedule: "foo", Image: "bla", Enabled: &falseVar},
			expected: v1alpha1.JaegerDependenciesSpec{Schedule: "foo", Image: "bla", Enabled: &falseVar}},
	}
	for _, test := range tests {
		normalizeSparkDependencies(&test.underTest, "elasticsearch")
		assert.Equal(t, test.expected, test.underTest)
	}
}

func getDeployments(objs []runtime.Object) []*appsv1.Deployment {
	var deps []*appsv1.Deployment

	for _, obj := range objs {
		switch obj.(type) {
		case *appsv1.Deployment:
			deps = append(deps, obj.(*appsv1.Deployment))
		}
	}
	return deps
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

func getTypesOf(
	objs []runtime.Object,
	typ reflect.Type,
) []runtime.Object {
	var theTypes []runtime.Object
	for _, obj := range objs {
		if typ == reflect.TypeOf(obj) {
			theTypes = append(theTypes, obj)
		}
	}
	return theTypes
}
