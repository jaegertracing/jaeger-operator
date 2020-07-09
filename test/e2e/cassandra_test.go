// +build cassandra

package e2e

import (
	goctx "context"
	"testing"

	framework "github.com/operator-framework/operator-sdk/pkg/test"
	"github.com/operator-framework/operator-sdk/pkg/test/e2eutil"
	log "github.com/sirupsen/logrus"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	v1 "github.com/jaegertracing/jaeger-operator/pkg/apis/jaegertracing/v1"
)

type CassandraTestSuite struct {
	suite.Suite
}

func (suite *CassandraTestSuite) SetupSuite() {
	t = suite.T()
	if skipCassandraTests {
		t.Skip()
	}
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

	err = WaitForStatefulset(t, fw.KubeClient, storageNamespace, "cassandra", retryInterval, timeout)
	require.NoError(t, err, "Error waiting for cassandra stateful set")
}

func (suite *CassandraTestSuite) TearDownSuite() {
	handleSuiteTearDown()
}

func TestCassandraSuite(t *testing.T) {
	suite.Run(t, new(CassandraTestSuite))
}

func (suite *CassandraTestSuite) SetupTest() {
	t = suite.T()
}

func (suite *CassandraTestSuite) AfterTest(suiteName, testName string) {
	handleTestFailure()
}

// Cassandra runs a test with Cassandra as the backing storage
func (suite *CassandraTestSuite) TestCassandra() {
	waitForCassandra()

	jaegerInstanceName := "with-cassandra"
	cleanupOptions := &framework.CleanupOptions{TestContext: ctx, Timeout: timeout, RetryInterval: retryInterval}
	j := getJaegerWithCassandra(jaegerInstanceName, namespace)

	log.Infof("passing %v", j)
	err := fw.Client.Create(goctx.TODO(), j, cleanupOptions)
	require.NoError(t, err, "Error deploying jaeger")
	defer undeployJaegerInstance(j)

	err = WaitForJob(t, fw.KubeClient, namespace, jaegerInstanceName+"-cassandra-schema-job", retryInterval, timeout)
	require.NoError(t, err, "Error waiting for startup")

	err = e2eutil.WaitForDeployment(t, fw.KubeClient, namespace, jaegerInstanceName, 1, retryInterval, timeout)
	require.NoError(t, err, "Error waiting for deployment")

	AllInOneSmokeTest("with-cassandra")
}

func (suite *CassandraTestSuite) TestCassandraSparkDependencies() {
	storage := v1.JaegerStorageSpec{
		Type:                  "cassandra",
		Options:               v1.NewOptions(map[string]interface{}{"cassandra.servers": cassandraServiceName, "cassandra.keyspace": cassandraKeyspace}),
		CassandraCreateSchema: v1.JaegerCassandraCreateSchemaSpec{Datacenter: cassandraDatacenter, Mode: "prod"},
	}
	err := sparkTest(t, framework.Global, ctx, storage)
	require.NoError(t, err, "SparkTest failed")
}

func waitForCassandra() {
	err := WaitForStatefulset(t, fw.KubeClient, storageNamespace, "cassandra", retryInterval, timeout)
	require.NoError(t, err, "Error waiting for cassandra")
}

func getJaegerWithCassandra(jaegerInstanceName, namespace string) *v1.Jaeger {
	ingressEnabled := true
	j := &v1.Jaeger{
		TypeMeta: metav1.TypeMeta{
			Kind:       "Jaeger",
			APIVersion: "jaegertracing.io/v1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      jaegerInstanceName,
			Namespace: namespace,
		},
		Spec: v1.JaegerSpec{
			Ingress: v1.JaegerIngressSpec{
				Enabled:  &ingressEnabled,
				Security: v1.IngressSecurityNoneExplicit,
			},
			Strategy: v1.DeploymentStrategyAllInOne,
			Storage: v1.JaegerStorageSpec{
				Type:    "cassandra",
				Options: v1.NewOptions(map[string]interface{}{"cassandra.servers": cassandraServiceName, "cassandra.keyspace": cassandraKeyspace}),
				CassandraCreateSchema: v1.JaegerCassandraCreateSchemaSpec{
					Datacenter: cassandraDatacenter,
				},
			},
		},
	}
	return j
}
