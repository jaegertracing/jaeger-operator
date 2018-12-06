package strategy

import (
	"context"
	"fmt"
	"reflect"
	"testing"

	"github.com/spf13/viper"
	"github.com/stretchr/testify/assert"
	batchv1beta1 "k8s.io/api/batch/v1beta1"
	"k8s.io/apimachinery/pkg/runtime"

	"github.com/jaegertracing/jaeger-operator/pkg/apis/io/v1alpha1"
	"github.com/jaegertracing/jaeger-operator/pkg/storage"
)

func init() {
	viper.SetDefault("jaeger-version", "1.6")
	viper.SetDefault("jaeger-agent-image", "jaegertracing/jaeger-agent")
}

func TestCreateAllInOneDeployment(t *testing.T) {
	name := "TestCreateAllInOneDeployment"
	c := newAllInOneStrategy(context.TODO(), v1alpha1.NewJaeger(name))
	objs := c.Create()
	assertDeploymentsAndServicesForAllInOne(t, name, objs, false, false, false)
}

func TestCreateAllInOneDeploymentOnOpenShift(t *testing.T) {
	viper.Set("platform", "openshift")
	defer viper.Reset()
	name := "TestCreateAllInOneDeploymentOnOpenShift"

	jaeger := v1alpha1.NewJaeger(name)
	normalize(jaeger)

	c := newAllInOneStrategy(context.TODO(), jaeger)
	objs := c.Create()
	assertDeploymentsAndServicesForAllInOne(t, name, objs, false, true, false)
}

func TestCreateAllInOneDeploymentWithDaemonSetAgent(t *testing.T) {
	name := "TestCreateAllInOneDeploymentWithDaemonSetAgent"

	j := v1alpha1.NewJaeger(name)
	j.Spec.Agent.Strategy = "DaemonSet"

	c := newAllInOneStrategy(context.TODO(), j)
	objs := c.Create()
	assertDeploymentsAndServicesForAllInOne(t, name, objs, true, false, false)
}

func TestCreateAllInOneDeploymentWithUIConfigMap(t *testing.T) {
	name := "TestCreateAllInOneDeploymentWithUIConfigMap"

	j := v1alpha1.NewJaeger(name)
	j.Spec.UI.Options = v1alpha1.NewFreeForm(map[string]interface{}{
		"tracking": map[string]interface{}{
			"gaID": "UA-000000-2",
		},
	})

	c := newAllInOneStrategy(context.TODO(), j)
	objs := c.Create()
	assertDeploymentsAndServicesForAllInOne(t, name, objs, false, false, true)
}

func TestUpdateAllInOneDeployment(t *testing.T) {
	c := newAllInOneStrategy(context.TODO(), v1alpha1.NewJaeger("TestUpdateAllInOneDeployment"))
	objs := c.Update()
	assert.Len(t, objs, 0)
}

func TestDelegateAllInOneDepedencies(t *testing.T) {
	// for now, we just have storage dependencies
	c := newAllInOneStrategy(context.TODO(), v1alpha1.NewJaeger("TestDelegateAllInOneDepedencies"))
	assert.Equal(t, c.Dependencies(), storage.Dependencies(c.jaeger))
}

func assertDeploymentsAndServicesForAllInOne(t *testing.T, name string, objs []runtime.Object, hasDaemonSet bool, hasOAuthProxy bool, hasConfigMap bool) {
	// TODO(jpkroehling): this func deserves a refactoring already

	expectedNumObjs := 6

	if hasDaemonSet {
		expectedNumObjs++
	}

	if hasOAuthProxy {
		expectedNumObjs++
	}

	if hasConfigMap {
		expectedNumObjs++
	}

	assert.Len(t, objs, expectedNumObjs)

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
	assertHasAllObjects(t, name, objs, deployments, daemonsets, services, ingresses, routes, serviceAccounts, configMaps)
}

func TestSparkDependenciesAllInOne(t *testing.T) {
	testSparkDependencies(t, func(jaeger *v1alpha1.Jaeger) S {
		return &allInOneStrategy{jaeger: jaeger}
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
		objs := s.Create()
		cronJobs := getTypesOf(objs, reflect.TypeOf(&batchv1beta1.CronJob{}))
		if test.sparkCronJobEnabled {
			assert.Equal(t, 1, len(cronJobs))
		} else {
			assert.Equal(t, 0, len(cronJobs))
		}
	}
}

func TestEsIndexCleanerAllInOne(t *testing.T) {
	testEsIndexCleaner(t, func(jaeger *v1alpha1.Jaeger) S {
		return &allInOneStrategy{jaeger: jaeger}
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
		objs := s.Create()
		cronJobs := getTypesOf(objs, reflect.TypeOf(&batchv1beta1.CronJob{}))
		if test.sparkCronJobEnabled {
			assert.Equal(t, 1, len(cronJobs))
		} else {
			assert.Equal(t, 0, len(cronJobs))
		}
	}
}
