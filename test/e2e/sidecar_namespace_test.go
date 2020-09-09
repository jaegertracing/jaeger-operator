// +build smoke

package e2e

import (
	"context"
	goctx "context"
	"encoding/json"
	"io/ioutil"
	"net/http"
	"testing"

	framework "github.com/operator-framework/operator-sdk/pkg/test"
	"github.com/operator-framework/operator-sdk/pkg/test/e2eutil"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/wait"

	"github.com/jaegertracing/jaeger-operator/pkg/inject"
)

type SidecarNamespaceTestSuite struct {
	suite.Suite
}

func (suite *SidecarNamespaceTestSuite) SetupSuite() {
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

func (suite *SidecarNamespaceTestSuite) TearDownSuite() {
	handleSuiteTearDown()
}

func TestSidecarNamespaceSuite(t *testing.T) {
	suite.Run(t, new(SidecarNamespaceTestSuite))
}

func (suite *SidecarNamespaceTestSuite) SetupTest() {
	t = suite.T()
}

func (suite *SidecarNamespaceTestSuite) AfterTest(suiteName, testName string) {
	handleTestFailure()
}

// Sidecar runs a test with the agent as sidecar
func (suite *SidecarNamespaceTestSuite) TestSidecarNamespace() {
	cleanupOptions := &framework.CleanupOptions{TestContext: ctx, Timeout: timeout, RetryInterval: retryInterval}

	jaegerInstanceName := "agent-as-sidecar-namespace"
	j := createJaegerAgentAsSidecarInstance(jaegerInstanceName, namespace, testOtelAgent, testOtelAllInOne)
	defer undeployJaegerInstance(j)

	dep := getVertxDefinition(namespace, map[string]string{})
	err := fw.Client.Create(goctx.TODO(), dep, cleanupOptions)
	require.NoError(t, err, "Failed to create vertx instance")

	err = e2eutil.WaitForDeployment(t, fw.KubeClient, namespace, dep.Name, 1, retryInterval, timeout)
	require.NoError(t, err, "Failed waiting for vertx-create-span-sidecar deployment")

	dep, err = fw.KubeClient.AppsV1().Deployments(namespace).Get(context.Background(), dep.Name, metav1.GetOptions{})
	require.NoError(t, err)
	hasAgent, _ := inject.HasJaegerAgent(dep)
	require.False(t, hasAgent)

	nss, err := fw.KubeClient.CoreV1().Namespaces().Get(context.Background(), namespace, metav1.GetOptions{})
	require.NoError(t, err)
	if nss.Annotations == nil {
		nss.Annotations = map[string]string{}
	}
	nss.Annotations[inject.Annotation] = "true"
	_, err = fw.KubeClient.CoreV1().Namespaces().Update(context.Background(), nss, metav1.UpdateOptions{})
	require.NoError(t, err)

	err = e2eutil.WaitForDeployment(t, fw.KubeClient, namespace, dep.Name, 1, retryInterval, timeout)
	require.NoError(t, err, "Failed waiting for %s deployment", dep.Name)

	url, httpClient := getQueryURLAndHTTPClient(jaegerInstanceName, "%s/api/traces?service=order", true)
	req, err := http.NewRequest(http.MethodGet, url, nil)
	require.NoError(t, err, "Failed to create httpRequest")
	err = wait.Poll(retryInterval, timeout, func() (done bool, err error) {
		res, err := httpClient.Do(req)
		require.NoError(t, err)

		body, err := ioutil.ReadAll(res.Body)
		require.NoError(t, err)

		resp := &resp{}
		err = json.Unmarshal(body, &resp)
		require.NoError(t, err)

		return len(resp.Data) > 0, nil
	})
	require.NoError(t, err, "Failed waiting for expected content")
}
