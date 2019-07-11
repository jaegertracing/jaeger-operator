// +build smoke

package e2e

import (
	goctx "context"
	"encoding/json"
	"io/ioutil"
	"net/http"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
	framework "github.com/operator-framework/operator-sdk/pkg/test"
	"github.com/operator-framework/operator-sdk/pkg/test/e2eutil"
	log "github.com/sirupsen/logrus"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
	"k8s.io/apimachinery/pkg/util/wait"

	"github.com/jaegertracing/jaeger-operator/pkg/apis/jaegertracing/v1"
	"github.com/jaegertracing/jaeger-operator/pkg/inject"
)

type SidecarTestSuite struct {
	suite.Suite
}

func(suite *SidecarTestSuite) SetupSuite() {
	t = suite.T()
	var err error
	ctx, err = prepare(t)
	if (err != nil) {
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

func (suite *SidecarTestSuite) TearDownSuite() {
	log.Info("Entering TearDownSuite()")
	ctx.Cleanup()
}

func TestSidecarSuite(t *testing.T) {
	suite.Run(t, new(SidecarTestSuite))
}

func (suite *SidecarTestSuite) SetupTest() {
	t = suite.T()
}

// Sidecar runs a test with the agent as sidecar
func (suite *SidecarTestSuite) TestSidecar() {
	cleanupOptions := &framework.CleanupOptions{TestContext: ctx, Timeout: timeout, RetryInterval: retryInterval}

	j := getJaegerAgentAsSidecarDefinition(namespace)
	err := fw.Client.Create(goctx.TODO(), j, cleanupOptions)
	require.NoError(t, err, "Failed to create jaeger instance")
	defer undeployJaegerInstance(j)

	dep := getVertxDefinition(namespace)
	err = fw.Client.Create(goctx.TODO(), dep, cleanupOptions)
	require.NoError(t, err, "Failed to create vertx instance")

	err = e2eutil.WaitForDeployment(t, fw.KubeClient, namespace, "vertx-create-span-sidecar", 1, retryInterval, timeout)
	require.NoError(t, err, "Failed waiting for vertx-create-span-sidecar deployment")

	portForward, closeChan := CreatePortForward(namespace, "agent-as-sidecar", "jaegertracing/all-in-one", []string{"16686"}, fw.KubeConfig)
	defer portForward.Close()
	defer close(closeChan)

	url := "http://localhost:16686/api/traces?service=order"
	c := http.Client{Timeout: time.Second}

	req, err := http.NewRequest(http.MethodGet, url, nil)
	require.NoError(t, err, "Failed to create httpRequest")

	err = wait.Poll(retryInterval, timeout, func() (done bool, err error) {
		res, err := c.Do(req)
		if err != nil {
			return false, err
		}

		body, err := ioutil.ReadAll(res.Body)
		if err != nil {
			return false, err
		}

		resp := &resp{}
		err = json.Unmarshal(body, &resp)
		if err != nil {
			return false, err
		}

		return len(resp.Data) > 0, nil
	})
	require.NoError(t, err, "Failed waiting for expected content")
}

func getVertxDefinition(s string) *appsv1.Deployment {
	selector := map[string]string{"app": "vertx-create-span-sidecar"}
	dep := &appsv1.Deployment{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "apps/v1",
			Kind:       "Deployment",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:        "vertx-create-span-sidecar",
			Namespace:   namespace,
			Annotations: map[string]string{inject.Annotation: "true"},
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

func getJaegerAgentAsSidecarDefinition(namespace string) *v1.Jaeger {
	j := &v1.Jaeger{
		TypeMeta: metav1.TypeMeta{
			Kind:       "Jaeger",
			APIVersion: "jaegertracing.io/v1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      "agent-as-sidecar",
			Namespace: namespace,
		},
		Spec: v1.JaegerSpec{
			Strategy: "allInOne",
			AllInOne: v1.JaegerAllInOneSpec{},
			Agent: v1.JaegerAgentSpec{
				Options: v1.NewOptions(map[string]interface{}{
					"log-level": "debug",
				}),
			},
		},
	}
	return j
}
