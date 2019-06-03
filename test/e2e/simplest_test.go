// +build smoke

package e2e

import (
	goctx "context"
	"testing"

	"github.com/stretchr/testify/suite"
	framework "github.com/operator-framework/operator-sdk/pkg/test"
	"github.com/operator-framework/operator-sdk/pkg/test/e2eutil"
	"github.com/stretchr/testify/require"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/jaegertracing/jaeger-operator/pkg/apis/jaegertracing/v1"
)

type SimplestTestSuite struct {
	suite.Suite
}

func (suite *SimplestTestSuite) SetupSuite() {
	t = suite.T()
	var err error
	ctx, err = prepare(t)
	if (err != nil) {
		if ctx != nil {
			ctx.Cleanup()
		}
		require.FailNow(t, "Failed in prepare with: " + err.Error())
	}
	fw = framework.Global
	namespace, _ = ctx.GetNamespace()
	require.NotNil(t, namespace, "GetNamespace failed")

	addToFrameworkSchemeForSmokeTests(t)
}

func (suite *SimplestTestSuite) TearDownSuite() {
	ctx.Cleanup()
}

func (suite *SimplestTestSuite) SetupTest() {
	t = suite.T()
}

func TestSimplestSuite(t *testing.T) {
	suite.Run(t, new(SimplestTestSuite))
}

func (suite *SimplestTestSuite) simplest() {
	// create jaeger custom resource
	simplestJaeger := getSimplestJaeger()
	err := fw.Client.Create(goctx.TODO(), simplestJaeger, &framework.CleanupOptions{TestContext: ctx, Timeout: timeout, RetryInterval: retryInterval})
	require.NoError(t, err, "Failed to create simplest jaeger")

	err = e2eutil.WaitForDeployment(t, fw.KubeClient, namespace, "my-jaeger", 1, retryInterval, timeout)
	require.NoError(t, err, "Failed to deploye simplest jaeger")
}

func getSimplestJaeger() *v1.Jaeger {
	simplestJaeger := &v1.Jaeger{
		TypeMeta: metav1.TypeMeta{
			Kind:       "Jaeger",
			APIVersion: "jaegertracing.io/v1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      "my-jaeger",
			Namespace: namespace,
		},
		Spec: v1.JaegerSpec{},
	}
	return simplestJaeger
}
