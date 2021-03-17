// +build smoke

package e2e

import (
	goctx "context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"strconv"
	"strings"
	"testing"
	"time"

	framework "github.com/operator-framework/operator-sdk/pkg/test"
	"github.com/operator-framework/operator-sdk/pkg/test/e2eutil"
	log "github.com/sirupsen/logrus"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
	"k8s.io/apimachinery/pkg/util/wait"

	v1 "github.com/jaegertracing/jaeger-operator/pkg/apis/jaegertracing/v1"
)

const TrackingID = "MyTrackingId"

type AllInOneTestSuite struct {
	suite.Suite
}

func (suite *AllInOneTestSuite) SetupSuite() {
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

func (suite *AllInOneTestSuite) TearDownSuite() {
	handleSuiteTearDown()
}

func TestAllInOneSuite(t *testing.T) {
	suite.Run(t, new(AllInOneTestSuite))
}

func (suite *AllInOneTestSuite) SetupTest() {
	t = suite.T()
}

func (suite *AllInOneTestSuite) AfterTest(suiteName, testName string) {
	handleTestFailure()
}

func (suite *AllInOneTestSuite) TestAllInOne() {
	// create jaeger custom resource
	instanceName := "my-jaeger"
	exampleJaeger := GetJaegerAllInOneCR(instanceName, namespace)

	log.Infof("passing %v", exampleJaeger)
	err := fw.Client.Create(goctx.TODO(), exampleJaeger, &framework.CleanupOptions{TestContext: ctx, Timeout: timeout, RetryInterval: retryInterval})
	require.NoError(t, err, "Error deploying example Jaeger")
	defer undeployJaegerInstance(exampleJaeger)

	err = e2eutil.WaitForDeployment(t, fw.KubeClient, namespace, instanceName, 1, retryInterval, timeout)
	require.NoError(t, err, "Error waiting for deployment")

	AllInOneSmokeTest(instanceName)
}

func (suite *AllInOneTestSuite) TestAllInOneWithIngress() {
	// create jaeger custom resource
	ingressEnabled := true
	name := "my-jaeger-with-ingress"
	exampleJaeger := GetJaegerAllInOneCR(name, namespace)
	exampleJaeger.Spec.Ingress = v1.JaegerIngressSpec{
		Enabled:  &ingressEnabled,
		Security: v1.IngressSecurityNoneExplicit,
	}

	log.Infof("passing %v", exampleJaeger)
	err := fw.Client.Create(goctx.TODO(), exampleJaeger, &framework.CleanupOptions{TestContext: ctx, Timeout: timeout, RetryInterval: retryInterval})
	require.NoError(t, err, "Failed to create Jaeger instance")
	defer undeployJaegerInstance(exampleJaeger)

	err = e2eutil.WaitForDeployment(t, fw.KubeClient, namespace, name, 1, retryInterval, 3*timeout)
	require.NoError(t, err, "Error waiting for Jaeger deployment")

	url, httpClient := getQueryURLAndHTTPClient(name, "%s/api/services", true)
	req, err := http.NewRequest(http.MethodGet, url, nil)
	require.NoError(t, err, "Failed to create httpRequest")
	// Hit this url once to make Jaeger itself create a trace, then it will show up in services
	httpClient.Do(req)

	err = wait.Poll(retryInterval, timeout, func() (done bool, err error) {
		res, err := httpClient.Do(req)
		require.NoError(t, err)

		body, err := ioutil.ReadAll(res.Body)
		require.NoError(t, err)

		resp := &services{}
		err = json.Unmarshal(body, &resp)
		if err != nil {
			return false, nil
		}

		for _, v := range resp.Data {
			if v == "jaeger-query" {
				return true, nil
			}
		}

		return false, nil
	})
	require.NoError(t, err, "Failed waiting for expected content")

	AllInOneSmokeTest(name)
}

func (suite *AllInOneTestSuite) TestAllInOneWithUIConfig() {
	cleanupOptions := &framework.CleanupOptions{TestContext: ctx, Timeout: timeout, RetryInterval: retryInterval}
	basePath := "/jaeger"

	instanceName := "all-in-one-with-ui-config"

	j := GetJaegerAllInOneWithUICR(instanceName, namespace, basePath, TrackingID)
	err := fw.Client.Create(goctx.TODO(), j, cleanupOptions)
	require.NoError(t, err, "Error creating jaeger instance")
	err = e2eutil.WaitForDeployment(t, fw.KubeClient, namespace, instanceName, 1, retryInterval, timeout)
	require.NoError(t, err, "Error waiting for jaeger deployment")
	defer undeployJaegerInstance(j)

	ports := []string{"0:16686"}
	portForward, closeChan := CreatePortForward(namespace, instanceName, "all-in-one", ports, fw.KubeConfig)
	defer portForward.Close()
	defer close(closeChan)
	forwardedPorts, err := portForward.GetPorts()
	require.NoError(t, err)
	queryPort := strconv.Itoa(int(forwardedPorts[0].Local))

	url := fmt.Sprintf("http://localhost:%s/%s/search", queryPort, basePath)
	c := http.Client{Timeout: 3 * time.Second}

	req, err := http.NewRequest(http.MethodGet, url, nil)
	require.NoError(t, err, "Failed to create httpRequest")

	err = wait.Poll(retryInterval, timeout, func() (done bool, err error) {
		res, err := c.Do(req)
		if err != nil && strings.Contains(err.Error(), "Timeout exceeded") {
			log.Infof("Retrying request after error %v", err)
			return false, nil
		}
		require.NoError(t, err)

		if res.StatusCode != 200 {
			return false, fmt.Errorf("unexpected status code %d", res.StatusCode)
		}

		body, err := ioutil.ReadAll(res.Body)
		require.NoError(t, err)

		if len(body) == 0 {
			return false, fmt.Errorf("empty body")
		}

		if !strings.Contains(string(body), TrackingID) {
			return false, fmt.Errorf("body does not include tracking id: %s", TrackingID)
		}

		return true, nil
	})
	require.NoError(t, err, "Failed waiting for expected content")

	AllInOneSmokeTestWithQueryBasePath(instanceName, basePath)
}
