package strategy

import (
	"context"
	"fmt"
	"strings"
	"testing"

	"github.com/spf13/viper"
	"github.com/stretchr/testify/assert"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"

	"github.com/jaegertracing/jaeger-operator/pkg/apis/io/v1alpha1"
	"github.com/jaegertracing/jaeger-operator/pkg/storage"
)

func init() {
	viper.SetDefault("jaeger-version", "1.6")
	viper.SetDefault("jaeger-agent-image", "jaegertracing/jaeger-agent")
}

func TestCreateProductionDeployment(t *testing.T) {
	name := "TestCreateProductionDeployment"
	c := newProductionStrategy(context.TODO(), v1alpha1.NewJaeger(name))
	objs := c.Create()
	assertDeploymentsAndServicesForProduction(t, name, objs, false, false, false)
}

func TestCreateProductionDeploymentOnOpenShift(t *testing.T) {
	viper.Set("platform", "openshift")
	defer viper.Reset()
	name := "TestCreateProductionDeploymentOnOpenShift"

	jaeger := v1alpha1.NewJaeger(name)
	normalize(jaeger)

	c := newProductionStrategy(context.TODO(), jaeger)
	objs := c.Create()
	assertDeploymentsAndServicesForProduction(t, name, objs, false, true, false)
}

func TestCreateProductionDeploymentWithDaemonSetAgent(t *testing.T) {
	name := "TestCreateProductionDeploymentWithDaemonSetAgent"

	j := v1alpha1.NewJaeger(name)
	j.Spec.Agent.Strategy = "DaemonSet"

	c := newProductionStrategy(context.TODO(), j)
	objs := c.Create()
	assertDeploymentsAndServicesForProduction(t, name, objs, true, false, false)
}

func TestCreateProductionDeploymentWithUIConfigMap(t *testing.T) {
	name := "TestCreateProductionDeploymentWithUIConfigMap"

	j := v1alpha1.NewJaeger(name)
	j.Spec.UI.Options = v1alpha1.NewFreeForm(map[string]interface{}{
		"tracking": map[string]interface{}{
			"gaID": "UA-000000-2",
		},
	})

	c := newProductionStrategy(context.TODO(), j)
	objs := c.Create()
	assertDeploymentsAndServicesForProduction(t, name, objs, false, false, true)
}

func TestUpdateProductionDeployment(t *testing.T) {
	name := "TestUpdateProductionDeployment"
	c := newProductionStrategy(context.TODO(), v1alpha1.NewJaeger(name))
	assert.Len(t, c.Update(), 0)
}

func TestOptionsArePassed(t *testing.T) {
	jaeger := &v1alpha1.Jaeger{
		TypeMeta: metav1.TypeMeta{
			Kind:       "Jaeger",
			APIVersion: "io.jaegertracing/v1alpha1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      "simple-prod",
			Namespace: "simple-prod-ns",
		},
		Spec: v1alpha1.JaegerSpec{
			Strategy: "production",
			Storage: v1alpha1.JaegerStorageSpec{
				Type: "elasticsearch",
				Options: v1alpha1.NewOptions(map[string]interface{}{
					"es.server-urls": "http://elasticsearch.default.svc:9200",
					"es.username":    "elastic",
					"es.password":    "changeme",
				}),
			},
		},
	}

	ctrl := For(context.TODO(), jaeger)
	objs := ctrl.Create()
	deployments := getDeployments(objs)
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

func TestDelegateProductionDepedencies(t *testing.T) {
	// for now, we just have storage dependencies
	c := newProductionStrategy(context.TODO(), v1alpha1.NewJaeger("TestDelegateProductionDepedencies"))
	assert.Equal(t, c.Dependencies(), storage.Dependencies(c.jaeger))
}

func assertDeploymentsAndServicesForProduction(t *testing.T, name string, objs []runtime.Object, hasDaemonSet bool, hasOAuthProxy bool, hasConfigMap bool) {
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
	if viper.GetString("platform") == v1alpha1.FlagPlatformOpenShift {
		routes[name] = false
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
