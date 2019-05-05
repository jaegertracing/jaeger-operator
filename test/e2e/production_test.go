// +build elasticsearch

package e2e

import (
	goctx "context"
	"testing"

	"github.com/stretchr/testify/suite"
	framework "github.com/operator-framework/operator-sdk/pkg/test"
	"github.com/operator-framework/operator-sdk/pkg/test/e2eutil"
	"github.com/stretchr/testify/require"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/jaegertracing/jaeger-operator/pkg/apis/jaegertracing/v1"
)

type ProductionTestSuite struct {
	suite.Suite
}

func(suite *ProductionTestSuite) SetupSuite() {
	t = suite.T()
	var err error
	ctx, err = prepare(t)
	if (err != nil) {
		ctx.Cleanup()
		require.FailNow(t, "Failed in prepare")
	}
	fw = framework.Global
	namespace, _ = ctx.GetNamespace()
	require.NotNil(t, namespace, "GetNamespace failed")

	addToFrameworkSchemeForSmokeTests(t)
}

func (suite *ProductionTestSuite) TearDownSuite() {
	ctx.Cleanup()
}

func TestProductionSuite(t *testing.T) {
	suite.Run(t, new(ProductionTestSuite))
}

func (suite *ProductionTestSuite) TestSimpleProd()  {
	err := WaitForStatefulset(t, fw.KubeClient, storageNamespace, "elasticsearch", retryInterval, timeout)
	require.NoError(t, err, "Error waiting for elasticsearch")

	// create jaeger custom resource
	exampleJaeger := &v1.Jaeger{
		TypeMeta: metav1.TypeMeta{
			Kind:       "Jaeger",
			APIVersion: "jaegertracing.io/v1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      "simple-prod",
			Namespace: namespace,
		},
		Spec: v1.JaegerSpec{
			Strategy: "production",
			Storage: v1.JaegerStorageSpec{
				Type: "elasticsearch",
				Options: v1.NewOptions(map[string]interface{}{
					"es.server-urls": esServerUrls,
				}),
			},
		},
	}
	err = fw.Client.Create(goctx.TODO(), exampleJaeger, &framework.CleanupOptions{TestContext: ctx, Timeout: timeout, RetryInterval: retryInterval})
	require.NoError(t, err, "Error deploying example Jaeger")

	err = e2eutil.WaitForDeployment(t, fw.KubeClient, namespace, "simple-prod-collector", 1, retryInterval, timeout)
	require.NoError(t, err, "Error waiting for collector deployment")

	err = e2eutil.WaitForDeployment(t, fw.KubeClient, namespace, "simple-prod-query", 1, retryInterval, timeout)
	require.NoError(t, err, "Error waiting for query deployment")

	queryPod, err := GetPod(namespace, "simple-prod-query", "jaegertracing/jaeger-query", fw.KubeClient)
	require.NoError(t, err, "Error getting Pod")

	collectorPod, err := GetPod(namespace, "simple-prod-collector", "jaegertracing/jaeger-collector", fw.KubeClient)
	require.NoError(t, err, "Error getting Pod")

	portForw, closeChan, err := CreatePortForward(namespace, queryPod.Name, []string{"16686"}, fw.KubeConfig)
	require.NoError(t, err, "Error creating port forward")

	defer portForw.Close()
	defer close(closeChan)
	portForwColl, closeChanColl, err := CreatePortForward(namespace, collectorPod.Name, []string{"14268"}, fw.KubeConfig)
	require.NoError(t, err, "Error creating port forward")

	defer portForwColl.Close()
	defer close(closeChanColl)
	err = SmokeTest("http://localhost:16686/api/traces", "http://localhost:14268/api/traces", "foobar", retryInterval, timeout)
	require.NoError(t, err, "Error running smoketest")
}
