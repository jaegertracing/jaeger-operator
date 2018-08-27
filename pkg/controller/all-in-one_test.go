package controller

import (
	"context"
	"fmt"
	"testing"

	"github.com/operator-framework/operator-sdk/pkg/sdk"
	"github.com/stretchr/testify/assert"

	"github.com/jaegertracing/jaeger-operator/pkg/apis/io/v1alpha1"
)

func TestCreateAllInOneDeployment(t *testing.T) {
	name := "TestCreateAllInOneDeployment"
	c := newAllInOneController(context.TODO(), v1alpha1.NewJaeger(name))
	objs := c.Create()
	assertDeploymentsAndServicesForAllInOne(t, name, objs)
}

func TestUpdateAllInOneDeployment(t *testing.T) {
	c := newAllInOneController(context.TODO(), v1alpha1.NewJaeger("TestUpdateAllInOneDeployment"))
	objs := c.Update()
	assert.Len(t, objs, 0)
}

func assertDeploymentsAndServicesForAllInOne(t *testing.T, name string, objs []sdk.Object) {
	assert.Len(t, objs, 6)

	// we should have one deployment, named after the Jaeger's name (ObjectMeta.Name)
	deployments := map[string]bool{
		name: false,
	}

	// and these services
	services := map[string]bool{
		fmt.Sprintf("%s-agent", name):     false,
		fmt.Sprintf("%s-collector", name): false,
		fmt.Sprintf("%s-query", name):     false,
		fmt.Sprintf("%s-zipkin", name):    false,
	}

	// and the ingress rule
	ingresses := map[string]bool{
		fmt.Sprintf("%s-query", name): false,
	}

	assertHasAllObjects(t, name, objs, deployments, services, ingresses)
}
