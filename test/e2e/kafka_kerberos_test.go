// +build kafka_kerberos

package e2e

import (
	"context"
	"fmt"
	"testing"
	"time"

	framework "github.com/operator-framework/operator-sdk/pkg/test"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	v1 "github.com/jaegertracing/jaeger-operator/pkg/apis/jaegertracing/v1"
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

func (suite *StreamingTestSuite) TestStreamingWithKerberos() {
	// Make sure ES and the kafka instance are available
	waitForElasticSearch()
	waitForKafkaInstance()
	jaegerInstanceNamespace := namespace

	kerberosConfigMap := getKerberosConfigMap(jaegerInstanceNamespace)
	err := fw.Client.Create(context.Background(), kerberosConfigMap, &framework.CleanupOptions{TestContext: ctx, Timeout: timeout, RetryInterval: retryInterval})
	require.NoError(t, err, "Error deploying kafkauser")
	defer func() {
		if !debugMode || !t.Failed() {
			err = fw.Client.Delete(context.TODO(), kerberosConfigMap)
			require.NoError(t, err)
		}
	}()

	jaegerInstanceName := "kerberos-streaming"

	jaegerInstance := jaegerStreamingDefinitionWithKerberos(jaegerInstanceNamespace, jaegerInstanceName)

	err = fw.Client.Create(context.TODO(), jaegerInstance, &framework.CleanupOptions{TestContext: ctx, Timeout: timeout, RetryInterval: retryInterval})

	require.NoError(t, err, "Error deploying jaeger")
	defer undeployJaegerInstance(jaegerInstance)

	err = WaitForDeployment(t, fw.KubeClient, jaegerInstanceNamespace, jaegerInstanceName+"-ingester", 1, retryInterval, timeout)
	require.NoError(t, err, "Error waiting for ingester deployment")

	err = WaitForDeployment(t, fw.KubeClient, jaegerInstanceNamespace, jaegerInstanceName+"-collector", 1, retryInterval, timeout)
	require.NoError(t, err, "Error waiting for collector deployment")

	err = WaitForDeployment(t, fw.KubeClient, jaegerInstanceNamespace, jaegerInstanceName+"-query", 1, retryInterval, timeout)
	require.NoError(t, err, "Error waiting for query deployment")

	ProductionSmokeTestWithNamespace(jaegerInstanceName, jaegerInstanceNamespace)
}

func jaegerStreamingDefinitionWithKerberos(namespace, name string) *v1.Jaeger {
	volumes := getKerberosConfigVolumes()
	volumeMounts := getKerberosConfigVolumeMounts()
	ingressEnabled := true

	kafkaClusterURL := fmt.Sprintf("kafka.%s.svc:9092", kafkaNamespace)
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
					"kafka.producer.authentication":       "kerberos",
					"kafka.producer.topic":                "jaeger-spans",
					"kafka.producer.brokers":              kafkaClusterURL,
					"kafka.producer.kerberos.config-file": "//kerberos-config/krb5.conf",
					"kafka.producer.kerberos.realm":       "EXAMPLE.COM",
					"kafka.producer.kerberos.username":    "collector/kafka.default.svc",
					"kafka.producer.kerberos.password":    "secret",
				}),
				JaegerCommonSpec: v1.JaegerCommonSpec{
					VolumeMounts: volumeMounts,
				},
			},
			Ingester: v1.JaegerIngesterSpec{
				Options: v1.NewOptions(map[string]interface{}{
					"kafka.consumer.authentication":       "kerberos",
					"kafka.consumer.topic":                "jaeger-spans",
					"kafka.consumer.brokers":              kafkaClusterURL,
					"kafka.consumer.kerberos.config-file": "//kerberos-config/krb5.conf",
					"kafka.consumer.kerberos.realm":       "EXAMPLE.COM",
					"kafka.consumer.kerberos.username":    "collector/kafka.default.svc",
					"kafka.consumer.kerberos.password":    "secret",
					"ingester.deadlockInterval":           0,
				}),
				JaegerCommonSpec: v1.JaegerCommonSpec{
					VolumeMounts: volumeMounts,
				},
			},
			Storage: v1.JaegerStorageSpec{
				Type: "elasticsearch",
				Options: v1.NewOptions(map[string]interface{}{
					"es.server-urls": esServerUrls,
				}),
			},
			JaegerCommonSpec: v1.JaegerCommonSpec{
				Volumes: volumes,
			},
		},
	}
	return j
}

func getKerberosConfigVolumeMounts() []corev1.VolumeMount {
	kafkaUserVolumeMount := corev1.VolumeMount{
		Name:      "config",
		MountPath: "/kerberos-config/",
	}

	volumeMounts := []corev1.VolumeMount{
		kafkaUserVolumeMount,
	}

	return volumeMounts
}

func getKerberosConfigMap(namespace string) *corev1.ConfigMap {
	data := map[string]string{
		"krb5-config": `
	[libdefaults]
		dns_lookup_realm = false
		ticket_lifetime = 24h
		renew_lifetime = 7d
		forwardable = true
		rdns = false
		default_realm = EXAMPLE.COM
		ignore_acceptor_hostname = true

	[realms]
		EXAMPLE.COM = {
		kdc = kafka.default.svc:8888
		admin_server = kafka.default.svc:8749
	}`,
	}

	return &corev1.ConfigMap{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "v1",
			Kind:       "ConfigMap",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      "kerberos-config",
			Namespace: namespace,
		},
		Data: data,
	}
}

func getKerberosConfigVolumes() []corev1.Volume {
	kerberosVolume := corev1.Volume{
		Name: "config",
		VolumeSource: corev1.VolumeSource{
			ConfigMap: &corev1.ConfigMapVolumeSource{
				LocalObjectReference: corev1.LocalObjectReference{
					Name: "kerberos-config",
				},
				Items: []corev1.KeyToPath{{
					Key:  "krb5-config",
					Path: "krb5.conf",
				}},
			},
		},
	}

	volumes := []corev1.Volume{
		kerberosVolume,
	}

	return volumes
}

func waitForKafkaInstance() {
	err := WaitForDeployment(t, fw.KubeClient, kafkaNamespace, "kafka", 1, retryInterval, timeout+1*time.Minute)
	require.NoError(t, err)
}

func waitForElasticSearch() {
	err := WaitForStatefulset(t, fw.KubeClient, storageNamespace, "elasticsearch", retryInterval, timeout)
	require.NoError(t, err, "Error waiting for elasticsearch")
}
