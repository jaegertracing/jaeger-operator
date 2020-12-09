// +build multiple

package e2e

import (
	"context"
	"testing"

	framework "github.com/operator-framework/operator-sdk/pkg/test"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type MultipleInstanceTestSuite struct {
	suite.Suite
}

func (suite *MultipleInstanceTestSuite) SetupSuite() {
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

func (suite *MultipleInstanceTestSuite) TearDownSuite() {
	handleSuiteTearDown()
}

func TestMultipleInstanceSuite(t *testing.T) {
	suite.Run(t, new(MultipleInstanceTestSuite))
}

func (suite *MultipleInstanceTestSuite) SetupTest() {
	t = suite.T()
}

func (suite *MultipleInstanceTestSuite) AfterTest(suiteName, testName string) {
	handleTestFailure()
}

/*
 * This test verifies that we create the elasticsearch secrets correctly if someone creates production Jaeger
 * instances with the same name in different namespaces
 */
func (suite *MultipleInstanceTestSuite) TestVerifySecrets() {
	if !isOpenShift(t) {
		t.Skip("This test is currently only supported on OpenShift")
	}

	jaegerInstanceName := "simple-prod"
	// In production we'd use 3 nodes but 1 is sufficient for this test.
	jaegerInstance := GetJaegerSelfProvSimpleProdCR(jaegerInstanceName, namespace, 1)
	createESSelfProvDeployment(jaegerInstance, jaegerInstanceName, namespace)
	defer undeployJaegerInstance(jaegerInstance)

	// Create a second instance with the same name but in a different namespace
	secondContext, err := createNewTestContext()
	defer secondContext.Cleanup()
	secondNamespace := secondContext.GetID()
	secondJaegerInstance := GetJaegerSelfProvSimpleProdCR(jaegerInstanceName, secondNamespace, 1)
	createESSelfProvDeployment(secondJaegerInstance, jaegerInstanceName, secondNamespace)
	defer undeployJaegerInstance(secondJaegerInstance)

	// Get the secrets from both and verify that the logging-es.crt values differ
	secretOne, err := fw.KubeClient.CoreV1().Secrets(namespace).Get(context.Background(), "elasticsearch", metav1.GetOptions{})
	require.NoError(t, err)
	loggingEsCrtOne := secretOne.Data["logging-es.crt"]
	require.NotNil(t, loggingEsCrtOne)

	secretTwo, err := fw.KubeClient.CoreV1().Secrets(secondNamespace).Get(context.Background(), "elasticsearch", metav1.GetOptions{})
	require.NoError(t, err)
	loggingEsCrtTwo := secretTwo.Data["logging-es.crt"]
	require.NotNil(t, loggingEsCrtTwo)

	require.NotEqual(t, string(loggingEsCrtOne), string(loggingEsCrtTwo))
}

func createNewTestContext() (*framework.Context, error) {
	secondContext, err := prepare(t)
	require.NoError(t, err, "Failed trying to create a new test context")

	return secondContext, err
}
