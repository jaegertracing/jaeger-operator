// +build self_provisioned_elasticsearch_kafka

package e2e

import (
	goctx "context"
	"testing"
	"time"

	framework "github.com/operator-framework/operator-sdk/pkg/test"
	"github.com/operator-framework/operator-sdk/pkg/test/e2eutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/jaegertracing/jaeger-operator/pkg/apis"
	v1 "github.com/jaegertracing/jaeger-operator/pkg/apis/jaegertracing/v1"
	kafkav1beta2 "github.com/jaegertracing/jaeger-operator/pkg/apis/kafka/v1beta2"
	esv1 "github.com/jaegertracing/jaeger-operator/pkg/storage/elasticsearch/v1"
)

type SelfProvisionedESWithKafkaTestSuite struct {
	suite.Suite
}

func (suite *SelfProvisionedESWithKafkaTestSuite) SetupSuite() {
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
	assert.NoError(t, framework.AddToFrameworkScheme(apis.AddToScheme, &kafkav1beta2.KafkaList{
		TypeMeta: metav1.TypeMeta{
			Kind:       "Kafka",
			APIVersion: "kafka.strimzi.io/v1beta2",
		},
	}))
	addToFrameworkSchemeForSmokeTests(t)

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
}

func (suite *SelfProvisionedESWithKafkaTestSuite) TearDownSuite() {
	handleSuiteTearDown()
}

func TestSelfProvisionedWithKafkaSuite(t *testing.T) {
	suite.Run(t, new(SelfProvisionedESWithKafkaTestSuite))
}

func (suite *SelfProvisionedESWithKafkaTestSuite) SetupTest() {
	t = suite.T()
}

func (suite *SelfProvisionedESWithKafkaTestSuite) AfterTest(suiteName, testName string) {
	handleTestFailure()
}

func (suite *SelfProvisionedESWithKafkaTestSuite) TestSelfProvisionedESAndKafkaSmokeTest() {
	// create jaeger custom resource
	jaegerInstanceName := "simple-prod"
	exampleJaeger := getJaegerSelfProvisionedESAndKafka(jaegerInstanceName)
	err := fw.Client.Create(goctx.TODO(), exampleJaeger, &framework.CleanupOptions{TestContext: ctx, Timeout: timeout, RetryInterval: retryInterval})
	require.NoError(t, err, "Error deploying example Jaeger")
	defer undeployJaegerInstance(exampleJaeger)
	defer deletePersistentVolumeClaims(namespace)

	err = WaitForStatefulset(t, fw.KubeClient, namespace, jaegerInstanceName+"-zookeeper", retryInterval, timeout+1*time.Minute)
	require.NoError(t, err)

	err = WaitForStatefulset(t, fw.KubeClient, namespace, jaegerInstanceName+"-kafka", retryInterval, timeout)
	require.NoError(t, err)

	err = WaitForDeployment(t, fw.KubeClient, namespace, jaegerInstanceName+"-entity-operator", 1, retryInterval, timeout)
	require.NoError(t, err, "Error waiting for entity-operator deployment")

	err = e2eutil.WaitForDeployment(t, fw.KubeClient, namespace, jaegerInstanceName+"-collector", 1, retryInterval, timeout)
	require.NoError(t, err, "Error waiting for collector deployment")

	err = e2eutil.WaitForDeployment(t, fw.KubeClient, namespace, jaegerInstanceName+"-query", 1, retryInterval, timeout)
	require.NoError(t, err, "Error waiting for query deployment")

	err = WaitForDeployment(t, fw.KubeClient, namespace, jaegerInstanceName+"-ingester", 1, retryInterval, timeout)
	require.NoError(t, err, "Error waiting for ingester deployment")

	ProductionSmokeTest(jaegerInstanceName)
}
