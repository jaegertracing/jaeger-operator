// +build smoke

package e2e

import (
	goctx "context"
	"encoding/json"
	"io/ioutil"
	"net/http"
	"testing"

	"github.com/sirupsen/logrus"

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

	firstJaegerInstanceName := "agent-as-sidecar"
	firstJaegerInstance := createJaegerAgentAsSidecarInstance(firstJaegerInstanceName, namespace, testOtelAgent, testOtelAllInOne)
	defer undeployJaegerInstance(firstJaegerInstance)

	verifyAllInOneImage(firstJaegerInstanceName, namespace, testOtelAllInOne)

	vertxDeploymentName := "vertx-create-span-sidecar"
	dep := getVertxDefinition(vertxDeploymentName, map[string]string{inject.Annotation: "true"})
	err := fw.Client.Create(goctx.TODO(), dep, cleanupOptions)
	require.NoError(t, err, "Failed to create vertx instance")
	err = e2eutil.WaitForDeployment(t, fw.KubeClient, namespace, vertxDeploymentName, 1, retryInterval, timeout)
	// TODO add a check to make sure the sidecar has been injected
	require.NoError(t, err, "Failed waiting for"+vertxDeploymentName+" deployment")

	url, httpClient := getQueryURLAndHTTPClient(firstJaegerInstanceName, "%s/api/traces?service=order", true)
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
	secondJaegerInstanceName := "agent-as-sidecar2"
	secondJaegerInstance := createJaegerAgentAsSidecarInstance(secondJaegerInstanceName, namespace, testOtelAgent, testOtelAllInOne)
	defer undeployJaegerInstance(secondJaegerInstance)

	persisted := &appsv1.Deployment{}
	err = fw.Client.Get(goctx.TODO(), types.NamespacedName{
		Name:      vertxDeploymentName,
		Namespace: namespace,
	}, persisted)
	require.NoError(t, err, "Error getting jaeger instance")
	require.Equal(t, "agent-as-sidecar", persisted.Labels[inject.Label])

	err = fw.Client.Delete(goctx.TODO(), firstJaegerInstance)
	require.NoError(t, err, "Error deleting instance")
	err = e2eutil.WaitForDeletion(t, fw.Client.Client, firstJaegerInstance, retryInterval, timeout)
	require.NoError(t, err, "Error waiting for jaeger instance deletion")

	url, httpClient = getQueryURLAndHTTPClient(secondJaegerInstanceName, "%s/api/traces?service=order", true)
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
	verifyAgentImage(vertxDeploymentName, namespace, testOtelAgent)
}

func getVertxDefinition(deploymentName string, annotations map[string]string) *appsv1.Deployment {
	selector := map[string]string{"app": deploymentName}
	dep := &appsv1.Deployment{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "apps/v1",
			Kind:       "Deployment",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:        deploymentName,
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
						Name:  deploymentName,
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

func createJaegerAgentAsSidecarInstance(name, namespace string, useOtelAgent, useOtelAllInOne bool) *v1.Jaeger {
	cleanupOptions := &framework.CleanupOptions{TestContext: ctx, Timeout: timeout, RetryInterval: retryInterval}

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

	if useOtelAllInOne {
		logrus.Infof("Using OTEL AllInOne image for %s", name)
		j.Spec.AllInOne.Image = otelAllInOneImage
		j.Spec.AllInOne.Config = v1.NewFreeForm(getOtelConfigForHealthCheckPort("14269"))
	}

	if useOtelAgent {
		logrus.Infof("Using OTEL Agent for %s", name)
		j.Spec.Agent.Image = otelAgentImage
		j.Spec.Agent.Config = v1.NewFreeForm(getOtelConfigForHealthCheckPort("14269"))
	}

	err := fw.Client.Create(goctx.TODO(), j, cleanupOptions)
	require.NoError(t, err, "Failed to create jaeger instance")

	err = e2eutil.WaitForDeployment(t, fw.KubeClient, namespace, name, 1, retryInterval, timeout)
	require.NoError(t, err, "Error waiting for Jaeger instance deployment")

	return j
}
