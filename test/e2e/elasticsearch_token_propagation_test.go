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
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
	"github.com/uber/jaeger-client-go/config"
	corev1 "k8s.io/api/core/v1"
	rbac "k8s.io/api/rbac/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/wait"

	"github.com/jaegertracing/jaeger-operator/pkg/account"
	v1 "github.com/jaegertracing/jaeger-operator/pkg/apis/jaegertracing/v1"
)

// Test parameters
const name = "tokenprop"
const collectorPodImageName = "jaeger-collector"
const testServiceName = "token-propagation"
const testAccount = "token-test-user"

type TokenTestSuite struct {
	suite.Suite
	exampleJaeger         *v1.Jaeger
	queryName             string
	collectorName         string
	queryServiceEndPoint  string
	host                  string
	token                 string
	testServiceAccount    *corev1.ServiceAccount
	testRoleBinding       *rbac.ClusterRoleBinding
	delegationRoleBinding *rbac.ClusterRoleBinding
}

func (suite *TokenTestSuite) SetupSuite() {
	t = suite.T()
	if !isOpenShift(t) {
		t.Skipf("Test %s is currently supported only on OpenShift because es-operator runs only on OpenShift\n", t.Name())
	}
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
	suite.deployJaegerWithPropagationEnabled()
}

func (suite *TokenTestSuite) cleanAccountBindings() {
	if !debugMode || !t.Failed() {
		err := fw.Client.Delete(goctx.Background(), suite.testServiceAccount)
		require.NoError(t, err, "Error deleting test service account")
		err = e2eutil.WaitForDeletion(t, fw.Client.Client, suite.testServiceAccount, retryInterval, timeout)
		require.NoError(t, err)

		err = fw.Client.Delete(goctx.Background(), suite.testRoleBinding)
		require.NoError(t, err, "Error deleting test service account bindings")
		err = e2eutil.WaitForDeletion(t, fw.Client.Client, suite.testRoleBinding, retryInterval, timeout)
		require.NoError(t, err)

		err = fw.Client.Delete(goctx.Background(), suite.delegationRoleBinding)
		require.NoError(t, err, "Error deleting delegation bindings")
		err = e2eutil.WaitForDeletion(t, fw.Client.Client, suite.delegationRoleBinding, retryInterval, timeout)
		require.NoError(t, err)
	}
}

func (suite *TokenTestSuite) TearDownSuite() {
	undeployJaegerInstance(suite.exampleJaeger)
	suite.cleanAccountBindings()
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
		if err != nil {
			return false, err
		}
		resp, err := client.Do(req)
		defer resp.Body.Close()
		if err != nil {
			return false, err
		}
		if resp.StatusCode == http.StatusServiceUnavailable {
			logrus.Warnf("Ignoring http response %d", resp.StatusCode)
			return false, nil
		}
		if resp.StatusCode == http.StatusForbidden {
			return true, nil
		}
		return false, errors.New(fmt.Sprintf("query service return http code: %d", resp.StatusCode))
	})
	require.NoError(t, err, "Token propagation test failed")
}

func (suite *TokenTestSuite) TestTokenPropagationValidToken() {
	/* Create an span */
	portForwColl, closeChanColl := CreatePortForward(namespace, suite.collectorName, collectorPodImageName, []string{"0:14268"}, fw.KubeConfig)
	defer portForwColl.Close()
	defer close(closeChanColl)

	forwardedPorts, err := portForwColl.GetPorts()
	require.NoError(t, err)
	collectorPort := forwardedPorts[0].Local

	collectorEndpoint := fmt.Sprintf("http://localhost:%d/api/traces", collectorPort)

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
		if err != nil {
			return false, err
		}
		req.Header.Add("Authorization", "Bearer "+suite.token)
		resp, err := client.Do(req)
		defer resp.Body.Close()
		if err != nil {
			return false, err
		}
		if resp.StatusCode == http.StatusOK {
			bodyBytes, err := ioutil.ReadAll(resp.Body)
			require.NoError(t, err)
			bodyString := string(bodyBytes)
			if !strings.Contains(bodyString, "errors\":null") {
				return false, errors.New("query service returns errors: " + bodyString)
			}
			return true, nil
		}
		return false, errors.New(fmt.Sprintf("query service return http code: %d", resp.StatusCode))
	})
	require.NoError(t, err, "Token propagation test failed")
}

func (suite *TokenTestSuite) deployJaegerWithPropagationEnabled() {
	queryName := fmt.Sprintf("%s-query", name)
	collectorName := fmt.Sprintf("%s-collector", name)
	suite.exampleJaeger = jaegerInstance()

	suite.bindOperatorWithAuthDelegator()
	suite.createTestServiceAccount()
	suite.token = testAccountToken()
	require.NotEmpty(t, suite.token)
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

	route := findRoute(t, fw, name, namespace)

	suite.host = route.Spec.Host
	suite.queryServiceEndPoint = fmt.Sprintf("https://%s/api/traces?service=%s", suite.host, testServiceName)
}

func TestTokenSuite(t *testing.T) {
	suite.Run(t, new(TokenTestSuite))
}

func (suite *TokenTestSuite) bindOperatorWithAuthDelegator() {

	operatorNamespace := getJaegerOperatorNamespace()

	suite.delegationRoleBinding = &rbac.ClusterRoleBinding{
		ObjectMeta: metav1.ObjectMeta{
			Name:      operatorNamespace + "jaeger-operator:system:auth-delegator",
			Namespace: operatorNamespace,
		},
		Subjects: []rbac.Subject{{
			Kind:      "ServiceAccount",
			Name:      "jaeger-operator",
			Namespace: operatorNamespace,
		}},
		RoleRef: rbac.RoleRef{
			Kind: "ClusterRole",
			Name: "system:auth-delegator",
		},
	}
	err := fw.Client.Create(goctx.Background(),
		suite.delegationRoleBinding,
		&framework.CleanupOptions{
			TestContext:   ctx,
			Timeout:       timeout,
			RetryInterval: retryInterval,
		})
	require.NoError(t, err, "Error binding operator service account with auth-delegator")

	suite.delegationRoleBinding = &rbac.ClusterRoleBinding{
		ObjectMeta: metav1.ObjectMeta{
			Name:      operatorNamespace + "proxy:system:auth-delegator",
			Namespace: operatorNamespace,
		},
		Subjects: []rbac.Subject{{
			Kind:      "ServiceAccount",
			Name:      account.OAuthProxyAccountNameFor(suite.exampleJaeger),
			Namespace: operatorNamespace,
		}},
		RoleRef: rbac.RoleRef{
			Kind: "ClusterRole",
			Name: "system:auth-delegator",
		},
	}
	err = fw.Client.Create(goctx.Background(),
		suite.delegationRoleBinding,
		&framework.CleanupOptions{
			TestContext:   ctx,
			Timeout:       timeout,
			RetryInterval: retryInterval,
		})
	require.NoError(t, err, "Error binding operator service account with auth-delegator")
}

func (suite *TokenTestSuite) createTestServiceAccount() {

	suite.testServiceAccount = &corev1.ServiceAccount{
		ObjectMeta: metav1.ObjectMeta{
			Name:      namespace + testAccount,
			Namespace: namespace,
		},
	}
	err := fw.Client.Create(goctx.Background(),
		suite.testServiceAccount,
		&framework.CleanupOptions{
			TestContext:   ctx,
			Timeout:       timeout,
			RetryInterval: retryInterval,
		})
	require.NoError(t, err, "Error deploying example Jaeger")

	suite.testRoleBinding = &rbac.ClusterRoleBinding{
		ObjectMeta: metav1.ObjectMeta{
			Name:      namespace + testAccount + "-bind",
			Namespace: namespace,
		},
		Subjects: []rbac.Subject{{
			Kind:      "ServiceAccount",
			Name:      suite.testServiceAccount.Name,
			Namespace: namespace,
		}},
		RoleRef: rbac.RoleRef{
			Kind: "ClusterRole",
			Name: "cluster-admin",
		},
	}

	err = fw.Client.Create(goctx.Background(),
		suite.testRoleBinding,
		&framework.CleanupOptions{
			TestContext:   ctx,
			Timeout:       timeout,
			RetryInterval: retryInterval,
		})
	require.NoError(t, err, "Error deploying example Jaeger")

}

func testAccountToken() string {
	token := ""
	err := wait.Poll(retryInterval, timeout, func() (done bool, err error) {
		serviceAccount := corev1.ServiceAccount{}
		e := fw.Client.Get(goctx.Background(), types.NamespacedName{
			Namespace: namespace,
			Name:      namespace + testAccount,
		}, &serviceAccount)
		if e != nil {
			return false, e
		}
		for _, s := range serviceAccount.Secrets {
			secret := corev1.Secret{}
			err = fw.Client.Get(goctx.Background(), types.NamespacedName{
				Namespace: namespace,
				Name:      s.Name,
			}, &secret)
			if secret.Type == corev1.SecretTypeServiceAccountToken {
				token = string(secret.Data["token"])
				return true, nil
			}
		}
		return false, nil
	})
	require.NoError(t, err, "Error getting service account token")
	return token

}

func jaegerInstance() *v1.Jaeger {
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
					"es.tls.enabled":                 "false",
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
