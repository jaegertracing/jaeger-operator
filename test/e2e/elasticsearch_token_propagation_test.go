// +build token_propagation_elasticsearch

package e2e

import (
	goctx "context"
	"crypto/tls"
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
	"strings"
	"testing"
	"time"

	framework "github.com/operator-framework/operator-sdk/pkg/test"
	"github.com/operator-framework/operator-sdk/pkg/test/e2eutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
	"github.com/uber/jaeger-client-go/config"
	corev1 "k8s.io/api/core/v1"
	rbac "k8s.io/api/rbac/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/wait"

	"k8s.io/apimachinery/pkg/types"

	"github.com/jaegertracing/jaeger-operator/pkg/apis"
	v1 "github.com/jaegertracing/jaeger-operator/pkg/apis/jaegertracing/v1"
	esv1 "github.com/jaegertracing/jaeger-operator/pkg/storage/elasticsearch/v1"
)

// Test parameters
const name = "token-prop"
const username = "user-test-token"
const password = "any"
const collectorPodImageName = "jaeger-collector"
const testServiceName = "token-propagation"
const test_account = "token-test-user"

type TokenTestSuite struct {
	suite.Suite
	exampleJaeger        *v1.Jaeger
	queryName            string
	collectorName        string
	queryServiceEndPoint string
	host                 string
	token                string
}

func (suite *TokenTestSuite) SetupSuite() {
	t = suite.T()
	if !isOpenShift(t) {
		t.Skipf("Test %s is currently supported only on OpenShift because es-operator runs only on OpenShift\n", t.Name())
	}
	assert.NoError(t, framework.AddToFrameworkScheme(apis.AddToScheme, &v1.JaegerList{
		TypeMeta: metav1.TypeMeta{
			Kind:       "Jaeger",
			APIVersion: "jaegertracing.io/v1",
		},
	}))
	assert.NoError(t, framework.AddToFrameworkScheme(apis.AddToScheme, &esv1.ElasticsearchList{
		TypeMeta: metav1.TypeMeta{
			Kind:       "Elasticsearch",
			APIVersion: "logging.openshift.io/v1",
		},
	}))
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
	suite.deployJaegerWithPropagationEnabled()
}

func (suite *TokenTestSuite) TearDownSuite() {
	undeployJaegerInstance(suite.exampleJaeger)
	handleSuiteTearDown()
}

func (suite *TokenTestSuite) TestTokenPropagationNoToken() {
	client := &http.Client{
		Transport: &http.Transport{
			TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
		},
	}
	err := wait.Poll(retryInterval, timeout, func() (done bool, err error) {
		req, err := http.NewRequest(http.MethodGet, suite.queryServiceEndPoint, nil)
		resp, err := client.Do(req)
		defer resp.Body.Close()
		require.Equal(t, http.StatusForbidden, resp.StatusCode)
		return true, nil
	})
	require.NoError(t, err, "Token propagation test failed")
}

func (suite *TokenTestSuite) TestTokenPropagationValidToken() {
	/* Create an span */
	collectorPort := randomPortNumber()
	collectorPorts := []string{collectorPort + ":14268"}
	portForwColl, closeChanColl := CreatePortForward(namespace, suite.collectorName, collectorPodImageName, collectorPorts, fw.KubeConfig)
	defer portForwColl.Close()
	defer close(closeChanColl)
	collectorEndpoint := fmt.Sprintf("http://localhost:%s/api/traces", collectorPort)
	cfg := config.Configuration{
		Reporter:    &config.ReporterConfig{CollectorEndpoint: collectorEndpoint},
		Sampler:     &config.SamplerConfig{Type: "const", Param: 1},
		ServiceName: testServiceName,
	}
	tracer, closer, err := cfg.NewTracer()
	require.NoError(t, err, "Failed to create tracer in token propagation test")
	tStr := time.Now().Format(time.RFC3339Nano)
	tracer.StartSpan("TokenTest").
		SetTag("time-RFC3339Nano", tStr).
		Finish()
	closer.Close()
	client := newHTTPSClient()
	/* Try to reach query endpoint */
	err = wait.Poll(retryInterval, timeout, func() (done bool, err error) {
		req, err := http.NewRequest(http.MethodGet, suite.queryServiceEndPoint, nil)
		req.Header.Add("Authorization", "Bearer "+suite.token)
		resp, err := client.Do(req)
		defer resp.Body.Close()
		require.Equal(t, http.StatusOK, resp.StatusCode)
		if resp.StatusCode != http.StatusOK {
			return false, errors.New("Query service returns http code: " + string(resp.StatusCode))
		}
		bodyBytes, err := ioutil.ReadAll(resp.Body)
		bodyString := string(bodyBytes)
		if !strings.Contains(bodyString, "errors\":null") {
			return false, errors.New("query service returns errors: " + bodyString)
		}
		return true, nil
	})
	require.NoError(t, err, "Token propagation test failed")
}

func (suite *TokenTestSuite) deployJaegerWithPropagationEnabled() {
	queryName := fmt.Sprintf("%s-query", name)
	collectorName := fmt.Sprintf("%s-collector", name)
	bindOperatorWithAuthDelegator()
	createTestServiceAccount()
	suite.token = testAccountToken()

	suite.exampleJaeger = jaegerInstance()
	err := fw.Client.Create(goctx.Background(),
		suite.exampleJaeger,
		&framework.CleanupOptions{
			TestContext:   ctx,
			Timeout:       timeout,
			RetryInterval: retryInterval,
		})
	require.NoError(t, err, "Error deploying example Jaeger")

	err = e2eutil.WaitForDeployment(t, fw.KubeClient, namespace, collectorName, 1, retryInterval, timeout)
	require.NoError(t, err, "Error waiting for collector deployment")

	err = e2eutil.WaitForDeployment(t, fw.KubeClient, namespace, queryName, 1, retryInterval, timeout)
	require.NoError(t, err, "Error waiting for query deployment")

	route := findRoute(t, fw, name)

	suite.host = route.Spec.Host
	suite.queryServiceEndPoint = fmt.Sprintf("https://%s/api/traces?service=%s", suite.host, testServiceName)
}

func TestTokenSuite(t *testing.T) {
	suite.Run(t, new(TokenTestSuite))
}

func bindOperatorWithAuthDelegator() {
	roleBinding := rbac.ClusterRoleBinding{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "jaeger-operator:system:auth-delegator",
			Namespace: namespace,
		},
		Subjects: []rbac.Subject{{
			Kind:      "ServiceAccount",
			Name:      "jaeger-operator",
			Namespace: namespace,
		}},
		RoleRef: rbac.RoleRef{
			Kind: "ClusterRole",
			Name: "system:auth-delegator",
		},
	}
	err := fw.Client.Create(goctx.Background(),
		&roleBinding,
		&framework.CleanupOptions{
			TestContext:   ctx,
			Timeout:       timeout,
			RetryInterval: retryInterval,
		})
	require.NoError(t, err, "Error binding operator service account with auth-delegator")
}

func createTestServiceAccount() {

	serviceAccount := corev1.ServiceAccount{
		ObjectMeta: metav1.ObjectMeta{
			Name:      test_account,
			Namespace: namespace,
		},
	}
	err := fw.Client.Create(goctx.Background(),
		&serviceAccount,
		&framework.CleanupOptions{
			TestContext:   ctx,
			Timeout:       timeout,
			RetryInterval: retryInterval,
		})
	require.NoError(t, err, "Error deploying example Jaeger")

	roleBinding := rbac.ClusterRoleBinding{
		ObjectMeta: metav1.ObjectMeta{
			Name:      test_account + "-bind",
			Namespace: namespace,
		},
		Subjects: []rbac.Subject{{
			Kind:      "ServiceAccount",
			Name:      serviceAccount.Name,
			Namespace: namespace,
		}},
		RoleRef: rbac.RoleRef{
			Kind: "ClusterRole",
			Name: "cluster-admin",
		},
	}

	err = fw.Client.Create(goctx.Background(),
		&roleBinding,
		&framework.CleanupOptions{
			TestContext:   ctx,
			Timeout:       timeout,
			RetryInterval: retryInterval,
		})
	require.NoError(t, err, "Error deploying example Jaeger")

}

func testAccountToken() string {
	var secretName string
	err := wait.Poll(retryInterval, timeout, func() (done bool, err error) {
		serviceAccount := corev1.ServiceAccount{}
		e := fw.Client.Get(goctx.Background(), types.NamespacedName{
			Namespace: namespace,
			Name:      test_account,
		}, &serviceAccount)
		if e != nil {
			return false, e
		}
		for _, s := range serviceAccount.Secrets {
			if strings.HasPrefix(s.Name, test_account+"-token") {
				secretName = s.Name
				return true, nil
			}
		}
		return false, nil
	})
	require.NoError(t, err, "Error getting service account token")
	require.NotEmpty(t, secretName, "secret with token not found")
	secret := corev1.Secret{}
	err = fw.Client.Get(goctx.Background(), types.NamespacedName{
		Namespace: namespace,
		Name:      secretName,
	}, &secret)
	require.NoError(t, err, "Error deploying example Jaeger")
	return string(secret.Data["token"])
}

func jaegerInstance() *v1.Jaeger {
	exampleJaeger := &v1.Jaeger{
		TypeMeta: metav1.TypeMeta{
			Kind:       "Jaeger",
			APIVersion: "jaegertracing.io/v1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      "token-prop",
			Namespace: namespace,
		},
		Spec: v1.JaegerSpec{
			Strategy: "production",
			Storage: v1.JaegerStorageSpec{
				Type: "elasticsearch",
				Elasticsearch: v1.ElasticsearchSpec{
					NodeCount: 1,
					Resources: &corev1.ResourceRequirements{
						Limits:   corev1.ResourceList{corev1.ResourceMemory: resource.MustParse("1Gi")},
						Requests: corev1.ResourceList{corev1.ResourceMemory: resource.MustParse("1Gi")},
					},
				},
			},
			Query: v1.JaegerQuerySpec{
				Options: v1.NewOptions(map[string]interface{}{
					"es.version":                     "5",
					"query.bearer-token-propagation": "true",
					"es.tls":                         "false",
				}),
			},
			Ingress: v1.JaegerIngressSpec{
				Openshift: v1.JaegerIngressOpenShiftSpec{
					SAR:          "{\"namespace\": \"default\", \"resource\": \"pods\", \"verb\": \"get\"}",
					DelegateUrls: `{"/":{"namespace": "default", "resource": "pods", "verb": "get"}}`,
				},
				Options: v1.NewOptions(map[string]interface{}{
					"pass-access-token":      "true",
					"pass-user-bearer-token": "true",
					"scope":                  "user:info user:check-access user:list-projects",
					"pass-basic-auth":        "false",
				}),
			},
		},
	}
	return exampleJaeger
}

func newHTTPSClient() *http.Client {
	tr := &http.Transport{
		TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
	}
	client := &http.Client{
		Transport: tr,
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			return http.ErrUseLastResponse
		},
	}
	return client
}
