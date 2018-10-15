package controller

import (
	"context"
	"fmt"
	"testing"

	"github.com/operator-framework/operator-sdk/pkg/sdk"
	"github.com/spf13/viper"
	"github.com/stretchr/testify/assert"
	appsv1 "k8s.io/api/apps/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/jaegertracing/jaeger-operator/pkg/apis/io/v1alpha1"
)

func init() {
	viper.SetDefault(versionKey, versionValue)
	viper.SetDefault(agentImageKey, agentImageValue)
}

func TestCreateProductionDeployment(t *testing.T) {
	name := "TestCreateProductionDeployment"
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

func TestAgentIsInjectedIntoQuery(t *testing.T) {
	name := "TestAgentIsInjectedIntoQuery"
	c := newProductionController(context.TODO(), v1alpha1.NewJaeger(name))
	objs := c.Create()
	var dep *appsv1.Deployment

	for _, obj := range objs {
		switch obj.(type) {
		case *appsv1.Deployment:
			dep = obj.(*appsv1.Deployment)
			break
		}
	}

	assert.Len(t, dep.Spec.Template.Spec.Containers, 2)
	assert.Contains(t, dep.Spec.Template.Spec.Containers[1].Image, "jaeger-agent")
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
			Strategy: productionStrategy,
			Storage: v1alpha1.JaegerStorageSpec{
				Type: elasticsearch,
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

func assertDeploymentsAndServicesForProduction(t *testing.T, name string, objs []sdk.Object, hasDaemonSet bool) {
	if hasDaemonSet {
		assert.Len(t, objs, 7)
	} else {
		assert.Len(t, objs, 6)
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
		fmt.Sprintf("%s-zipkin", name):    false,
	}

	ingresses := map[string]bool{
		fmt.Sprintf("%s-query", name): false,
	}

	assertHasAllObjects(t, name, objs, deployments, daemonsets, services, ingresses)
}
