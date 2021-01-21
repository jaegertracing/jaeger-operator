// +build istio

package e2e

import (
	"context"
	"encoding/json"
	"errors"
	"io/ioutil"
	"net/http"
	"os"
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
	businessAppCR := getBusinessAppCR(err)
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

	exists := testContainerInPod(vertxDeploymentName, "istio-proxy", nil)
	require.True(suite.T(), exists)

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

func getBusinessAppCR(err error) *os.File {
	content, err := ioutil.ReadFile("../../examples/business-application-injected-sidecar.yaml")
	require.NoError(t, err)
	newContent := strings.Replace(string(content), "image: jaegertracing/vertx-create-span:operator-e2e-tests", "image: "+vertxExampleImage, 1)
	file, err := ioutil.TempFile("", "vertx-example")
	require.NoError(t, err)
	err = ioutil.WriteFile(file.Name(), []byte(newContent), 0666)
	require.NoError(t, err)
	return file
}
