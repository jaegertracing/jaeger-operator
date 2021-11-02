// +build streaming

package e2e

import (
	"context"
	"fmt"
	"testing"
	"time"

	framework "github.com/operator-framework/operator-sdk/pkg/test"
	log "github.com/sirupsen/logrus"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	v1 "github.com/jaegertracing/jaeger-operator/pkg/apis/jaegertracing/v1"
	kafkav1beta2 "github.com/jaegertracing/jaeger-operator/pkg/apis/kafka/v1beta2"
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
	namespace = ctx.GetID()
	require.NotNil(t, namespace, "GetID failed")

	addToFrameworkSchemeForSmokeTests(t)
}

func (suite *StreamingTestSuite) TearDownSuite() {
	handleSuiteTearDown()
}

func TestStreamingSuite(t *testing.T) {
	suite.Run(t, new(StreamingTestSuite))
}

func (suite *StreamingTestSuite) SetupTest() {
	t = suite.T()
}

func (suite *StreamingTestSuite) AfterTest(suiteName, testName string) {
	handleTestFailure()
}

func (suite *StreamingTestSuite) TestStreaming() {
	if skipESExternal {
		t.Skip("This case is covered by the self_provisioned_elasticsearch_kafka_test")
	}
	waitForElasticSearch()
	waitForKafkaInstance()

	jaegerInstanceName := "simple-streaming"
	j := jaegerStreamingDefinition(namespace, jaegerInstanceName)
	log.Infof("passing %v", j)
	err := fw.Client.Create(context.TODO(), j, &framework.CleanupOptions{TestContext: ctx, Timeout: timeout, RetryInterval: retryInterval})
	require.NoError(t, err, "Error deploying jaeger")
	defer undeployJaegerInstance(j)

	err = WaitForDeployment(t, fw.KubeClient, namespace, jaegerInstanceName+"-ingester", 1, retryInterval, timeout)
	require.NoError(t, err, "Error waiting for ingester deployment")

	err = WaitForDeployment(t, fw.KubeClient, namespace, jaegerInstanceName+"-collector", 1, retryInterval, timeout)
	require.NoError(t, err, "Error waiting for collector deployment")

	err = WaitForDeployment(t, fw.KubeClient, namespace, jaegerInstanceName+"-query", 1, retryInterval, timeout)
	require.NoError(t, err, "Error waiting for query deployment")

	ProductionSmokeTest(jaegerInstanceName)
}

func (suite *StreamingTestSuite) TestStreamingWithTLS() {
	if !usingJaegerViaOLM {
		t.Skip("This test should only run when using OLM")
	}
	if skipESExternal {
		t.Skip()
	}
	// Make sure ES and the kafka instance are available
	waitForElasticSearch()
	waitForKafkaInstance()

	kafkaUserName := "my-user"
	kafkaUser := getKafkaUser(kafkaUserName, kafkaNamespace)
	err := fw.Client.Create(context.Background(), kafkaUser, &framework.CleanupOptions{TestContext: ctx, Timeout: timeout, RetryInterval: retryInterval})
	require.NoError(t, err, "Error deploying kafkauser")
	WaitForSecret(kafkaUserName, kafkaNamespace)

	defer func() {
		if !debugMode || !t.Failed() {
			err = fw.Client.Delete(context.TODO(), kafkaUser)
			require.NoError(t, err)
		}
	}()

	// Now create a jaeger instance with TLS enabled -- note it has to be deployed in the same namespace as the kafka instance
	jaegerInstanceName := "tls-streaming"
	jaegerInstance := jaegerStreamingDefinitionWithTLS(kafkaNamespace, jaegerInstanceName, kafkaUserName)
	err = fw.Client.Create(context.TODO(), jaegerInstance, &framework.CleanupOptions{TestContext: ctx, Timeout: timeout, RetryInterval: retryInterval})
	require.NoError(t, err, "Error deploying jaeger")
	defer undeployJaegerInstance(jaegerInstance)

	err = WaitForDeployment(t, fw.KubeClient, kafkaNamespace, jaegerInstanceName+"-ingester", 1, retryInterval, timeout)
	require.NoError(t, err, "Error waiting for ingester deployment")

	err = WaitForDeployment(t, fw.KubeClient, kafkaNamespace, jaegerInstanceName+"-collector", 1, retryInterval, timeout)
	require.NoError(t, err, "Error waiting for collector deployment")

	err = WaitForDeployment(t, fw.KubeClient, kafkaNamespace, jaegerInstanceName+"-query", 1, retryInterval, timeout)
	require.NoError(t, err, "Error waiting for query deployment")

	ProductionSmokeTestWithNamespace(jaegerInstanceName, kafkaNamespace)
}

func (suite *StreamingTestSuite) TestStreamingWithAutoProvisioning() {
	if skipESExternal {
		t.Skip("This case is covered by the self_provisioned_elasticsearch_kafka_test")
	}
	// Make sure ES instance is available
	waitForElasticSearch()

	// Now create a jaeger instance which will auto provision a kafka instance
	jaegerInstanceName := "auto-provisioned"
	jaegerInstanceNamespace := namespace
	jaegerInstance := jaegerAutoProvisionedDefinition(jaegerInstanceNamespace, jaegerInstanceName)
	err := fw.Client.Create(context.TODO(), jaegerInstance, &framework.CleanupOptions{TestContext: ctx, Timeout: timeout, RetryInterval: retryInterval})
	require.NoError(t, err, "Error deploying jaeger")
	defer undeployJaegerInstance(jaegerInstance)
	defer deletePersistentVolumeClaims(namespace)

	for _, n := range []string{"zookeeper", "kafka"} {
		depName := fmt.Sprintf("%s-%s", jaegerInstanceName, n)
		err = WaitForStatefulset(t, fw.KubeClient, namespace, depName, retryInterval, timeout+1*time.Minute)
		require.NoError(t, err, fmt.Sprintf("Error waiting for statefulset: %s", depName))
	}

	for _, n := range []string{"entity-operator", "ingester", "collector", "query"} {
		depName := fmt.Sprintf("%s-%s", jaegerInstanceName, n)
		err = WaitForDeployment(t, fw.KubeClient, jaegerInstanceNamespace, depName, 1, retryInterval, timeout)
		require.NoError(t, err, fmt.Sprintf("Error waiting for deployment: %s", depName))
	}

	ProductionSmokeTestWithNamespace(jaegerInstanceName, jaegerInstanceNamespace)
}

func jaegerStreamingDefinition(namespace string, name string) *v1.Jaeger {
	kafkaClusterURL := fmt.Sprintf("my-cluster-kafka-brokers.%s:9092", kafkaNamespace)
	ingressEnabled := true
	collectorOptions := make(map[string]interface{})
	collectorOptions["kafka.producer.topic"] = "jaeger-spans"
	collectorOptions["kafka.producer.brokers"] = kafkaClusterURL

	j := &v1.Jaeger{
		TypeMeta: metav1.TypeMeta{
			Kind:       "Jaeger",
			APIVersion: "jaegertracing.io/v1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
		},
		Spec: v1.JaegerSpec{
			Ingress: v1.JaegerIngressSpec{
				Enabled:  &ingressEnabled,
				Security: v1.IngressSecurityNoneExplicit,
			},
			Strategy: v1.DeploymentStrategyStreaming,
			Collector: v1.JaegerCollectorSpec{
				Options: v1.NewOptions(collectorOptions),
			},
			Ingester: v1.JaegerIngesterSpec{
				Options: v1.NewOptions(map[string]interface{}{
					"kafka.consumer.topic":   "jaeger-spans",
					"kafka.consumer.brokers": kafkaClusterURL,
				}),
			},
			Storage: v1.JaegerStorageSpec{
				Type: v1.JaegerESStorage,
				Options: v1.NewOptions(map[string]interface{}{
					"es.server-urls": esServerUrls,
				}),
			},
		},
	}

	return j
}

func jaegerStreamingDefinitionWithTLS(namespace string, name, kafkaUserName string) *v1.Jaeger {
	volumes := getTLSVolumes(kafkaUserName)
	volumeMounts := getTLSVolumeMounts()
	ingressEnabled := true

	kafkaClusterURL := fmt.Sprintf("my-cluster-kafka-bootstrap.%s.svc.cluster.local:9093", kafkaNamespace)
	ingesterOptions := make(map[string]interface{})
	ingesterOptions["kafka.consumer.authentication"] = "tls"
	ingesterOptions["kafka.consumer.topic"] = "jaeger-spans"
	ingesterOptions["kafka.consumer.brokers"] = kafkaClusterURL
	ingesterOptions["kafka.consumer.tls.ca"] = "/var/run/secrets/cluster-ca/ca.crt"
	ingesterOptions["kafka.consumer.tls.cert"] = "/var/run/secrets/kafkauser/user.crt"
	ingesterOptions["kafka.consumer.tls.key"] = "/var/run/secrets/kafkauser/user.key"

	j := &v1.Jaeger{
		TypeMeta: metav1.TypeMeta{
			Kind:       "Jaeger",
			APIVersion: "jaegertracing.io/v1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
		},
		Spec: v1.JaegerSpec{
			Ingress: v1.JaegerIngressSpec{
				Enabled:  &ingressEnabled,
				Security: v1.IngressSecurityNoneExplicit,
			},
			Strategy: v1.DeploymentStrategyStreaming,
			Collector: v1.JaegerCollectorSpec{
				Options: v1.NewOptions(map[string]interface{}{
					"kafka.producer.authentication": "tls",
					"kafka.producer.topic":          "jaeger-spans",
					"kafka.producer.brokers":        kafkaClusterURL,
					"kafka.producer.tls.ca":         "/var/run/secrets/cluster-ca/ca.crt",
					"kafka.producer.tls.cert":       "/var/run/secrets/kafkauser/user.crt",
					"kafka.producer.tls.key":        "/var/run/secrets/kafkauser/user.key",
				}),
			},
			Ingester: v1.JaegerIngesterSpec{
				Options: v1.NewOptions(ingesterOptions),
			},
			Storage: v1.JaegerStorageSpec{
				Type: v1.JaegerESStorage,
				Options: v1.NewOptions(map[string]interface{}{
					"es.server-urls": esServerUrls,
				}),
			},
			JaegerCommonSpec: v1.JaegerCommonSpec{
				Volumes:      volumes,
				VolumeMounts: volumeMounts,
			},
		},
	}

	return j
}

func jaegerAutoProvisionedDefinition(namespace string, name string) *v1.Jaeger {
	ingressEnabled := true
	jaegerInstance := &v1.Jaeger{
		TypeMeta: metav1.TypeMeta{
			Kind:       "Jaeger",
			APIVersion: "jaegertracing.io/v1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
		},
		Spec: v1.JaegerSpec{
			Ingress: v1.JaegerIngressSpec{
				Enabled:  &ingressEnabled,
				Security: v1.IngressSecurityNoneExplicit,
			},
			Strategy: v1.DeploymentStrategyStreaming,
			Storage: v1.JaegerStorageSpec{
				Type: v1.JaegerESStorage,
				Options: v1.NewOptions(map[string]interface{}{
					"es.server-urls": esServerUrls,
				}),
			},
		},
	}

	return jaegerInstance
}

func getKafkaUser(name, namespace string) *kafkav1beta2.KafkaUser {
	kafkaUser := &kafkav1beta2.KafkaUser{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "kafka.strimzi.io/v1beta2",
			Kind:       "KafkaUser",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
			Labels: map[string]string{
				"strimzi.io/cluster": "my-cluster",
			},
		},
		Spec: kafkav1beta2.KafkaUserSpec{
			v1.NewFreeForm(map[string]interface{}{
				"authentication": map[string]interface{}{
					"type": "tls",
				},
			}),
		},
	}
	return kafkaUser
}

func getTLSVolumeMounts() []corev1.VolumeMount {
	kafkaUserVolumeMount := corev1.VolumeMount{
		Name:      "kafkauser",
		MountPath: "/var/run/secrets/kafkauser",
	}
	clusterCaVolumeMount := corev1.VolumeMount{
		Name:      "cluster-ca",
		MountPath: "/var/run/secrets/cluster-ca",
	}

	volumeMounts := []corev1.VolumeMount{
		kafkaUserVolumeMount, clusterCaVolumeMount,
	}

	return volumeMounts
}

func getTLSVolumes(kafkaUserName string) []corev1.Volume {
	kafkaUserSecretName := corev1.SecretVolumeSource{
		SecretName: kafkaUserName,
	}
	clusterCaSecretName := corev1.SecretVolumeSource{
		SecretName: "my-cluster-cluster-ca-cert",
	}

	kafkaUserVolume := corev1.Volume{
		Name: "kafkauser",
		VolumeSource: corev1.VolumeSource{
			Secret: &kafkaUserSecretName,
		},
	}
	clusterCaVolume := corev1.Volume{
		Name: "cluster-ca",
		VolumeSource: corev1.VolumeSource{
			Secret: &clusterCaSecretName,
		},
	}

	volumes := []corev1.Volume{
		kafkaUserVolume,
		clusterCaVolume,
	}

	return volumes
}
