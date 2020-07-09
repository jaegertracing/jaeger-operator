// +build smoke

package e2e

import (
	"context"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"strconv"
	"strings"
	"testing"
	"time"

	framework "github.com/operator-framework/operator-sdk/pkg/test"
	"github.com/operator-framework/operator-sdk/pkg/test/e2eutil"
	log "github.com/sirupsen/logrus"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/wait"

	v1 "github.com/jaegertracing/jaeger-operator/pkg/apis/jaegertracing/v1"
	"github.com/jaegertracing/jaeger-operator/pkg/util"
)

type MiscTestSuite struct {
	suite.Suite
}

func (suite *MiscTestSuite) SetupSuite() {
	t = suite.T()
	var err error
	ctx, err = prepare(t)
	if err != nil {
		if ctx != nil {
			ctx.Cleanup()
		}
		require.FailNow(t, "Failed in prepare with: "+err.Error())
	}
	fw = framework.Global
	namespace = ctx.GetID()
	require.NotNil(t, namespace, "GetID failed")

	addToFrameworkSchemeForSmokeTests(t)
}

func (suite *MiscTestSuite) TearDownSuite() {
	handleSuiteTearDown()
}

func TestMiscSuite(t *testing.T) {
	suite.Run(t, new(MiscTestSuite))
}

func (suite *MiscTestSuite) SetupTest() {
	t = suite.T()
}

func (suite *MiscTestSuite) AfterTest(suiteName, testName string) {
	if debugMode && t.Failed() {
		log.Errorf("Test %s failed", t.Name())
	}
}

// Confirms fix for https://github.com/jaegertracing/jaeger-operator/issues/670.  Deploy an
// invalid CR, delete it, and make sure the operator responds properly.
func (suite *MiscTestSuite) TestDeleteResource() {
	jaegerInstanceName := "invalid-jaeger"
	jaegerInstance := getInvalidJaeger(jaegerInstanceName, namespace)
	cleanupOptions := &framework.CleanupOptions{TestContext: ctx, Timeout: timeout, RetryInterval: retryInterval}
	err := fw.Client.Create(context.Background(), jaegerInstance, cleanupOptions)
	require.NoError(t, err, "Error deploying invalid Jaeger")
	time.Sleep(5 * time.Second)

	undeployJaegerInstance(jaegerInstance)
	time.Sleep(5 * time.Second) // Give operator long enough to write to its log

	logs := getLogsForNamespace(getJaegerOperatorNamespace(), "name=jaeger-operator", "operator")
	operatorLog := logs["operator.log"]
	require.Contains(t, operatorLog, "\"Deployment has been removed.\" name="+jaegerInstanceName)
	require.Contains(t, operatorLog, "level=error msg=\"failed to apply the changes\" error=\"deployment has been removed\"")
}

// Make sure we're testing correct image
func (suite *MiscTestSuite) TestValidateBuildImage() {
	// TODO reinstate this if we come up with a good solution, but skip for now when using OLM installed operators
	if usingOLM {
		t.Skip()
	}

	buildImage := os.Getenv("BUILD_IMAGE")
	require.NotEmptyf(t, buildImage, "BUILD_IMAGE must be defined")

	// If we did the normal test setup the operator will be in the current namespace.  If not we need to iterate
	// over all namespaces to find it, being sure to match on WATCH_NAMESPACE
	if usingOLM {
		validateBuildImageInCluster(buildImage)
	} else {
		validateBuildImageInTestNamespace(buildImage)
	}
}

// This is a test of the findRoute function, not a product test
func (suite *MiscTestSuite) TestFindRoute() {
	if !isOpenShift(t) {
		t.Skip("This test only runs on Openshift")
	}
	cleanupOptions := &framework.CleanupOptions{TestContext: ctx, Timeout: timeout, RetryInterval: retryInterval}

	jaegerInstanceName := "simplest"
	jaegerInstance := getSimplestJaeger(jaegerInstanceName, namespace)
	err := fw.Client.Create(context.Background(), jaegerInstance, cleanupOptions)
	require.NoError(t, err, "Error deploying example Jaeger")
	defer undeployJaegerInstance(jaegerInstance)

	// Create a second namespace and deploy another instance named "simplest"
	secondContext, err := prepare(t)
	if err != nil {
		if secondContext != nil {
			secondContext.Cleanup()
		}
		require.FailNow(t, "Failed in prepare with: "+err.Error())
	}
	defer secondContext.Cleanup()

	secondJaegerInstanceName := jaegerInstanceName + "but-even-longer"
	secondNamespace := secondContext.GetID()
	secondJaegerInstance := getSimplestJaeger(secondJaegerInstanceName, secondNamespace)
	err = fw.Client.Create(context.Background(), secondJaegerInstance, cleanupOptions)
	require.NoError(t, err, "Error deploying example Jaeger")
	defer undeployJaegerInstance(secondJaegerInstance)

	err = e2eutil.WaitForDeployment(t, fw.KubeClient, namespace, jaegerInstanceName, 1, retryInterval, timeout)
	require.NoError(t, err, "Error waiting for deployment")

	err = e2eutil.WaitForDeployment(t, fw.KubeClient, secondNamespace, secondJaegerInstanceName, 1, retryInterval, timeout)
	require.NoError(t, err, "Error waiting for deployment")

	// Make sure findRoute returns the correct routes for each namespace
	route := findRoute(t, fw, jaegerInstanceName, namespace)
	require.Equal(t, namespace, route.Namespace)
	truncatedInstanceName := util.DNSName(util.Truncate(jaegerInstanceName, 62-len(namespace)))
	require.True(t, strings.HasPrefix(route.Spec.Host, truncatedInstanceName))
	require.True(t, strings.Contains(route.Spec.Host, namespace))

	secondRoute := findRoute(t, fw, secondJaegerInstanceName, secondNamespace)
	require.Equal(t, secondNamespace, secondRoute.Namespace)
	secondTruncatedInstanceName := util.DNSName(util.Truncate(secondJaegerInstanceName, 62-len(secondNamespace)))
	require.True(t, strings.HasPrefix(secondRoute.Spec.Host, secondTruncatedInstanceName))
	require.True(t, strings.Contains(secondRoute.Spec.Host, secondNamespace))
	require.False(t, secondTruncatedInstanceName == secondJaegerInstanceName)
}

func (suite *MiscTestSuite) TestBasicOAuth() {
	if !isOpenShift(t) {
		t.Skip("This test only runs on Openshift")
	}

	username := "e2etestuser"
	password := "befuddled"
	// To update this, use the output from: htpasswd -nbs e2etestuser befuddled
	userPasswordHash := "e2etestuser:{SHA}MfHY85GCR7WcTE7cQ2CGmXg9uTA="

	userPasswordSecret := make(map[string][]byte)
	userPasswordSecret["htpasswd"] = []byte(userPasswordHash)
	secret := createSecret("test-oauth-secret", namespace, userPasswordSecret)

	jaegerInstanceName := "test-oauth"
	jaeger := jaegerWithPassword(namespace, jaegerInstanceName, secret.Name)
	err := fw.Client.Create(context.Background(), jaeger, &framework.CleanupOptions{TestContext: ctx, Timeout: timeout, RetryInterval: retryInterval})
	require.NoError(t, err, "Error deploying jaeger")
	defer undeployJaegerInstance(jaeger)

	err = e2eutil.WaitForDeployment(t, fw.KubeClient, namespace, jaegerInstanceName, 1, retryInterval, timeout)
	require.NoError(t, err, "Error waiting for Jaeger deployment")

	urlPattern := "%s/api/services"
	route := findRoute(t, fw, jaegerInstanceName, namespace)
	require.Len(t, route.Status.Ingress, 1, "Wrong number of ingresses.")
	url := fmt.Sprintf("https://"+urlPattern, route.Spec.Host)
	transport := &http.Transport{
		TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
	}
	httpClient := http.Client{Timeout: 30 * time.Second, Transport: transport}
	log.Infof("Using Query URL [%v]", url)

	// A request without credentials should return a 403
	request, err := http.NewRequest(http.MethodGet, url, nil)
	require.NoError(t, err, "Failed to create httpRequest")
	response := &http.Response{}
	err = wait.Poll(retryInterval, timeout, func() (done bool, err error) {
		response, err = httpClient.Do(request)
		require.NoError(t, err)

		if response.StatusCode == 403 {
			return true, nil
		}
		if response.StatusCode == 503 {
			log.Info("Ignoring http response status 503")
			return false, nil
		}
		require.Failf(t, "Expected status 403 or 503 but got %d", strconv.Itoa(response.StatusCode))
		return false, nil
	})
	require.NoError(t, err)
	require.Equal(t, 403, response.StatusCode)

	// Add basic auth to the request and retry
	request.SetBasicAuth(username, password)
	err = wait.Poll(retryInterval, timeout, func() (done bool, err error) {
		response, err := httpClient.Do(request)
		require.NoError(t, err)
		require.Equal(t, 200, response.StatusCode)

		body, err := ioutil.ReadAll(response.Body)
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
}

func jaegerWithPassword(namespace string, instanceName, secretName string) *v1.Jaeger {
	volumes := getVolumes(secretName)
	volumeMounts := getVolumeMounts()

	j := &v1.Jaeger{
		TypeMeta: metav1.TypeMeta{
			Kind:       "Jaeger",
			APIVersion: "jaegertracing.io/v1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      instanceName,
			Namespace: namespace,
		},
		Spec: v1.JaegerSpec{
			Ingress: v1.JaegerIngressSpec{
				Openshift: v1.JaegerIngressOpenShiftSpec{
					SAR:          "{\"namespace\": " + "\"" + namespace + "\"" + ", \"resource\": \"pods\", \"verb\": \"get\"}",
					HtpasswdFile: "/usr/local/data/htpasswd",
				},
			},
			JaegerCommonSpec: v1.JaegerCommonSpec{
				Volumes:      volumes,
				VolumeMounts: volumeMounts,
			},
		},
	}

	return j
}

func getVolumeMounts() []corev1.VolumeMount {
	htpasswdVolume := corev1.VolumeMount{
		Name:      "htpasswd-volume",
		MountPath: "/usr/local/data",
	}

	volumeMounts := []corev1.VolumeMount{
		htpasswdVolume,
	}

	return volumeMounts
}

func getVolumes(secretName string) []corev1.Volume {
	htpasswdSecretName := corev1.SecretVolumeSource{
		SecretName: secretName,
	}

	htpasswdVolume := corev1.Volume{
		Name: "htpasswd-volume",
		VolumeSource: corev1.VolumeSource{
			Secret: &htpasswdSecretName,
		},
	}

	volumes := []corev1.Volume{
		htpasswdVolume,
	}

	return volumes
}

func validateBuildImageInCluster(buildImage string) {
	emptyOptions := new(metav1.ListOptions)
	namespaces, err := fw.KubeClient.CoreV1().Namespaces().List(context.Background(), *emptyOptions)
	require.NoError(t, err)
	found := false
	for _, item := range namespaces.Items {
		imagesMap, err := getJaegerOperatorImages(fw.KubeClient, item.Name)
		require.NoError(t, err)

		if len(imagesMap) > 0 {
			watchNamespace, ok := imagesMap[buildImage]
			if ok {
				if len(watchNamespace) == 0 || strings.Contains(watchNamespace, item.Name) {
					found = true
					t.Logf("Using jaeger-operator image(s) %s\n", imagesMap)
					break
				}
			}
		}
	}
	require.Truef(t, found, "Could not find an operator with image %s", buildImage)
}

func validateBuildImageInTestNamespace(buildImage string) {
	imagesMap, err := getJaegerOperatorImages(fw.KubeClient, namespace)
	require.NoError(t, err)
	require.Len(t, imagesMap, 1, "Expected 1 deployed operator")
	_, ok := imagesMap[buildImage]
	require.Truef(t, ok, "Expected operator image %s not found in map %s\n", buildImage, imagesMap)
	t.Logf("Using jaeger-operator image(s) %s\n", imagesMap)
}

func getSimplestJaeger(jaegerInstanceName, namespace string) *v1.Jaeger {
	jaeger := &v1.Jaeger{
		TypeMeta: metav1.TypeMeta{
			Kind:       "Jaeger",
			APIVersion: "jaegertracing.io/v1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      jaegerInstanceName,
			Namespace: namespace,
		},
	}

	return jaeger
}

func getInvalidJaeger(jaegerInstanceName, namespace string) *v1.Jaeger {
	jaeger := &v1.Jaeger{
		TypeMeta: metav1.TypeMeta{
			Kind:       "Jaeger",
			APIVersion: "jaegertracing.io/v1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      jaegerInstanceName,
			Namespace: namespace,
		},
		Spec: v1.JaegerSpec{
			AllInOne: v1.JaegerAllInOneSpec{
				Options: v1.NewOptions(map[string]interface{}{
					"invalidoptions": "invalid",
				}),
			},
		},
	}

	return jaeger
}
