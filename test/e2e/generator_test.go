// +build generate

package e2e

import (
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"os/exec"
	"strings"
	"testing"
	"time"

	framework "github.com/operator-framework/operator-sdk/pkg/test"
	"github.com/operator-framework/operator-sdk/pkg/test/e2eutil"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
	"k8s.io/apimachinery/pkg/util/wait"
)

type GeneratorAllInOneTestSuite struct {
	suite.Suite
}

func (suite *GeneratorAllInOneTestSuite) SetupSuite() {
	t = suite.T()
	ctx = framework.NewTestCtx(t)
	fw = framework.Global
	namespace, _ = ctx.GetNamespace()
	require.NotNil(t, namespace, "GetNamespace failed")
}

func (suite *GeneratorAllInOneTestSuite) TearDownSuite() {
	handleSuiteTearDown()
}

func TestGeneratorAllInOneSuite(t *testing.T) {
	suite.Run(t, new(GeneratorAllInOneTestSuite))
}

func (suite *GeneratorAllInOneTestSuite) SetupTest() {
	t = suite.T()
}

func (suite *GeneratorAllInOneTestSuite) AfterTest(suiteName, testName string) {
	handleTestFailure()
}

// Returns deployment name and allInOne as a temporary file. Caller must delete
func getAllInOneTempFile() (string, *os.File) {
	cr := `apiVersion: jaegertracing.io/v1
kind: "Jaeger"
metadata:
  name: "my-jaeger"
spec:
  strategy: allInOne
  allInOne:
    image: jaegertracing/all-in-one:1.13
`
	name := "my-jaeger"

	f, err := ioutil.TempFile("", "crd*")
	require.NoError(t, err, "temp file")

	f.Write([]byte(cr))

	return name, f
}

func (suite *GeneratorAllInOneTestSuite) TestAllInOne() {
	// Get a *os.File for Jaeger CR with all in one, and the name of the deployment
	name, cr := getAllInOneTempFile()
	defer func() {
		cr.Close()
		os.Remove(cr.Name())
	}()

	// Create a temporary file for the output
	output, err := ioutil.TempFile("", "output*")
	require.NoError(t, err, "temp file")

	defer func() {
		output.Close()
		os.Remove(output.Name())
	}()

	// Execute the generate command
	generateOutput, err := exec.Command("../../build/_output/bin/jaeger-operator", "generate", "--cr", cr.Name(), "--output", output.Name()).CombinedOutput()
	require.NoError(t, err, "generate failed: %s", generateOutput)

	kubectlOutput, err := exec.Command("kubectl", "create", "-n", namespace, "-f", output.Name()).CombinedOutput()
	require.NoError(t, err, "could not create objects from yaml: %s", kubectlOutput)

	err = e2eutil.WaitForDeployment(t, fw.KubeClient, namespace, name, 1, retryInterval, 3*timeout)
	require.NoError(t, err, "Error waiting for Jaeger deployment")

	// Check that deployment seems OK
	ports := []string{"0:16686", "0:14268"}
	portForward, closeChan := CreatePortForward(namespace, name, "all-in-one", ports, fw.KubeConfig)
	defer portForward.Close()
	defer close(closeChan)
	forwardedPorts, err := portForward.GetPorts()
	require.NoError(t, err)

	url := fmt.Sprintf("http://localhost:%d/search", forwardedPorts[0].Local)
	c := http.Client{Timeout: 3 * time.Second}

	req, err := http.NewRequest(http.MethodGet, url, nil)
	require.NoError(t, err, "Failed to create httpRequest")

	err = wait.Poll(retryInterval, timeout, func() (done bool, err error) {
		res, err := c.Do(req)
		if err != nil && strings.Contains(err.Error(), "Timeout exceeded") {
			t.Logf("Retrying request after error %v", err)
			return false, nil
		}
		require.NoError(t, err)

		require.Equal(t, 200, res.StatusCode)

		body, err := ioutil.ReadAll(res.Body)
		require.NoError(t, err)

		require.NotEqual(t, 0, len(body), "Empty body")

		return true, nil
	})
	require.NoError(t, err, "Failed waiting for expected content")

	queryPort := forwardedPorts[0].Local
	collectorPort := forwardedPorts[1].Local

	apiTracesEndpoint := fmt.Sprintf("http://localhost:%d/api/traces", queryPort)
	collectorEndpoint := fmt.Sprintf("http://localhost:%d/api/traces", collectorPort)
	executeSmokeTest(apiTracesEndpoint, collectorEndpoint, false)
}
