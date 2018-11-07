package controller

import (
	"context"
	"fmt"
	"testing"

	"github.com/operator-framework/operator-sdk/pkg/sdk"
	"github.com/spf13/viper"
	"github.com/stretchr/testify/assert"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/jaegertracing/jaeger-operator/pkg/apis/io/v1alpha1"
	"github.com/jaegertracing/jaeger-operator/pkg/storage"
)

func init() {
	viper.SetDefault("jaeger-version", "1.6")
	viper.SetDefault("jaeger-agent-image", "jaegertracing/jaeger-agent")
}

func TestCreateProductionDeployment(t *testing.T) {
	name := "TestCreateProductionDeployment"
	c := newProductionController(context.TODO(), v1alpha1.NewJaeger(name))
	objs := c.Create()
	assertDeploymentsAndServicesForProduction(t, name, objs, false)
}

func TestCreateProductionDeploymentOnOpenShift(t *testing.T) {
	viper.Set("platform", "openshift")
	defer viper.Reset()
	name := "TestCreateProductionDeploymentOnOpenShift"
	c := newProductionController(context.TODO(), v1alpha1.NewJaeger(name))
	objs := c.Create()
	assertDeploymentsAndServicesForProduction(t, name, objs, false)
}

func TestCreateProductionDeploymentWithDaemonSetAgent(t *testing.T) {
	name := "TestCreateProductionDeploymentWithDaemonSetAgent"

	j := v1alpha1.NewJaeger(name)
	j.Spec.Agent.Strategy = "DaemonSet"

	c := newProductionController(context.TODO(), j)
	objs := c.Create()
	assertDeploymentsAndServicesForProduction(t, name, objs, true)
}

func TestUpdateProductionDeployment(t *testing.T) {
	name := "TestUpdateProductionDeployment"
	c := newProductionController(context.TODO(), v1alpha1.NewJaeger(name))
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

	ctrl := NewController(context.TODO(), jaeger)
	objs := ctrl.Create()
	deployments := getDeployments(objs)
	for _, dep := range deployments {
		args := dep.Spec.Template.Spec.Containers[0].Args
		assert.Len(t, args, 3)
		for _, arg := range args {
			assert.Contains(t, arg, "es.")
		}
	}
}

func TestDelegateProductionDepedencies(t *testing.T) {
	// for now, we just have storage dependencies
	c := newProductionController(context.TODO(), v1alpha1.NewJaeger("TestDelegateProductionDepedencies"))
	assert.Equal(t, c.Dependencies(), storage.Dependencies(c.jaeger))
}

func assertDeploymentsAndServicesForProduction(t *testing.T, name string, objs []sdk.Object, hasDaemonSet bool) {
	if hasDaemonSet {
		assert.Len(t, objs, 6)
	} else {
		assert.Len(t, objs, 5)
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

	var ingresses map[string]bool
	var routes map[string]bool
	if viper.GetString("platform") == v1alpha1.FlagPlatformOpenShift {
		ingresses = map[string]bool{}
		routes = map[string]bool{
			fmt.Sprintf("%s", name): false,
		}
	} else {
		ingresses = map[string]bool{
			fmt.Sprintf("%s-query", name): false,
		}
		routes = map[string]bool{}
	}

	assertHasAllObjects(t, name, objs, deployments, daemonsets, services, ingresses, routes)
}
