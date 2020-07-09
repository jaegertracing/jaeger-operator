// +build examples1

package e2e

import (
	"context"
	"encoding/json"
	"errors"
	"io/ioutil"
	"net/http"
	"os/exec"
	"strconv"
	"strings"
	"testing"
	"time"

	framework "github.com/operator-framework/operator-sdk/pkg/test"
	log "github.com/sirupsen/logrus"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
	"k8s.io/apimachinery/pkg/util/wait"
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
	namespace = ctx.GetID()
	require.NotNil(t, namespace, "GetID failed")

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
	// First deploy a Jaeger instance
	jaegerInstanceName := "simplest"
	jaegerInstance := createJaegerInstanceFromFile(jaegerInstanceName, "../../deploy/examples/simplest.yaml")
	defer undeployJaegerInstance(jaegerInstance)
	err := WaitForDeployment(t, fw.KubeClient, namespace, jaegerInstanceName, 1, retryInterval, timeout+(1*time.Minute))
	require.NoError(t, err)

	// Now deploy deploy/examples/business-application-injected-sidecar.yaml
	cmd := exec.Command("kubectl", "create", "--namespace", namespace, "--filename", "../../deploy/examples/business-application-injected-sidecar.yaml")
	output, err := cmd.CombinedOutput()
	if err != nil && !strings.Contains(string(output), "AlreadyExists") {
		require.NoError(t, err, "Failed creating Jaeger instance with: [%s]\n", string(output))
	}
	const vertxDeploymentName = "myapp"
	err = WaitForDeployment(t, fw.KubeClient, namespace, vertxDeploymentName, 1, retryInterval, timeout)
	require.NoError(t, err, "Failed waiting for myapp deployment")

	// Add a liveliness probe to create some traces
	vertxPort := intstr.IntOrString{IntVal: 8080}
	livelinessHandler := &corev1.HTTPGetAction{Path: "/", Port: vertxPort, Scheme: corev1.URISchemeHTTP}
	handler := &corev1.Handler{HTTPGet: livelinessHandler}
	livelinessProbe := &corev1.Probe{Handler: *handler, InitialDelaySeconds: 1, FailureThreshold: 3, PeriodSeconds: 10, SuccessThreshold: 1, TimeoutSeconds: 1}

	err = wait.Poll(retryInterval, timeout, func() (done bool, err error) {
		vertxDeployment, err := fw.KubeClient.AppsV1().Deployments(namespace).Get(context.Background(), vertxDeploymentName, metav1.GetOptions{})
		require.NoError(t, err)
		containers := vertxDeployment.Spec.Template.Spec.Containers
		for index, container := range containers {
			if container.Name == vertxDeploymentName {
				vertxDeployment.Spec.Template.Spec.Containers[index].LivenessProbe = livelinessProbe
				updatedVertxDeployment, err := fw.KubeClient.AppsV1().Deployments(namespace).Update(context.Background(), vertxDeployment, metav1.UpdateOptions{})
				if err != nil {
					log.Warnf("Error %v updating vertx app, retrying", err)
					return false, nil
				}
				log.Infof("Updated deployment %v", updatedVertxDeployment.Name)
				return true, nil
			}
		}
		return false, errors.New("Vertx deployment not found")
	})
	require.NoError(t, err)

	// Confirm that we've created some traces
	ports := []string{"0:16686"}
	portForward, closeChan := CreatePortForward(namespace, jaegerInstanceName, "all-in-one", ports, fw.KubeConfig)
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
