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

	"github.com/jaegertracing/jaeger-operator/pkg/apis/jaegertracing/v1"
)

type CassandraTestSuite struct {
	suite.Suite
}

func(suite *CassandraTestSuite) SetupSuite() {
	t = suite.T()
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

	addToFrameworkSchemeForSmokeTests(t)
}

func (suite *CassandraTestSuite) TearDownSuite() {
	log.Info("Entering TearDownSuite()")
	ctx.Cleanup()
}

func TestCassandraSuite(t *testing.T) {
	suite.Run(t, new(CassandraTestSuite))
}

func (suite *CassandraTestSuite) SetupTest() {
	t = suite.T()
}

// Cassandra runs a test with Cassandra as the backing storage
func (suite *CassandraTestSuite) TestCassandra()  {
	cleanupOptions := &framework.CleanupOptions{TestContext: ctx, Timeout: timeout, RetryInterval: retryInterval}
	j := getJaegerWithCassandra(namespace)

	log.Infof("passing %v", j)
	err := fw.Client.Create(goctx.TODO(), j, cleanupOptions)
	require.NoError(t, err, "Error deploying jaeger")
	defer undeployJaegerInstance(j)

	err = WaitForJob(t, fw.KubeClient, namespace, "with-cassandra-cassandra-schema-job", retryInterval, timeout)
	require.NoError(t, err, "Error waiting for startup")

	err = e2eutil.WaitForDeployment(t, fw.KubeClient, namespace, "with-cassandra", 1, retryInterval, timeout)
	require.NoError(t, err, "Error waiting for deployment")

	portForw, closeChan := CreatePortForward(namespace, "with-cassandra", "jaegertracing/all-in-one", []string{"16686", "14268"}, fw.KubeConfig)
	defer portForw.Close()
	defer close(closeChan)

	err = SmokeTest("http://localhost:16686/api/traces", "http://localhost:14268/api/traces", "foobar", retryInterval, timeout)
	require.NoError(t, err, "SmokeTest Failed")
}

func (suite *CassandraTestSuite) TestCassandraSparkDependencies()  {
	storage := v1.JaegerStorageSpec{
		Type: "cassandra",
		Options: v1.NewOptions(map[string]interface{}{"cassandra.servers": cassandraServiceName, "cassandra.keyspace": "jaeger_v1_datacenter1"}),
		CassandraCreateSchema:v1.JaegerCassandraCreateSchemaSpec{Datacenter:"datacenter1", Mode: "prod"},
	}
	err := sparkTest(t, framework.Global, ctx, storage)
	require.NoError(t, err, "SparkTest failed")
}

func getJaegerWithCassandra(s string) *v1.Jaeger {
	j := &v1.Jaeger{
		TypeMeta: metav1.TypeMeta{
			Kind:       "Jaeger",
			APIVersion: "jaegertracing.io/v1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      "with-cassandra",
			Namespace: namespace,
		},
		Spec: v1.JaegerSpec{
			Strategy: "allInOne",
			Storage: v1.JaegerStorageSpec{
				Type:    "cassandra",
				Options: v1.NewOptions(map[string]interface{}{"cassandra.servers": cassandraServiceName, "cassandra.keyspace": "jaeger_v1_datacenter1"}),
				CassandraCreateSchema: v1.JaegerCassandraCreateSchemaSpec{
					Datacenter: "datacenter1",
				},
			},
		},
	}
	return j
}
