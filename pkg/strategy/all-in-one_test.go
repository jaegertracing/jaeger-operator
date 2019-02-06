package strategy

import (
	"fmt"
	"testing"

	"github.com/spf13/viper"
	"github.com/stretchr/testify/assert"

	"github.com/jaegertracing/jaeger-operator/pkg/apis/io/v1alpha1"
	"github.com/jaegertracing/jaeger-operator/pkg/storage"
)

func init() {
	viper.SetDefault("jaeger-version", "1.6")
	viper.SetDefault("jaeger-agent-image", "jaegertracing/jaeger-agent")
}

func TestCreateAllInOneDeployment(t *testing.T) {
	name := "TestCreateAllInOneDeployment"
	c := newAllInOneStrategy(v1alpha1.NewJaeger(name))
	assertDeploymentsAndServicesForAllInOne(t, name, c, false, false, false)
}

func TestCreateAllInOneDeploymentOnOpenShift(t *testing.T) {
	viper.Set("platform", "openshift")
	defer viper.Reset()
	name := "TestCreateAllInOneDeploymentOnOpenShift"

	jaeger := v1alpha1.NewJaeger(name)
	normalize(jaeger)

	c := newAllInOneStrategy(jaeger)
	assertDeploymentsAndServicesForAllInOne(t, name, c, false, true, false)
}

func TestCreateAllInOneDeploymentWithDaemonSetAgent(t *testing.T) {
	name := "TestCreateAllInOneDeploymentWithDaemonSetAgent"

	j := v1alpha1.NewJaeger(name)
	j.Spec.Agent.Strategy = "DaemonSet"

	c := newAllInOneStrategy(j)
	assertDeploymentsAndServicesForAllInOne(t, name, c, true, false, false)
}

func TestCreateAllInOneDeploymentWithUIConfigMap(t *testing.T) {
	name := "TestCreateAllInOneDeploymentWithUIConfigMap"

	j := v1alpha1.NewJaeger(name)
	j.Spec.UI.Options = v1alpha1.NewFreeForm(map[string]interface{}{
		"tracking": map[string]interface{}{
			"gaID": "UA-000000-2",
		},
	})

	c := newAllInOneStrategy(j)
	assertDeploymentsAndServicesForAllInOne(t, name, c, false, false, true)
}

func TestDelegateAllInOneDependencies(t *testing.T) {
	// for now, we just have storage dependencies
	j := v1alpha1.NewJaeger("TestDelegateAllInOneDependencies")
	c := newAllInOneStrategy(j)
	assert.Equal(t, c.Dependencies(), storage.Dependencies(j))
}

func assertDeploymentsAndServicesForAllInOne(t *testing.T, name string, s S, hasDaemonSet bool, hasOAuthProxy bool, hasConfigMap bool) {
	// TODO(jpkroehling): this func deserves a refactoring already

	expectedNumObjs := 7

	if hasDaemonSet {
		expectedNumObjs++
	}

	if hasOAuthProxy {
		expectedNumObjs++
	}

	if hasConfigMap {
		expectedNumObjs++
	}

	// we should have one deployment, named after the Jaeger's name (ObjectMeta.Name)
	deployments := map[string]bool{
		name: false,
	}

	daemonsets := map[string]bool{
		fmt.Sprintf("%s-agent-daemonset", name): !hasDaemonSet,
	}

	// and these services
	services := map[string]bool{
		fmt.Sprintf("%s-agent", name):     false,
		fmt.Sprintf("%s-collector", name): false,
		fmt.Sprintf("%s-query", name):     false,
	}

	// the ingress rule, if we are not on openshift
	ingresses := map[string]bool{}
	routes := map[string]bool{}
	if viper.GetString("platform") == v1alpha1.FlagPlatformOpenShift {
		routes[fmt.Sprintf("%s", name)] = false
	} else {
		ingresses[fmt.Sprintf("%s-query", name)] = false
	}

	serviceAccounts := map[string]bool{}
	if hasOAuthProxy {
		serviceAccounts[fmt.Sprintf("%s-ui-proxy", name)] = false
	}

	configMaps := map[string]bool{}
	if hasConfigMap {
		configMaps[fmt.Sprintf("%s-ui-configuration", name)] = false
	}
	assertHasAllObjects(t, name, s, deployments, daemonsets, services, ingresses, routes, serviceAccounts, configMaps)
}

func TestSparkDependenciesAllInOne(t *testing.T) {
	testSparkDependencies(t, func(jaeger *v1alpha1.Jaeger) S {
		return newAllInOneStrategy(jaeger)
	})
}

func testSparkDependencies(t *testing.T, fce func(jaeger *v1alpha1.Jaeger) S) {
	trueVar := true
	tests := []struct {
		jaeger              *v1alpha1.Jaeger
		sparkCronJobEnabled bool
	}{
		{jaeger: &v1alpha1.Jaeger{Spec: v1alpha1.JaegerSpec{
			Storage: v1alpha1.JaegerStorageSpec{Type: "elasticsearch",
				SparkDependencies: v1alpha1.JaegerDependenciesSpec{Enabled: &trueVar}},
		}}, sparkCronJobEnabled: true},
		{jaeger: &v1alpha1.Jaeger{Spec: v1alpha1.JaegerSpec{
			Storage: v1alpha1.JaegerStorageSpec{Type: "cassandra",
				SparkDependencies: v1alpha1.JaegerDependenciesSpec{Enabled: &trueVar}},
		}}, sparkCronJobEnabled: true},
		{jaeger: &v1alpha1.Jaeger{Spec: v1alpha1.JaegerSpec{
			Storage: v1alpha1.JaegerStorageSpec{Type: "kafka",
				SparkDependencies: v1alpha1.JaegerDependenciesSpec{Enabled: &trueVar}},
		}}, sparkCronJobEnabled: false},
		{jaeger: &v1alpha1.Jaeger{Spec: v1alpha1.JaegerSpec{
			Storage: v1alpha1.JaegerStorageSpec{Type: "elasticsearch"},
		}}, sparkCronJobEnabled: false},
	}
	for _, test := range tests {
		s := fce(test.jaeger)
		cronJobs := s.CronJobs()
		if test.sparkCronJobEnabled {
			assert.Equal(t, 1, len(cronJobs))
		} else {
			assert.Equal(t, 0, len(cronJobs))
		}
	}
}

func TestEsIndexCleanerAllInOne(t *testing.T) {
	testEsIndexCleaner(t, func(jaeger *v1alpha1.Jaeger) S {
		return newAllInOneStrategy(jaeger)
	})
}

func testEsIndexCleaner(t *testing.T, fce func(jaeger *v1alpha1.Jaeger) S) {
	trueVar := true
	tests := []struct {
		jaeger              *v1alpha1.Jaeger
		sparkCronJobEnabled bool
	}{
		{jaeger: &v1alpha1.Jaeger{Spec: v1alpha1.JaegerSpec{
			Storage: v1alpha1.JaegerStorageSpec{Type: "elasticsearch",
				EsIndexCleaner: v1alpha1.JaegerEsIndexCleanerSpec{Enabled: &trueVar}},
		}}, sparkCronJobEnabled: true},
		{jaeger: &v1alpha1.Jaeger{Spec: v1alpha1.JaegerSpec{
			Storage: v1alpha1.JaegerStorageSpec{Type: "cassandra",
				EsIndexCleaner: v1alpha1.JaegerEsIndexCleanerSpec{Enabled: &trueVar}},
		}}, sparkCronJobEnabled: false},
		{jaeger: &v1alpha1.Jaeger{Spec: v1alpha1.JaegerSpec{
			Storage: v1alpha1.JaegerStorageSpec{Type: "kafka",
				EsIndexCleaner: v1alpha1.JaegerEsIndexCleanerSpec{Enabled: &trueVar}},
		}}, sparkCronJobEnabled: false},
		{jaeger: &v1alpha1.Jaeger{Spec: v1alpha1.JaegerSpec{
			Storage: v1alpha1.JaegerStorageSpec{Type: "elasticsearch"},
		}}, sparkCronJobEnabled: false},
	}
	for _, test := range tests {
		s := fce(test.jaeger)
		cronJobs := s.CronJobs()
		if test.sparkCronJobEnabled {
			assert.Equal(t, 1, len(cronJobs))
		} else {
			assert.Equal(t, 0, len(cronJobs))
		}
	}
}
