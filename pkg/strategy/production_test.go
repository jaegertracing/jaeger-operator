package strategy

import (
	"context"
	"fmt"
	"strings"
	"testing"

	"github.com/spf13/viper"
	"github.com/stretchr/testify/assert"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/jaegertracing/jaeger-operator/pkg/apis/jaegertracing/v1"
	"github.com/jaegertracing/jaeger-operator/pkg/storage"
)

func init() {
	viper.SetDefault("jaeger-version", "1.6")
	viper.SetDefault("jaeger-agent-image", "jaegertracing/jaeger-agent")
}

func TestCreateProductionDeployment(t *testing.T) {
	name := "TestCreateProductionDeployment"
	c := newProductionStrategy(v1.NewJaeger(name))
	assertDeploymentsAndServicesForProduction(t, name, c, false, false, false)
}

func TestCreateProductionDeploymentOnOpenShift(t *testing.T) {
	viper.Set("platform", "openshift")
	defer viper.Reset()
	name := "TestCreateProductionDeploymentOnOpenShift"

	jaeger := v1.NewJaeger(name)
	normalize(jaeger)

	c := newProductionStrategy(jaeger)
	assertDeploymentsAndServicesForProduction(t, name, c, false, true, false)
}

func TestCreateProductionDeploymentWithDaemonSetAgent(t *testing.T) {
	name := "TestCreateProductionDeploymentWithDaemonSetAgent"

	j := v1.NewJaeger(name)
	j.Spec.Agent.Strategy = "DaemonSet"

	c := newProductionStrategy(j)
	assertDeploymentsAndServicesForProduction(t, name, c, true, false, false)
}

func TestCreateProductionDeploymentWithUIConfigMap(t *testing.T) {
	name := "TestCreateProductionDeploymentWithUIConfigMap"

	j := v1.NewJaeger(name)
	j.Spec.UI.Options = v1.NewFreeForm(map[string]interface{}{
		"tracking": map[string]interface{}{
			"gaID": "UA-000000-2",
		},
	})

	c := newProductionStrategy(j)
	assertDeploymentsAndServicesForProduction(t, name, c, false, false, true)
}

func TestOptionsArePassed(t *testing.T) {
	jaeger := &v1.Jaeger{
		TypeMeta: metav1.TypeMeta{
			Kind:       "Jaeger",
			APIVersion: "jaegertracing.io/v1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      "simple-prod",
			Namespace: "simple-prod-ns",
		},
		Spec: v1.JaegerSpec{
			Strategy: "production",
			Storage: v1.JaegerStorageSpec{
				Type: "elasticsearch",
				Options: v1.NewOptions(map[string]interface{}{
					"es.server-urls": "http://elasticsearch.default.svc:9200",
					"es.username":    "elastic",
					"es.password":    "changeme",
				}),
			},
		},
	}

	ctrl := For(context.TODO(), jaeger)
	deployments := ctrl.Deployments()
	for _, dep := range deployments {
		args := dep.Spec.Template.Spec.Containers[0].Args
		if strings.Contains(dep.Name, "collector") {
			// Including parameter for sampling config
			assert.Len(t, args, 4)
		} else {
			assert.Len(t, args, 3)
		}
		var escount int
		for _, arg := range args {
			if strings.Contains(arg, "es.") {
				escount++
			}
		}
		assert.Equal(t, 3, escount)
	}
}

func TestDelegateProductionDependencies(t *testing.T) {
	// for now, we just have storage dependencies
	j := v1.NewJaeger("TestDelegateProductionDependencies")
	j.Spec.Storage.Type = "cassandra"
	c := newProductionStrategy(j)
	assert.Equal(t, c.Dependencies(), storage.Dependencies(j))
}

func assertDeploymentsAndServicesForProduction(t *testing.T, name string, s S, hasDaemonSet bool, hasOAuthProxy bool, hasConfigMap bool) {
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

	deployments := map[string]bool{
		fmt.Sprintf("%s-collector", name): false,
		fmt.Sprintf("%s-query", name):     false,
	}

	daemonsets := map[string]bool{
		fmt.Sprintf("%s-agent-daemonset", name): !hasDaemonSet,
	}

	services := map[string]bool{
		fmt.Sprintf("%s-collector", name): false,
		fmt.Sprintf("%s-query", name):     false,
	}

	ingresses := map[string]bool{}
	routes := map[string]bool{}
	if viper.GetString("platform") == v1.FlagPlatformOpenShift {
		routes[name] = false
	} else {
		ingresses[fmt.Sprintf("%s-query", name)] = false
	}

	serviceAccounts := map[string]bool{fmt.Sprintf("%s", name): false}
	if hasOAuthProxy {
		serviceAccounts[fmt.Sprintf("%s-ui-proxy", name)] = false
	}

	configMaps := map[string]bool{}
	if hasConfigMap {
		configMaps[fmt.Sprintf("%s-ui-configuration", name)] = false
	}
	assertHasAllObjects(t, name, s, deployments, daemonsets, services, ingresses, routes, serviceAccounts, configMaps)
}

func TestSparkDependenciesProduction(t *testing.T) {
	testSparkDependencies(t, func(jaeger *v1.Jaeger) S {
		return newProductionStrategy(jaeger)
	})
}

func TestEsIndexCleanerProduction(t *testing.T) {
	testEsIndexCleaner(t, func(jaeger *v1.Jaeger) S {
		return newProductionStrategy(jaeger)
	})
}

func TestAgentSidecarIsInjectedIntoQueryForStreamingForProduction(t *testing.T) {
	j := v1.NewJaeger("TestAgentSidecarIsInjectedIntoQueryForStreamingForProduction")
	c := newProductionStrategy(j)
	for _, dep := range c.Deployments() {
		if strings.HasSuffix(dep.Name, "-query") {
			assert.Equal(t, 2, len(dep.Spec.Template.Spec.Containers))
			assert.Equal(t, "jaeger-agent", dep.Spec.Template.Spec.Containers[1].Name)
		}
	}
}
