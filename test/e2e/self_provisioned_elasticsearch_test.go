// +build self_provisioned_elasticsearch

package e2e

import (
	goctx "context"
	"testing"

	framework "github.com/operator-framework/operator-sdk/pkg/test"
	"github.com/operator-framework/operator-sdk/pkg/test/e2eutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/jaegertracing/jaeger-operator/pkg/apis"
	"github.com/jaegertracing/jaeger-operator/pkg/apis/jaegertracing/v1"
	esv1 "github.com/jaegertracing/jaeger-operator/pkg/storage/elasticsearch/v1"
)

type SelfProvisionedTestSuite struct {
	suite.Suite
}

func(suite *SelfProvisionedTestSuite) SetupSuite() {
	t = suite.T()
	if !isOpenShift(t) {
		t.Skipf("Test %s is currently supported only on OpenShift because es-operator runs only on OpenShift\n", t.Name())
	}

	assert.NoError(t, framework.AddToFrameworkScheme(apis.AddToScheme, &v1.JaegerList{
		TypeMeta: metav1.TypeMeta{
			Kind:       "Jaeger",
			APIVersion: "jaegertracing.io/v1",
		},
	}))
	assert.NoError(t, framework.AddToFrameworkScheme(apis.AddToScheme, &esv1.ElasticsearchList{
		TypeMeta: metav1.TypeMeta{
			Kind:       "Elasticsearch",
			APIVersion: "logging.openshift.io/v1",
		},
	}))

	var err error
	ctx, err = prepare(t)
	if (err != nil) {
		if ctx != nil {
			ctx.Cleanup()
		}
		require.FailNow(t, "Failed in prepare")
	}
	fw = framework.Global
	namespace, _ = ctx.GetNamespace()
	require.NotNil(t, namespace, "GetNamespace failed")
}

func (suite *SelfProvisionedTestSuite) TearDownSuite() {
	ctx.Cleanup()
}

func TestSelfProvisionedSuite(t *testing.T) {
	suite.Run(t, new(SelfProvisionedTestSuite))
}

func (suite *SelfProvisionedTestSuite) SetupTest() {
	t = suite.T()
}

func (suite *SelfProvisionedTestSuite) TestSelfProvisionedESSmokeTest() {
	// create jaeger custom resource
	exampleJaeger := getJaegerSimpleProd()
	err := fw.Client.Create(goctx.TODO(), exampleJaeger, &framework.CleanupOptions{TestContext: ctx, Timeout: timeout, RetryInterval: retryInterval})
	require.NoError(t, err, "Error deploying example Jaeger")
	defer undeployJaegerInstance(exampleJaeger)

	err = e2eutil.WaitForDeployment(t, fw.KubeClient, namespace, "simple-prod-collector", 1, retryInterval, timeout)
	require.NoError(t, err, "Error waiting for collector deployment")

	err = e2eutil.WaitForDeployment(t, fw.KubeClient, namespace, "simple-prod-query", 1, retryInterval, timeout)
	require.NoError(t, err, "Error waiting for query deployment")

	queryPort := randomPortNumber()
	queryPorts := []string{queryPort + ":16686"}
	portForw, closeChan := CreatePortForward(namespace, "simple-prod-query", "jaegertracing/jaeger-query", queryPorts, fw.KubeConfig)
	defer portForw.Close()
	defer close(closeChan)

	collectorPort := randomPortNumber()
	collectorPorts := []string{collectorPort + ":14268"}
	portForwColl, closeChanColl := CreatePortForward(namespace, "simple-prod-collector", "jaegertracing/jaeger-collector", collectorPorts, fw.KubeConfig)
	defer portForwColl.Close()
	defer close(closeChanColl)

	err = SmokeTest("http://localhost:" + queryPort + "/api/traces", "http://localhost:" + collectorPort + "/api/traces", "foobar", retryInterval, timeout)
	require.NoError(t, err, "Error running smoketest")
}

func getJaegerSimpleProd() *v1.Jaeger {
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
			},
		},
	}
	return exampleJaeger
}
