// +build examples2

package e2e

import (
	"context"
	"fmt"
	"testing"

	"github.com/sirupsen/logrus"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	framework "github.com/operator-framework/operator-sdk/pkg/test"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
)

type ExamplesTestSuite2 struct {
	suite.Suite
}

func (suite *ExamplesTestSuite2) SetupSuite() {
	t = suite.T()
	var err error
	ctx, err = prepare(t)
	if err != nil {
		if ctx != nil {
			ctx.Cleanup()
		}
		logrus.Errorf("Prepare returned error: %v", err)
		require.FailNow(t, "Failed in prepare")
	}
	fw = framework.Global
	namespace = ctx.GetID()
	require.NotNil(t, namespace, "GetID failed")

	addToFrameworkSchemeForSmokeTests(t)
}

func (suite *ExamplesTestSuite2) TearDownSuite() {
	handleSuiteTearDown()
}

func TestExamplesSuite2(t *testing.T) {
	suite.Run(t, new(ExamplesTestSuite2))
}

func (suite *ExamplesTestSuite2) SetupTest() {
	t = suite.T()
}

func (suite *ExamplesTestSuite2) AfterTest(suiteName, testName string) {
	handleTestFailure()
}

func (suite *ExamplesTestSuite2) TestSimplestExample() {
	yamlFileName := "../../examples/simplest.yaml"
	smokeTestAllInOneExample("simplest", yamlFileName)
}

func (suite *ExamplesTestSuite2) TestWithBadgerExample() {
	smokeTestAllInOneExample("with-badger", "../../examples/with-badger.yaml")
}

func (suite *ExamplesTestSuite2) TestWithBadgerAndVolumeExample() {
	smokeTestAllInOneExample("with-badger-and-volume", "../../examples/with-badger-and-volume.yaml")
}

func (suite *ExamplesTestSuite2) TestServiceTypesExample() {
	yamlFileName := "../../examples/service-types.yaml"
	name := "service-types"
	jaegerInstance := createJaegerInstanceFromFile(name, yamlFileName)
	defer undeployJaegerInstance(jaegerInstance)

	err := WaitForDeployment(t, fw.KubeClient, namespace, name, 1, retryInterval, timeout)
	require.NoErrorf(t, err, "Error waiting for %s to deploy", name)

	AllInOneSmokeTest(name)

	collectorService, err := fw.KubeClient.CoreV1().Services(namespace).Get(context.Background(), fmt.Sprintf("%s-collector", name), v1.GetOptions{})
	require.NoError(t, err)
	require.Equal(t, "LoadBalancer", string(collectorService.Spec.Type))
	queryService, err := fw.KubeClient.CoreV1().Services(namespace).Get(context.Background(), fmt.Sprintf("%s-query", name), v1.GetOptions{})
	require.NoError(t, err)
	require.Equal(t, "LoadBalancer", string(queryService.Spec.Type))
}

func (suite *ExamplesTestSuite2) TestSimpleProdWithVolumes() {
	if skipESExternal {
		t.Skip("This example requires an external ElasticSearch instance")
	}
	yamlFileName := "../../examples/simple-prod-with-volumes.yaml"
	smokeTestProductionExample("simple-prod", yamlFileName)
}

func (suite *ExamplesTestSuite2) TestSimpleProdExample() {
	if skipESExternal {
		t.Skip("This example requires an external ElasticSearch instance")
	}
	yamlFileName := "../../examples/simple-prod.yaml"
	smokeTestProductionExample("simple-prod", yamlFileName)
}

func (suite *ExamplesTestSuite2) TestSimpleStreamingExample() {
	if skipESExternal {
		t.Skip("This example requires an external ElasticSearch instance")
	}
	yamlFileName := "../../examples/simple-streaming.yaml"
	smokeTestProductionExample("simple-streaming", yamlFileName)
}

func (suite *ExamplesTestSuite2) TestWithSampling() {
	name := "with-sampling"
	yamlFileName := "../../examples/with-sampling.yaml"
	// This is the same as smokeTestAllInOneExample, but we need to check the jaegerInstance after it finishes
	jaegerInstance := createJaegerInstanceFromFile(name, yamlFileName)
	defer undeployJaegerInstance(jaegerInstance)

	err := WaitForDeployment(t, fw.KubeClient, namespace, name, 1, retryInterval, timeout)
	require.NoErrorf(t, err, "Error waiting for %s to deploy", name)

	// Check sampling options.  t would be nice to create some spans and check that they are being sampled at the correct rate
	samplingOptions, err := jaegerInstance.Spec.Sampling.Options.MarshalJSON()
	require.NoError(t, err)
	require.Equal(t, "{\"default_strategy\":{\"param\":50,\"type\":\"probabilistic\"}}", string(samplingOptions))
}
