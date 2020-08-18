// +build streaming

package e2e

import (
	"context"
	"fmt"
	"strings"
	"testing"
	"time"

	framework "github.com/operator-framework/operator-sdk/pkg/test"
	log "github.com/sirupsen/logrus"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/wait"

	v1 "github.com/jaegertracing/jaeger-operator/pkg/apis/jaegertracing/v1"
	kafkav1beta1 "github.com/jaegertracing/jaeger-operator/pkg/apis/kafka/v1beta1"
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
	waitForElasticSearch()
	waitForKafkaInstance()

	jaegerInstanceName := "simple-streaming"
	j := jaegerStreamingDefinition(namespace, jaegerInstanceName, testOtelCollector)
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

	// Make sure we were using the correct collector image
	verifyCollectorImage(jaegerInstanceName, namespace, testOtelCollector)
}

func (suite *StreamingTestSuite) TestStreamingWithTLS() {
	if !usingJaegerViaOLM {
		t.Skip("This test should only run when using OLM")
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
	jaegerInstance := jaegerStreamingDefinitionWithTLS(kafkaNamespace, jaegerInstanceName, kafkaUserName, testOtelCollector)
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

	// Make sure we were using the correct collector image
	verifyCollectorImage(jaegerInstanceName, kafkaNamespace, testOtelCollector)
}

func (suite *StreamingTestSuite) TestStreamingWithAutoProvisioning() {
	// Make sure ES instance is available
	waitForElasticSearch()

	// Now create a jaeger instance which will auto provision a kafka instance
	jaegerInstanceName := "auto-provisioned"
	jaegerInstanceNamespace := namespace
	jaegerInstance := jaegerAutoProvisionedDefinition(jaegerInstanceNamespace, jaegerInstanceName, testOtelCollector)
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

	// Make sure we were using the correct collector image
	verifyCollectorImage(jaegerInstanceName, namespace, testOtelCollector)
}

func jaegerStreamingDefinition(namespace string, name string, useOtelCollector bool) *v1.Jaeger {
	kafkaClusterURL := fmt.Sprintf("my-cluster-kafka-brokers.%s:9092", kafkaNamespace)
	ingressEnabled := true
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
					"kafka.producer.topic":   "jaeger-spans",
					"kafka.producer.brokers": kafkaClusterURL,
					// The following 3 flags are not required for this test, but are added to ensure we passed them correctly
					"kafka.producer.batch-linger":       "1s",
					"kafka.producer.batch-size":         "128000",
					"kafka.producer.batch-max-messages": "100",
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

	if useOtelCollector {
		log.Infof("Using OTEL collector for %s", name)
		j.Spec.Collector.Image = otelCollectorImage
		j.Spec.Collector.Config = v1.NewFreeForm(getOtelCollectorOptions())
	}

	return j
}

func jaegerStreamingDefinitionWithTLS(namespace string, name, kafkaUserName string, useOtelCollector bool) *v1.Jaeger {
	volumes := getTLSVolumes(kafkaUserName)
	volumeMounts := getTLSVolumeMounts()
	ingressEnabled := true

	kafkaClusterURL := fmt.Sprintf("my-cluster-kafka-bootstrap.%s.svc.cluster.local:9093", kafkaNamespace)
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
				Options: v1.NewOptions(map[string]interface{}{
					"kafka.consumer.authentication": "tls",
					"kafka.consumer.topic":          "jaeger-spans",
					"kafka.consumer.brokers":        kafkaClusterURL,
					"kafka.consumer.tls.ca":         "/var/run/secrets/cluster-ca/ca.crt",
					"kafka.consumer.tls.cert":       "/var/run/secrets/kafkauser/user.crt",
					"kafka.consumer.tls.key":        "/var/run/secrets/kafkauser/user.key",
					"ingester.deadlockInterval":     0,
				}),
			},
			Storage: v1.JaegerStorageSpec{
				Type: "elasticsearch",
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

	if useOtelCollector {
		log.Infof("Using OTEL collector for %s", name)
		j.Spec.Collector.Image = otelCollectorImage
		j.Spec.Collector.Config = v1.NewFreeForm(getOtelCollectorOptions())
	}

	return j
}

func jaegerAutoProvisionedDefinition(namespace string, name string, useOtelCollector bool) *v1.Jaeger {
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
				Type: "elasticsearch",
				Options: v1.NewOptions(map[string]interface{}{
					"es.server-urls": esServerUrls,
				}),
			},
		},
	}

	if useOtelCollector {
		log.Infof("Using OTEL collector for %s", name)
		jaegerInstance.Spec.Collector.Image = otelCollectorImage
		jaegerInstance.Spec.Collector.Config = v1.NewFreeForm(getOtelCollectorOptions())
	}

	return jaegerInstance
}

func getKafkaUser(name, namespace string) *kafkav1beta1.KafkaUser {
	kafkaUser := &kafkav1beta1.KafkaUser{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "kafka.strimzi.io/v1beta1",
			Kind:       "KafkaUser",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
			Labels: map[string]string{
				"strimzi.io/cluster": "my-cluster",
			},
		},
		Spec: kafkav1beta1.KafkaUserSpec{
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

func waitForKafkaInstance() {
	kafkaInstance := &kafkav1beta1.Kafka{}

	err := WaitForStatefulset(t, fw.KubeClient, kafkaNamespace, "my-cluster-zookeeper", retryInterval, timeout+1*time.Minute)
	require.NoError(t, err)

	err = WaitForStatefulset(t, fw.KubeClient, kafkaNamespace, "my-cluster-kafka", retryInterval, timeout)
	require.NoError(t, err)

	err = wait.Poll(retryInterval, timeout, func() (done bool, err error) {
		err = fw.Client.Get(context.Background(), types.NamespacedName{Name: "my-cluster", Namespace: kafkaNamespace}, kafkaInstance)
		require.NoError(t, err)

		for _, condition := range kafkaInstance.Status.Conditions {
			if strings.EqualFold(condition.Type, "ready") && strings.EqualFold(condition.Status, "true") {
				return true, nil
			}
		}

		return false, nil
	})
	require.NoError(t, err)
}

func waitForElasticSearch() {
	err := WaitForStatefulset(t, fw.KubeClient, storageNamespace, "elasticsearch", retryInterval, timeout)
	require.NoError(t, err, "Error waiting for elasticsearch")
}
