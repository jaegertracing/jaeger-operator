// +build examples

package e2e

import (
	"github.com/jaegertracing/jaeger-operator/pkg/apis/jaegertracing/v1"
	framework "github.com/operator-framework/operator-sdk/pkg/test"
	"github.com/operator-framework/operator-sdk/pkg/test/e2eutil"
	log "github.com/sirupsen/logrus"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"

	"os/exec"
	"strings"
	"testing"
)

type ExamplesTestSuite struct {
	suite.Suite
}

func (suite *ExamplesTestSuite) SetupSuite() {
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

func (suite *ExamplesTestSuite) TearDownSuite() {
	log.Info("Entering TearDownSuite()")
	ctx.Cleanup()
}

func TestExamplesSuite(t *testing.T) {
	suite.Run(t, new(ExamplesTestSuite))
}

func (suite *ExamplesTestSuite) SetupTest() {
	t = suite.T()
}

func (suite *ExamplesTestSuite) TestAgentAsDaemonSet() {
	var yamlFileName string
	name := "agent-as-daemonset"

	if isOpenShift(t) {
		yamlFileName = "../../deploy/examples/openshift/agent-as-daemonset.yaml"

		execOcCommand("create", "--namespace", namespace, "-f", "../../deploy/examples/openshift/hostport-scc-daemonset.yaml")
		execOcCommand("create", "--namespace", namespace, "-f", "../../deploy/examples/openshift/service_account_jaeger-agent-daemonset.yaml")
		execOcCommand("adm", "policy", "--namespace", namespace, "add-scc-to-user", "daemonset-with-hostport", "-z", "jaeger-agent-daemonset")
	} else {
		yamlFileName = "../../deploy/examples/agent-as-daemonset.yaml"
	}

	jaegerInstance := createJaegerInstanceFromFile(name, yamlFileName)
	defer undeployJaegerInstance(jaegerInstance)

	err := WaitForDaemonSet(t, fw.KubeClient, namespace, name+"-agent-daemonset", retryInterval, timeout)
	require.NoError(t, err)

	AllInOneSmokeTest(name)
}

func (suite *ExamplesTestSuite) TestSimplestExample() {
	smokeTestAllInOneExample("simplest", "../../deploy/examples/simplest.yaml")
}

func (suite *ExamplesTestSuite) TestSimpleProdDeployEsExample() {
	if !isOpenShift(t) {
		t.Skip("Only applies to openshift")
	}
	yamlFileName := "../../deploy/examples/simple-prod-deploy-es.yaml"
	smokeTestProductionExample("simple-prod", yamlFileName)
}

func (suite *ExamplesTestSuite) TestSimpleProdWithVolumes() {
	yamlFileName := "../../deploy/examples/simple-prod-with-volumes.yaml"
	smokeTestProductionExample("simple-prod", yamlFileName)
}

func (suite *ExamplesTestSuite) TestSimpleProdExample() {
	yamlFileName := "../../deploy/examples/simple-prod.yaml"
	smokeTestProductionExample("simple-prod", yamlFileName)
}

func (suite *ExamplesTestSuite) TestSimpleStreamingExample() {
	yamlFileName := "../../deploy/examples/simple-streaming.yaml"
	smokeTestProductionExample("simple-streaming", yamlFileName)
}

func (suite *ExamplesTestSuite) TestWithCassandra() {
	yamlFileName := "../../deploy/examples/with-cassandra.yaml"
	smokeTestAllInOneExample("with-cassandra", yamlFileName)
}

func (suite *ExamplesTestSuite) TestWithSampling() {
	name := "with-sampling"
	yamlFileName := "../../deploy/examples/with-sampling.yaml"
	// This is the same as smokeTestAllInOneExample, but we need to check the jaegerInstance after it finishes
	jaegerInstance := createJaegerInstanceFromFile(name, yamlFileName)
	defer undeployJaegerInstance(jaegerInstance)

	err := e2eutil.WaitForDeployment(t, fw.KubeClient, namespace, name, 1, retryInterval, timeout)
	require.NoErrorf(t, err, "Error waiting for %s to deploy", name)

	// Check sampling options.  t would be nice to create some spans and check that they are being sampled at the correct rate
	samplingOptions, err := jaegerInstance.Spec.Sampling.Options.MarshalJSON()
	require.NoError(t, err)
	require.Equal(t, "{\"default_strategy\":{\"param\":50,\"type\":\"probabilistic\"}}", string(samplingOptions))
}

func smokeTestAllInOneExample(name, yamlFileName string) {
	jaegerInstance := createJaegerInstanceFromFile(name, yamlFileName)
	defer undeployJaegerInstance(jaegerInstance)

	err := e2eutil.WaitForDeployment(t, fw.KubeClient, namespace, name, 1, retryInterval, timeout)
	require.NoErrorf(t, err, "Error waiting for %s to deploy", name)

	AllInOneSmokeTest(name)
}

func smokeTestProductionExample(name, yamlFileName string) {
	jaegerInstance := createJaegerInstanceFromFile(name, yamlFileName)
	defer undeployJaegerInstance(jaegerInstance)

	queryDeploymentName := name + "-query"
	collectorDeploymentName := name + "-collector"

	err := e2eutil.WaitForDeployment(t, fw.KubeClient, namespace, queryDeploymentName, 1, retryInterval, timeout)
	require.NoErrorf(t, err, "Error waiting for %s to deploy", queryDeploymentName)
	err = e2eutil.WaitForDeployment(t, fw.KubeClient, namespace, collectorDeploymentName, 1, retryInterval, timeout)
	require.NoErrorf(t, err, "Error waiting for %s to deploy", collectorDeploymentName)

	ProductionSmokeTest(name)
}

func createJaegerInstanceFromFile(name, filename string) *v1.Jaeger {
	cmd := exec.Command("kubectl", "create", "--namespace", namespace, "--filename", filename)
	output, err := cmd.CombinedOutput()
	if err != nil && !strings.Contains(string(output), "AlreadyExists") {
		require.NoError(t, err, "Failed creating Jaeger instance with: [%s]\n", string(output))
	}

	return getJaegerInstance(name, namespace)
}

func execOcCommand(args ...string) {
	cmd := exec.Command("oc", args...)
	output, err := cmd.CombinedOutput()
	if err != nil && !strings.Contains(string(output), "AlreadyExists") {
		require.NoErrorf(t, err, "Failed executing oc command with [%v]\n", err)
	}
}
