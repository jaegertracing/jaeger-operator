//+build elasticsearch

package e2e

import (
	"context"
	"testing"

	framework "github.com/operator-framework/operator-sdk/pkg/test"
	"github.com/operator-framework/operator-sdk/pkg/test/e2eutil"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"

	v1 "github.com/jaegertracing/jaeger-operator/pkg/apis/jaegertracing/v1"
)

type ElasticSearchBasicTestSuite struct {
	suite.Suite
}

func (suite *ElasticSearchBasicTestSuite) SetupSuite() {
	t = suite.T()
	var err error
	ctx, err = prepare(t)
	if err != nil {
		if ctx != nil {
			ctx.Cleanup()
		}
		require.FailNow(t, "Failed in prepare")
	}
	fw = framework.Global
	namespace = ctx.GetID()
	require.NotNil(t, namespace, "GetID failed")

	addToFrameworkSchemeForSmokeTests(t)

	if isOpenShift(t) {
		esServerUrls = "http://elasticsearch." + storageNamespace + ".svc.cluster.local:9200"
	}
}

func (suite *ElasticSearchBasicTestSuite) TearDownSuite() {
	handleSuiteTearDown()
}

func TestElasticSearchSuite(t *testing.T) {
	suite.Run(t, new(ElasticSearchBasicTestSuite))
}

func (suite *ElasticSearchBasicTestSuite) SetupTest() {
	t = suite.T()
}

func (suite *ElasticSearchBasicTestSuite) AfterTest(suiteName, testName string) {
	handleTestFailure()
}

func (suite *ElasticSearchBasicTestSuite) TestSparkDependenciesES() {
	if skipESExternal {
		t.Skip("This test requires an insecure ElasticSearch instance")
	}
	storage := v1.JaegerStorageSpec{
		Type: v1.JaegerESStorage,
		Options: v1.NewOptions(map[string]interface{}{
			"es.server-urls": esServerUrls,
		}),
	}
	err := sparkTest(t, framework.Global, ctx, storage)
	require.NoError(t, err, "SparkTest failed")
}

func (suite *ElasticSearchBasicTestSuite) TestSimpleProd() {
	if skipESExternal {
		t.Skip("This case is covered by the self_provisioned_elasticsearch_test")
	}
	err := WaitForStatefulset(t, fw.KubeClient, storageNamespace, string(v1.JaegerESStorage), retryInterval, timeout)
	require.NoError(t, err, "Error waiting for elasticsearch")

	// create jaeger custom resource
	name := "simple-prod"
	exampleJaeger := GetJaegerSimpleProdWithServerUrlsCR(name, esServerUrls)
	err = fw.Client.Create(context.TODO(), exampleJaeger, &framework.CleanupOptions{TestContext: ctx, Timeout: timeout, RetryInterval: retryInterval})
	require.NoError(t, err, "Error deploying example Jaeger")
	defer undeployJaegerInstance(exampleJaeger)

	err = e2eutil.WaitForDeployment(t, fw.KubeClient, namespace, name+"-collector", 1, retryInterval, timeout)
	require.NoError(t, err, "Error waiting for collector deployment")

	err = e2eutil.WaitForDeployment(t, fw.KubeClient, namespace, name+"-query", 1, retryInterval, timeout)
	require.NoError(t, err, "Error waiting for query deployment")

	ProductionSmokeTest(name)
}
