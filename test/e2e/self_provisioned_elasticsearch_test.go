// +build self_provisioned_elasticsearch

package e2e

import (
	"context"
	goctx "context"
	"fmt"
	"os"
	"strings"
	"testing"

	framework "github.com/operator-framework/operator-sdk/pkg/test"
	"github.com/operator-framework/operator-sdk/pkg/test/e2eutil"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/kubernetes"

	"github.com/jaegertracing/jaeger-operator/pkg/apis"
	v1 "github.com/jaegertracing/jaeger-operator/pkg/apis/jaegertracing/v1"
	esv1 "github.com/jaegertracing/jaeger-operator/pkg/storage/elasticsearch/v1"
)

type SelfProvisionedTestSuite struct {
	suite.Suite
}

func (suite *SelfProvisionedTestSuite) SetupSuite() {
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

func (suite *SelfProvisionedTestSuite) TearDownSuite() {
	handleSuiteTearDown()
}

func TestSelfProvisionedSuite(t *testing.T) {
	suite.Run(t, new(SelfProvisionedTestSuite))
}

func (suite *SelfProvisionedTestSuite) SetupTest() {
	t = suite.T()
}

func (suite *SelfProvisionedTestSuite) AfterTest(suiteName, testName string) {
	handleTestFailure()
}

func (suite *SelfProvisionedTestSuite) TestSelfProvisionedESSmokeTest() {
	// create jaeger custom resource
	jaegerInstanceName := "simple-prod"
	exampleJaeger := getJaegerSimpleProd(jaegerInstanceName, testOtelCollector)
	err := fw.Client.Create(goctx.TODO(), exampleJaeger, &framework.CleanupOptions{TestContext: ctx, Timeout: timeout, RetryInterval: retryInterval})
	require.NoError(t, err, "Error deploying example Jaeger")
	defer undeployJaegerInstance(exampleJaeger)

	err = e2eutil.WaitForDeployment(t, fw.KubeClient, namespace, jaegerInstanceName+"-collector", 1, retryInterval, timeout)
	require.NoError(t, err, "Error waiting for collector deployment")

	err = e2eutil.WaitForDeployment(t, fw.KubeClient, namespace, jaegerInstanceName+"-query", 1, retryInterval, timeout)
	require.NoError(t, err, "Error waiting for query deployment")

	ProductionSmokeTest(jaegerInstanceName)

	// Make sure we were using the correct collector image
	verifyCollectorImage(jaegerInstanceName, namespace, testOtelCollector)
}

func (suite *SelfProvisionedTestSuite) TestIncreasingReplicas() {
	jaegerInstanceName := "simple-prod2"
	exampleJaeger := getJaegerSimpleProd(jaegerInstanceName, testOtelCollector)
	err := fw.Client.Create(goctx.TODO(), exampleJaeger, &framework.CleanupOptions{TestContext: ctx, Timeout: timeout, RetryInterval: retryInterval})
	require.NoError(t, err, "Error deploying example Jaeger")
	defer undeployJaegerInstance(exampleJaeger)

	err = e2eutil.WaitForDeployment(t, fw.KubeClient, namespace, jaegerInstanceName+"-collector", 1, retryInterval, timeout)
	require.NoError(t, err, "Error waiting for collector deployment")

	err = e2eutil.WaitForDeployment(t, fw.KubeClient, namespace, jaegerInstanceName+"-query", 1, retryInterval, timeout)
	require.NoError(t, err, "Error waiting for query deployment")

	ProductionSmokeTest(jaegerInstanceName)

	updateESNodeCount := 2
	updateCollectorCount := int32(2)
	updateQueryCount := int32(2)

	changeNodeCount(jaegerInstanceName, updateESNodeCount, updateCollectorCount, updateQueryCount)
	updatedJaegerInstance := getJaegerInstance(jaegerInstanceName, namespace)
	require.EqualValues(t, updateESNodeCount, updatedJaegerInstance.Spec.Storage.Elasticsearch.NodeCount)
	require.EqualValues(t, updateCollectorCount, *updatedJaegerInstance.Spec.Collector.Replicas)
	require.EqualValues(t, updateQueryCount, *updatedJaegerInstance.Spec.Query.Replicas)

	err = e2eutil.WaitForDeployment(t, fw.KubeClient, namespace, jaegerInstanceName+"-collector", int(updateCollectorCount), retryInterval, timeout)
	require.NoError(t, err, "Error waiting for collector deployment")

	err = e2eutil.WaitForDeployment(t, fw.KubeClient, namespace, jaegerInstanceName+"-query", int(updateQueryCount), retryInterval, timeout)
	require.NoError(t, err, "Error waiting for query deployment")

	// wait for second ES node to come up
	err = e2eutil.WaitForDeployment(t, fw.KubeClient, namespace, esDeploymentName(namespace, jaegerInstanceName, 2), 1, retryInterval, timeout)
	require.NoError(t, err, "Error waiting for Elasticsearch deployment")

	// Make sure there are 2 ES deployments and wait for them to be available
	listOptions := metav1.ListOptions{
		LabelSelector: "component=elasticsearch",
	}

	deployments, err := fw.KubeClient.AppsV1().Deployments(namespace).List(context.Background(), listOptions)
	require.NoError(t, err)
	require.Equal(t, updateESNodeCount, len(deployments.Items))
	for _, deployment := range deployments.Items {
		if deployment.Namespace == namespace {
			logrus.Infof("Looking for deployment %s with annotations %v", deployment.Name, deployment.Annotations)
			err = e2eutil.WaitForDeployment(t, fw.KubeClient, namespace, deployment.Name, 1, retryInterval, timeout)
			require.NoError(t, err, "Error waiting for deployment: "+deployment.Name)
		}
	}

	/// Verify the number of Collector and Query pods
	var collectorPodCount int32
	var queryPodCount int32

	// Wait until pod counts equalize, otherwise we risk counting or port forwarding to a terminating pod
	err = wait.Poll(retryInterval, timeout, func() (done bool, err error) {
		collectorPodCount = 0
		queryPodCount = 0
		pods, err := fw.KubeClient.CoreV1().Pods(namespace).List(context.Background(), metav1.ListOptions{})
		require.NoError(t, err)

		for _, pod := range pods.Items {
			if strings.HasPrefix(pod.Name, jaegerInstanceName+"-collector") {
				collectorPodCount++
			} else if strings.HasPrefix(pod.Name, jaegerInstanceName+"-query") {
				queryPodCount++
			}
		}

		if queryPodCount == updateQueryCount && collectorPodCount == updateCollectorCount {
			return true, nil
		} else {
			return false, nil
		}
	})
	require.EqualValues(t, updateCollectorCount, collectorPodCount)
	require.EqualValues(t, updateQueryCount, queryPodCount)
	require.NoError(t, err)

	ProductionSmokeTest(jaegerInstanceName)

	// Make sure we were using the correct collector image
	verifyCollectorImage(jaegerInstanceName, namespace, testOtelCollector)
}

func esDeploymentName(ns, jaegerName string, instances int) string {
	nsAndName := strings.ReplaceAll(ns, "-", "") + strings.ReplaceAll(jaegerName, "-", "")
	return fmt.Sprintf("elasticsearch-cdm-%s-%d", nsAndName[:36], instances)
}

func changeNodeCount(name string, newESNodeCount int, newCollectorNodeCount, newQueryNodeCount int32) {
	jaegerInstance := getJaegerInstance(name, namespace)
	jaegerInstance.Spec.Collector.Replicas = &newCollectorNodeCount
	jaegerInstance.Spec.Query.Replicas = &newQueryNodeCount
	jaegerInstance.Spec.Storage.Elasticsearch.NodeCount = int32(newESNodeCount)
	err := fw.Client.Update(context.Background(), jaegerInstance)
	require.NoError(t, err)
}

func (suite *SelfProvisionedTestSuite) TestValidateEsOperatorImage() {
	// TODO reinstate this if we come up with a good solution, but skip for now when using OLM installed operators
	if usingOLM {
		t.Skip()
	}
	expectedEsOperatorImage := os.Getenv("ES_OPERATOR_IMAGE")
	require.NotEmpty(t, expectedEsOperatorImage, "ES_OPERATOR_IMAGE must be defined")
	esOperatorNamespace := os.Getenv("ES_OPERATOR_NAMESPACE")
	require.NotEmpty(t, esOperatorNamespace, "ES_OPERATOR_NAMESPACE must be defined")

	imageName := getElasticSearchOperatorImage(fw.KubeClient, esOperatorNamespace)
	t.Logf("Using elasticsearch-operator image: %s\n", imageName)
	require.Equal(t, expectedEsOperatorImage, imageName)
}

func getJaegerSimpleProd(instanceName string, useOtelCollector bool) *v1.Jaeger {
	ingressEnabled := true
	exampleJaeger := &v1.Jaeger{
		TypeMeta: metav1.TypeMeta{
			Kind:       "Jaeger",
			APIVersion: "jaegertracing.io/v1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      instanceName,
			Namespace: namespace,
		},
		Spec: v1.JaegerSpec{
			Ingress: v1.JaegerIngressSpec{
				Enabled:  &ingressEnabled,
				Security: v1.IngressSecurityNoneExplicit,
			},
			Strategy: v1.DeploymentStrategyProduction,
			Storage: v1.JaegerStorageSpec{
				Type: "elasticsearch",
				Elasticsearch: v1.ElasticsearchSpec{
					NodeCount: 1,
					Resources: &corev1.ResourceRequirements{
						Limits:   corev1.ResourceList{corev1.ResourceMemory: resource.MustParse("1Gi")},
						Requests: corev1.ResourceList{corev1.ResourceMemory: resource.MustParse("1Gi")},
					},
				},
			},
		},
	}

	if useOtelCollector {
		logrus.Infof("Using OTEL collector for %s", instanceName)
		exampleJaeger.Spec.Collector.Image = otelCollectorImage
		exampleJaeger.Spec.Collector.Config = v1.NewFreeForm(getOtelCollectorOptions())
	}

	return exampleJaeger
}

func getElasticSearchOperatorImage(kubeclient kubernetes.Interface, namespace string) string {
	deployment, err := kubeclient.AppsV1().Deployments(namespace).Get(context.Background(), "elasticsearch-operator", metav1.GetOptions{})
	require.NoErrorf(t, err, "Did not find elasticsearch-operator in namespace %s\n", namespace)

	containers := deployment.Spec.Template.Spec.Containers
	for _, container := range containers {
		if container.Name == "elasticsearch-operator" {
			return container.Image
		}
	}

	require.FailNowf(t, "Did not find elasticsearch-operator in namespace %s\n", namespace)
	return ""
}
