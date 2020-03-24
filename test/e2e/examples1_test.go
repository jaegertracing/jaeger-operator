// +build examples1

package e2e

import (
	"context"
	"encoding/json"
	"io/ioutil"
	"net/http"
	"os/exec"
	"strconv"
	"strings"
	"testing"
	"time"

	framework "github.com/operator-framework/operator-sdk/pkg/test"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/intstr"
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
	handleSuiteTearDown()
}

func TestExamplesSuite(t *testing.T) {
	suite.Run(t, new(ExamplesTestSuite))
}

func (suite *ExamplesTestSuite) SetupTest() {
	t = suite.T()
}

func (suite *ExamplesTestSuite) AfterTest(suiteName, testName string) {
	handleTestFailure()
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

	err = WaitForDeployment(t, fw.KubeClient, namespace, "agent-as-daemonset", 1, retryInterval, timeout)
	require.NoError(t, err)

	AllInOneSmokeTest(name)
}

func (suite *ExamplesTestSuite) TestSimpleProdDeployEsExample() {
	if !isOpenShift(t) {
		t.Skip("Only applies to openshift")
	}
	yamlFileName := "../../deploy/examples/openshift/simple-prod-deploy-es.yaml"
	smokeTestProductionExample("simple-prod", yamlFileName)
}

func (suite *ExamplesTestSuite) TestWithCassandra() {
	if skipCassandraTests {
		t.Skip()
	}
	// make sure cassandra deployment has finished
	err := WaitForStatefulset(t, fw.KubeClient, storageNamespace, "cassandra", retryInterval, timeout)
	require.NoError(t, err, "Error waiting for cassandra")

	yamlFileName := "../../deploy/examples/with-cassandra.yaml"
	smokeTestAllInOneExampleWithTimeout("with-cassandra", yamlFileName, timeout+1*time.Minute)
}

func (suite *ExamplesTestSuite) TestBusinessApp() {
	if !isOpenShift(t) {
		t.Skip("Skipping until issue #974 is fixed")
	}
	// First deploy a Jaeger instance
	jaegerInstance := createJaegerInstanceFromFile("simplest", "../../deploy/examples/simplest.yaml")
	defer undeployJaegerInstance(jaegerInstance)
	err := WaitForDeployment(t, fw.KubeClient, namespace, "simplest", 1, retryInterval, timeout+(1*time.Minute))
	require.NoError(t, err)

	// Now deploy deploy/examples/business-application-injected-sidecar.yaml
	cmd := exec.Command("kubectl", "create", "--namespace", namespace, "--filename", "../../deploy/examples/business-application-injected-sidecar.yaml")
	output, err := cmd.CombinedOutput()
	if err != nil && !strings.Contains(string(output), "AlreadyExists") {
		require.NoError(t, err, "Failed creating Jaeger instance with: [%s]\n", string(output))
	}
	err = WaitForDeployment(t, fw.KubeClient, namespace, "myapp", 1, retryInterval, timeout)
	require.NoError(t, err, "Failed waiting for myapp deployment")

	// Add a liveliness probe to create some traces
	vertxDeployment := &appsv1.Deployment{}
	key := types.NamespacedName{Name: "myapp", Namespace: namespace}
	err = fw.Client.Get(context.Background(), key, vertxDeployment)
	require.NoError(t, err)

	vertxPort := intstr.IntOrString{IntVal: 8080}
	livelinessHandler := &corev1.HTTPGetAction{Path: "/", Port: vertxPort, Scheme: corev1.URISchemeHTTP}
	handler := &corev1.Handler{HTTPGet: livelinessHandler}
	livelinessProbe := &corev1.Probe{Handler: *handler, InitialDelaySeconds: 1, FailureThreshold: 3, PeriodSeconds: 10, SuccessThreshold: 1, TimeoutSeconds: 1}

	containers := vertxDeployment.Spec.Template.Spec.Containers
	for index, container := range containers {
		if container.Name == "myapp" {
			vertxDeployment.Spec.Template.Spec.Containers[index].LivenessProbe = livelinessProbe
			err = fw.Client.Update(context.Background(), vertxDeployment)
			require.NoError(t, err)
			break
		}
	}

	// Confirm that we've created some traces
	ports := []string{"0:16686"}
	portForward, closeChan := CreatePortForward(namespace, "simplest", "all-in-one", ports, fw.KubeConfig)
	defer portForward.Close()
	defer close(closeChan)
	forwardedPorts, err := portForward.GetPorts()
	require.NoError(t, err)
	queryPort := strconv.Itoa(int(forwardedPorts[0].Local))

	url := "http://localhost:" + queryPort + "/api/traces?service=order"
	err = WaitAndPollForHTTPResponse(url, func(response *http.Response) (bool, error) {
		body, err := ioutil.ReadAll(response.Body)
		if err != nil {
			return false, err
		}

		resp := &resp{}
		err = json.Unmarshal(body, &resp)
		if err != nil {
			return false, err
		}

		return len(resp.Data) > 0 && strings.Contains(string(body), "traceID"), nil
	})
	require.NoError(t, err, "SmokeTest failed")
}

func execOcCommand(args ...string) {
	cmd := exec.Command("oc", args...)
	output, err := cmd.CombinedOutput()
	if err != nil && !strings.Contains(string(output), "AlreadyExists") {
		require.NoErrorf(t, err, "Failed executing oc command with [%v]\n", err)
	}
}
