// +build streaming

package e2e

import (
	"context"
	"fmt"
	"testing"

	framework "github.com/operator-framework/operator-sdk/pkg/test"
	"github.com/operator-framework/operator-sdk/pkg/test/e2eutil"
	log "github.com/sirupsen/logrus"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/jaegertracing/jaeger-operator/pkg/apis/jaegertracing/v1"
)

type StreamingTestSuite struct {
	suite.Suite
}

func (suite *StreamingTestSuite) SetupSuite() {
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
	namespace, _ = ctx.GetNamespace()
	require.NotNil(t, namespace, "GetNamespace failed")

	addToFrameworkSchemeForSmokeTests(t)
}

func (suite *StreamingTestSuite) TearDownSuite() {
	log.Info("Entering TearDownSuite()")
	ctx.Cleanup()
}

func TestStreamingSuite(t *testing.T) {
	suite.Run(t, new(StreamingTestSuite))
}

func (suite *StreamingTestSuite) SetupTest() {
	t = suite.T()
}

func (suite *StreamingTestSuite) TestStreaming() {
	err := WaitForStatefulset(t, fw.KubeClient, storageNamespace, "elasticsearch", retryInterval, timeout)
	require.NoError(t, err, "Error waiting for elasticsearch")

	err = WaitForStatefulset(t, fw.KubeClient, kafkaNamespace, "my-cluster-kafka", retryInterval, timeout)
	require.NoError(t, err, "Error waiting for my-cluster-kafka")

	j := jaegerStreamingDefinition(namespace, "simple-streaming")
	log.Infof("passing %v", j)
	err = fw.Client.Create(context.TODO(), j, &framework.CleanupOptions{TestContext: ctx, Timeout: timeout, RetryInterval: retryInterval})
	require.NoError(t, err, "Error deploying jaeger")
	defer undeployJaegerInstance(j)

	err = e2eutil.WaitForDeployment(t, fw.KubeClient, namespace, "simple-streaming-collector", 1, retryInterval, timeout)
	require.NoError(t, err, "Error waiting for collector deployment")

	err = e2eutil.WaitForDeployment(t, fw.KubeClient, namespace, "simple-streaming-query", 1, retryInterval, timeout)
	require.NoError(t, err, "Error waiting for query deployment")

	ProductionSmokeTest("simple-streaming")
}

func jaegerStreamingDefinition(namespace string, name string) *v1.Jaeger {
	kafkaClusterURL := fmt.Sprintf("my-cluster-kafka-brokers.%s:9092", kafkaNamespace)
	j := &v1.Jaeger{
		TypeMeta: metav1.TypeMeta{
			Kind:       "Jaeger",
			APIVersion: "jaegertracing.io/v1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      "simple-streaming",
			Namespace: namespace,
		},
		Spec: v1.JaegerSpec{
			Strategy: "streaming",
			Collector: v1.JaegerCollectorSpec{
				Options: v1.NewOptions(map[string]interface{}{
					"kafka.producer.topic":   "jaeger-spans",
					"kafka.producer.brokers": kafkaClusterURL,
				}),
			},
			Ingester: v1.JaegerIngesterSpec{
				Options: v1.NewOptions(map[string]interface{}{
					"kafka.consumer.topic":   "jaeger-spans",
					"kafka.consumer.brokers": kafkaClusterURL,
				}),
			},
			Storage: v1.JaegerStorageSpec{
				Type: "elasticsearch",
				Options: v1.NewOptions(map[string]interface{}{
					"es.server-urls": esServerUrls,
				}),
			},
		},
	}
	return j
}
