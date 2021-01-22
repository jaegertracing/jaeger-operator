// +build istio

package e2e

import (
	"context"
	"encoding/json"
	"io/ioutil"
	"net/http"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"testing"
	"time"

	framework "github.com/operator-framework/operator-sdk/pkg/test"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
)

type IstioTestSuite struct {
	suite.Suite
}

// LIFECYCLE - Suite
func (suite *IstioTestSuite) SetupSuite() {
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

	// label namespace
	ns, err := framework.Global.KubeClient.CoreV1().Namespaces().Get(context.Background(), namespace, metav1.GetOptions{})
	require.NoError(t, err, "failed to get the namespaces details: %v", err)

	nsLabels := ns.GetLabels()
	if nsLabels == nil {
		nsLabels = make(map[string]string)
	}
	nsLabels["istio-injection"] = "enabled"
	ns.SetLabels(nsLabels)

	ns, err = framework.Global.KubeClient.CoreV1().Namespaces().Update(context.Background(), ns, metav1.UpdateOptions{})
	require.NoError(t, err, "failed to update labels of the namespace %s", namespace)

	addToFrameworkSchemeForSmokeTests(t)
}

func (suite *IstioTestSuite) TearDownSuite() {
	handleSuiteTearDown()
}

// LIFECYCLE - Test

func (suite *IstioTestSuite) SetupTest() {
	t = suite.T()
}

func (suite *IstioTestSuite) AfterTest(suiteName, testName string) {
	handleTestFailure()
}

func TestIstioSuite(t *testing.T) {
	suite.Run(t, new(IstioTestSuite))
}

func (suite *IstioTestSuite) TestEnvoySidecar() {
	// First deploy a Jaeger instance
	jaegerInstanceName := "simplest"
	jaegerInstance := createJaegerInstanceFromFile(jaegerInstanceName, "../../examples/simplest.yaml")
	defer undeployJaegerInstance(jaegerInstance)
	err := WaitForDeployment(t, fw.KubeClient, namespace, jaegerInstanceName, 1, retryInterval, timeout+(1*time.Minute))
	require.NoError(t, err)

	// Now deploy examples/business-application-injected-sidecar.yaml
	businessAppCR := getBusinessAppCR()
	defer os.Remove(businessAppCR.Name())
	cmd := exec.Command("kubectl", "create", "--namespace", namespace, "--filename", businessAppCR.Name())
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

	err = waitForDeploymentAndUpdate(vertxDeploymentName, vertxDeploymentName, func(container *corev1.Container) {
		container.LivenessProbe = livelinessProbe
	})
	require.NoError(t, err)

	exists := testContainerInPod(namespace, vertxDeploymentName, "istio-proxy", nil)
	require.True(t, exists)

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
		require.NoError(t, err)

		resp := &resp{}
		err = json.Unmarshal(body, &resp)
		require.NoError(t, err)

		return len(resp.Data) > 0 && strings.Contains(string(body), "traceID"), nil
	})
	require.NoError(t, err, "SmokeTest failed")
}
