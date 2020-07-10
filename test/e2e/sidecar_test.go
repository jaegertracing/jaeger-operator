// +build smoke

package e2e

import (
	goctx "context"
	"encoding/json"
	"io/ioutil"
	"net/http"
	"testing"

	framework "github.com/operator-framework/operator-sdk/pkg/test"
	"github.com/operator-framework/operator-sdk/pkg/test/e2eutil"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/intstr"
	"k8s.io/apimachinery/pkg/util/wait"

	v1 "github.com/jaegertracing/jaeger-operator/pkg/apis/jaegertracing/v1"
	"github.com/jaegertracing/jaeger-operator/pkg/inject"
)

var ingressEnabled = true

type SidecarTestSuite struct {
	suite.Suite
}

func (suite *SidecarTestSuite) SetupSuite() {
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

func (suite *SidecarTestSuite) TearDownSuite() {
	handleSuiteTearDown()
}

func TestSidecarSuite(t *testing.T) {
	suite.Run(t, new(SidecarTestSuite))
}

func (suite *SidecarTestSuite) SetupTest() {
	t = suite.T()
}

func (suite *SidecarTestSuite) AfterTest(suiteName, testName string) {
	handleTestFailure()
}

// Sidecar runs a test with the agent as sidecar
func (suite *SidecarTestSuite) TestSidecar() {

	cleanupOptions := &framework.CleanupOptions{TestContext: ctx, Timeout: timeout, RetryInterval: retryInterval}

	jaegerInstanceName := "agent-as-sidecar"
	j := getJaegerAgentAsSidecarDefinition(jaegerInstanceName, namespace)
	undeployJaegerInstance(j)
	err := fw.Client.Create(goctx.TODO(), j, cleanupOptions)
	require.NoError(t, err, "Failed to create jaeger instance")

	err = e2eutil.WaitForDeployment(t, fw.KubeClient, namespace, jaegerInstanceName, 1, retryInterval, timeout)
	require.NoError(t, err, "Error waiting for Jaeger instance deployment")

	dep := getVertxDefinition(map[string]string{inject.Annotation: "true"})
	err = fw.Client.Create(goctx.TODO(), dep, cleanupOptions)
	require.NoError(t, err, "Failed to create vertx instance")

	err = e2eutil.WaitForDeployment(t, fw.KubeClient, namespace, "vertx-create-span-sidecar", 1, retryInterval, timeout)
	require.NoError(t, err, "Failed waiting for vertx-create-span-sidecar deployment")

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

	/* Testing other instance */

	otherJaegerInstanceName := "agent-as-sidecar2"
	j2 := getJaegerAgentAsSidecarDefinition(otherJaegerInstanceName, namespace)
	defer undeployJaegerInstance(j2)

	err = fw.Client.Create(goctx.TODO(), j2, cleanupOptions)
	err = e2eutil.WaitForDeployment(t, fw.KubeClient, namespace, otherJaegerInstanceName, 1, retryInterval, timeout)
	require.NoError(t, err, "Error waiting for Jaeger instance deployment")

	persisted := &appsv1.Deployment{}
	err = fw.Client.Get(goctx.TODO(), types.NamespacedName{
		Name:      "vertx-create-span-sidecar",
		Namespace: namespace,
	}, persisted)
	require.NoError(t, err, "Error getting jaeger instance")
	require.Equal(t, "agent-as-sidecar", persisted.Labels[inject.Label])

	err = fw.Client.Delete(goctx.TODO(), j)
	require.NoError(t, err, "Error deleting instance")

	url, httpClient = getQueryURLAndHTTPClient(otherJaegerInstanceName, "%s/api/traces?service=order", true)
	req, err = http.NewRequest(http.MethodGet, url, nil)

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

func getVertxDefinition(annotations map[string]string) *appsv1.Deployment {
	selector := map[string]string{"app": "vertx-create-span-sidecar"}
	dep := &appsv1.Deployment{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "apps/v1",
			Kind:       "Deployment",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:        "vertx-create-span-sidecar",
			Namespace:   namespace,
			Annotations: annotations,
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

func getJaegerAgentAsSidecarDefinition(name, namespace string) *v1.Jaeger {
	j := &v1.Jaeger{
		TypeMeta: metav1.TypeMeta{
			Kind:       "Jaeger",
			APIVersion: "jaegertracing.io/v1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
		},
		Spec: v1.JaegerSpec{
			Strategy: v1.DeploymentStrategyAllInOne,
			JaegerCommonSpec: v1.JaegerCommonSpec{
				// do not inject jaeger-agent into Jaeger deployment - it will result in port collision
				Annotations: map[string]string{inject.Annotation: "doesNotExists"},
			},
			AllInOne: v1.JaegerAllInOneSpec{},
			Agent: v1.JaegerAgentSpec{
				Options: v1.NewOptions(map[string]interface{}{
					"log-level": "debug",
				}),
			},
			Ingress: v1.JaegerIngressSpec{
				Enabled:  &ingressEnabled,
				Security: v1.IngressSecurityNoneExplicit,
			},
		},
	}
	return j
}
