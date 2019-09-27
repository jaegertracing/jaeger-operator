// +build self_provisioned_elasticsearch

package e2e

import (
	goctx "context"
	"os"
	"testing"

	framework "github.com/operator-framework/operator-sdk/pkg/test"
	"github.com/operator-framework/operator-sdk/pkg/test/e2eutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"

	"github.com/jaegertracing/jaeger-operator/pkg/apis"
	"github.com/jaegertracing/jaeger-operator/pkg/apis/jaegertracing/v1"
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
	exampleJaeger := getJaegerSimpleProd()
	err := fw.Client.Create(goctx.TODO(), exampleJaeger, &framework.CleanupOptions{TestContext: ctx, Timeout: timeout, RetryInterval: retryInterval})
	require.NoError(t, err, "Error deploying example Jaeger")
	defer undeployJaegerInstance(exampleJaeger)

	err = e2eutil.WaitForDeployment(t, fw.KubeClient, namespace, "simple-prod-collector", 1, retryInterval, timeout)
	require.NoError(t, err, "Error waiting for collector deployment")

	err = e2eutil.WaitForDeployment(t, fw.KubeClient, namespace, "simple-prod-query", 1, retryInterval, timeout)
	require.NoError(t, err, "Error waiting for query deployment")

	ProductionSmokeTest("simple-prod")
}

func (suite *SelfProvisionedTestSuite) TestValidateEsOperatorImage() {
	expectedEsOperatorImage := os.Getenv("ES_OPERATOR_IMAGE")
	require.NotEmpty(t, expectedEsOperatorImage, "ES_OPERATOR_IMAGE must be defined")
	esOperatorNamespace := os.Getenv("ES_OPERATOR_NAMESPACE")
	require.NotEmpty(t, esOperatorNamespace, "ES_OPERATOR_NAMESPACE must be defined")

	imageName := getElasticSearchOperatorImage(fw.KubeClient, esOperatorNamespace)
	t.Logf("Using elasticsearch-operator image: %s\n", imageName)
	require.Equal(t, expectedEsOperatorImage, imageName)
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
	return exampleJaeger
}

func getElasticSearchOperatorImage(kubeclient kubernetes.Interface, namespace string) string {
	deployment, err := kubeclient.AppsV1().Deployments(namespace).Get("elasticsearch-operator", metav1.GetOptions{IncludeUninitialized: false})
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
