// +build examples_openshift

package e2e

import (
	"testing"

	framework "github.com/operator-framework/operator-sdk/pkg/test"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
)

type ExamplesTestSuiteOCP struct {
	suite.Suite
}

func (suite *ExamplesTestSuiteOCP) SetupSuite() {
	t = suite.T()
	if !isOpenShift(t) {
		t.Skip("This suite should only be run on OpenShift")
	}
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

func (suite *ExamplesTestSuiteOCP) TearDownSuite() {
	handleSuiteTearDown()
}

func TestOpenShiftExamplesSuite(t *testing.T) {
	suite.Run(t, new(ExamplesTestSuiteOCP))
}

func (suite *ExamplesTestSuiteOCP) SetupTest() {
	t = suite.T()
}

func (suite *ExamplesTestSuiteOCP) AfterTest(suiteName, testName string) {
	handleTestFailure()
}

func (suite *ExamplesTestSuiteOCP) TestSimpleProdDeployEsExample() {
	yamlFileName := "../../deploy/examples/openshift/simple-prod-deploy-es.yaml"
	smokeTestProductionExample("simple-prod", yamlFileName)
}
