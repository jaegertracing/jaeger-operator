// +build plugin

package e2e

import (
	goctx "context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"testing"

	framework "github.com/operator-framework/operator-sdk/pkg/test"
	"github.com/operator-framework/operator-sdk/pkg/test/e2eutil"
	log "github.com/sirupsen/logrus"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	v1 "github.com/jaegertracing/jaeger-operator/pkg/apis/jaegertracing/v1"
)

type PluginTestSuite struct {
	suite.Suite
}

func (suite *PluginTestSuite) SetupSuite() {
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
	require.NotEmpty(t, namespace, "GetID failed")

	addToFrameworkSchemeForSmokeTests(t)
}

func (suite *PluginTestSuite) TearDownSuite() {
	handleSuiteTearDown()
}

func TestPluginSuite(t *testing.T) {
	suite.Run(t, new(PluginTestSuite))
}

func (suite *PluginTestSuite) SetupTest() {
	t = suite.T()
}

func (suite *PluginTestSuite) AfterTest(suiteName, testName string) {
	handleTestFailure()
}

func (suite *PluginTestSuite) TestPlugin() {
	pluginImage := os.Getenv("DEMO_STORAGE_PLUGIN_IMAGE")
	require.NotEmpty(t, pluginImage, "DEMO_STORAGE_PLUGIN_IMAGE must be set")

	// create jaeger custom resource
	name := "my-jaeger"
	exampleJaeger := getJaegerPluginDefinition(namespace, name, pluginImage)

	secret := &corev1.Secret{
		TypeMeta: metav1.TypeMeta{
			Kind:       "Secret",
			APIVersion: "v1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Namespace: namespace,
			Name:      "plugin-config",
		},
		StringData: map[string]string{
			"config.json": "{}",
		},
	}
	err := fw.Client.Create(goctx.TODO(), secret, &framework.CleanupOptions{TestContext: ctx, Timeout: timeout, RetryInterval: retryInterval})
	require.NoError(t, err, "Error deploying secret")

	log.Infof("passing %v", exampleJaeger)
	err = fw.Client.Create(goctx.TODO(), exampleJaeger, &framework.CleanupOptions{TestContext: ctx, Timeout: timeout, RetryInterval: retryInterval})
	require.NoError(t, err, "Error deploying example Jaeger")
	defer undeployJaegerInstance(exampleJaeger)

	err = e2eutil.WaitForDeployment(t, fw.KubeClient, namespace, name, 1, retryInterval, timeout)
	require.NoError(t, err, "Error waiting for deployment")

	ports := []string{"0:16686"}
	portForward, closeChan := CreatePortForward(namespace, name, "all-in-one", ports, fw.KubeConfig)
	defer portForward.Close()
	defer close(closeChan)
	forwardedPorts, err := portForward.GetPorts()
	require.NoError(t, err)

	// Check that the plugins hard coded services response is there
	url := fmt.Sprintf("http://localhost:%d/api/services", forwardedPorts[0].Local)
	err = WaitAndPollForHTTPResponse(url, func(response *http.Response) (bool, error) {
		body, err := ioutil.ReadAll(response.Body)
		require.NoError(t, err)

		resp := &services{}
		err = json.Unmarshal(body, &resp)
		if err != nil {
			return false, nil
		}

		for _, v := range resp.Data {
			if v == "dummy1" { // hardcoded service
				return true, nil
			}
		}

		return false, nil
	})
	require.NoError(t, err, "Failed waiting for expected content")
}

func getJaegerPluginDefinition(namespace, name, pluginImage string) *v1.Jaeger {
	exampleJaeger := &v1.Jaeger{
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
			Storage: v1.JaegerStorageSpec{
				Type: "grpc-plugin",
				GRPCPlugin: v1.GRPCStoragePluginSpec{
					Image:             pluginImage,
					Binary:            "/plugin/demo-storage-plugin",
					ConfigurationFile: "/etc/plugin-config/config.json",
				},
			},
			JaegerCommonSpec: v1.JaegerCommonSpec{
				Volumes: []corev1.Volume{
					{
						Name: "plugin-config-volume",
						VolumeSource: corev1.VolumeSource{
							Secret: &corev1.SecretVolumeSource{
								SecretName: "plugin-config",
							},
						},
					},
				},
				VolumeMounts: []corev1.VolumeMount{
					{
						Name:      "plugin-config-volume",
						MountPath: "/etc/plugin-config",
					},
				},
			},
		},
	}
	return exampleJaeger
}
