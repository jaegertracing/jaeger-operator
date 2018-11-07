package controller

import (
	"context"
	"fmt"
	"testing"

	"github.com/operator-framework/operator-sdk/pkg/sdk"
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
	c := newAllInOneController(context.TODO(), v1alpha1.NewJaeger(name))
	objs := c.Create()
	assertDeploymentsAndServicesForAllInOne(t, name, objs, false)
}

func TestCreateAllInOneDeploymentOnOpenShift(t *testing.T) {
	viper.Set("platform", "openshift")
	defer viper.Reset()
	name := "TestCreateAllInOneDeploymentOnOpenShift"
	c := newAllInOneController(context.TODO(), v1alpha1.NewJaeger(name))
	objs := c.Create()
	assertDeploymentsAndServicesForAllInOne(t, name, objs, false)
}

func TestCreateAllInOneDeploymentWithDaemonSetAgent(t *testing.T) {
	name := "TestCreateAllInOneDeploymentWithDaemonSetAgent"

	j := v1alpha1.NewJaeger(name)
	j.Spec.Agent.Strategy = "DaemonSet"

	c := newAllInOneController(context.TODO(), j)
	objs := c.Create()
	assertDeploymentsAndServicesForAllInOne(t, name, objs, true)
}

func TestUpdateAllInOneDeployment(t *testing.T) {
	c := newAllInOneController(context.TODO(), v1alpha1.NewJaeger("TestUpdateAllInOneDeployment"))
	objs := c.Update()
	assert.Len(t, objs, 0)
}

func TestDelegateAllInOneDepedencies(t *testing.T) {
	// for now, we just have storage dependencies
	c := newAllInOneController(context.TODO(), v1alpha1.NewJaeger("TestDelegateAllInOneDepedencies"))
	assert.Equal(t, c.Dependencies(), storage.Dependencies(c.jaeger))
}

func assertDeploymentsAndServicesForAllInOne(t *testing.T, name string, objs []sdk.Object, hasDaemonSet bool) {
	if hasDaemonSet {
		assert.Len(t, objs, 6)
	} else {
		assert.Len(t, objs, 5)
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
