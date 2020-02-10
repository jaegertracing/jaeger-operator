// +build smokeA

package e2e

import (
	goctx "context"
	"encoding/json"
	"io/ioutil"
	"net/http"
	"testing"

	"github.com/jaegertracing/jaeger-operator/pkg/inject"

	framework "github.com/operator-framework/operator-sdk/pkg/test"
	"github.com/operator-framework/operator-sdk/pkg/test/e2eutil"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
	"k8s.io/apimachinery/pkg/util/wait"
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
	namespace, _ = ctx.GetNamespace()
	require.NotNil(t, namespace, "GetNamespace failed")

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

	jaegerInstanceName := "agent-as-sidecar"
	j := getJaegerAgentAsSidecarDefinition(jaegerInstanceName, namespace)
	err := fw.Client.Create(goctx.TODO(), j, cleanupOptions)
	require.NoError(t, err, "Failed to create jaeger instance")
	defer undeployJaegerInstance(j)

	err = e2eutil.WaitForDeployment(t, fw.KubeClient, namespace, jaegerInstanceName, 1, retryInterval, timeout)
	require.NoError(t, err, "Error waiting for Jaeger instance deployment")

	dep := getVertxDefinition2(namespace)
	err = fw.Client.Create(goctx.TODO(), dep, cleanupOptions)
	require.NoError(t, err, "Failed to create vertx instance")

	err = e2eutil.WaitForDeployment(t, fw.KubeClient, namespace, "vertx-create-span-sidecar", 1, retryInterval, timeout)
	require.NoError(t, err, "Failed waiting for vertx-create-span-sidecar deployment")

	nss, err := fw.KubeClient.CoreV1().Namespaces().Get(namespace, metav1.GetOptions{})
	require.NoError(t, err)
	nss.Annotations[inject.Annotation] = "true"
	_, err = fw.KubeClient.CoreV1().Namespaces().Update(nss)
	require.NoError(t, err)

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

func getVertxDefinition2(s string) *appsv1.Deployment {
	selector := map[string]string{"app": "vertx-create-span-sidecar"}
	dep := &appsv1.Deployment{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "apps/v1",
			Kind:       "Deployment",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      "vertx-create-span-sidecar",
			Namespace: namespace,
		},
		Spec: appsv1.DeploymentSpec{
			Selector: &metav1.LabelSelector{
				MatchLabels: selector,
			},
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: selector,
				},
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{{
						Image: "jaegertracing/vertx-create-span:operator-e2e-tests",
						Name:  "vertx-create-span-sidecar",
						Ports: []corev1.ContainerPort{
							{
								ContainerPort: 8080,
							},
						},
						ReadinessProbe: &corev1.Probe{
							Handler: corev1.Handler{
								HTTPGet: &corev1.HTTPGetAction{
									Path: "/",
									Port: intstr.FromInt(8080),
								},
							},
							InitialDelaySeconds: 1,
						},
						LivenessProbe: &corev1.Probe{
							Handler: corev1.Handler{
								HTTPGet: &corev1.HTTPGetAction{
									Path: "/",
									Port: intstr.FromInt(8080),
								},
							},
							InitialDelaySeconds: 1,
						},
					}},
				},
			},
		},
	}
	return dep
}
