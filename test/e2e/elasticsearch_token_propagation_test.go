// +build token_propagation_elasticsearch

package e2e

import (
	"bytes"
	goctx "context"
	"crypto/tls"
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/http/cookiejar"
	"net/url"
	"strings"
	"testing"
	"time"

	framework "github.com/operator-framework/operator-sdk/pkg/test"
	"github.com/operator-framework/operator-sdk/pkg/test/e2eutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
	"github.com/uber/jaeger-client-go/config"
	"golang.org/x/net/html"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/wait"

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

type TokenPropagationTestSuite struct {
	suite.Suite
	exampleJaeger        *v1.Jaeger
	queryName            string
	collectorName        string
	queryServiceEndPoint string
	host                 string
}

func (suite *TokenPropagationTestSuite) SetupSuite() {
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

func (suite *TokenPropagationTestSuite) TearDownSuite() {
	// undeployJaegerInstance(suite.exampleJaeger)
	// handleSuiteTearDown()
}

func (suite *TokenPropagationTestSuite) TestTokenPropagationNoToken() {

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

func (suite *TokenPropagationTestSuite) TestTokenPropagationValidToken() {
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

	/* Get token using oauth OpenShift authorization */
	client, err := oAuthAuthorization(suite.host, username, password)

	/* Try to reach query endpoint */
	err = wait.Poll(retryInterval, timeout, func() (done bool, err error) {
		req, err := http.NewRequest(http.MethodGet, suite.queryServiceEndPoint, nil)
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

		return strings.Contains(bodyString, tStr), nil
		return true, nil
	})

	require.NoError(t, err, "Token propagation test failed")
}

func getESJaegerInstance() *v1.Jaeger {
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

func (suite *TokenPropagationTestSuite) deployJaegerWithPropagationEnabled() {
	queryName := fmt.Sprintf("%s-query", name)
	collectorName := fmt.Sprintf("%s-collector", name)

	// create jaeger custom resource
	suite.exampleJaeger = getESJaegerInstance()
	err := fw.Client.Create(goctx.TODO(),
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

func TestTokenPropagationSuite(t *testing.T) {
	suite.Run(t, new(TokenPropagationTestSuite))
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

func oAuthAuthorization(host, user, pass string) (*http.Client, error) {
	/* Setup client*/
	cookieJar, _ := cookiejar.New(nil)
	client := &http.Client{
		Transport: &http.Transport{
			TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
		},
		Jar: cookieJar,
	}
	/* Start oauth */
	resp, err := client.Get("https://" + host + "/oauth/start")
	defer resp.Body.Close()
	if err != nil {
		return nil, err
	}
	responseBytes, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	var req *http.Request
	/* Submit form */
	if hasForm(responseBytes) {
		req = getLoginFormRequest(responseBytes, resp.Request.URL, user, pass)
	} else {
		// OCP 4.2 or newer.
		// Choose idp
		link := getLinkToHtpassIDP(responseBytes)
		resp, err := client.Get("https://" + resp.Request.URL.Host + link)
		if err != nil {
			return nil, err
		}

		responseBytes, err := ioutil.ReadAll(resp.Body)
		defer resp.Body.Close()

		if err != nil {
			return nil, err
		}
		req = getLoginFormRequest(responseBytes, resp.Request.URL, user, pass)
	}
	resp, err = client.Do(req)
	defer resp.Body.Close()
	if resp.Request.URL.Path == "/oauth/authorize/approve" {
		req = submitGrantForm(resp)
		resp, err = client.Do(req)
		defer resp.Body.Close()
	}
	return client, nil
}

func hasForm(responseBytes []byte) bool {
	root, _ := html.Parse(bytes.NewBuffer(responseBytes))
	form := false
	visit(root, func(n *html.Node) {
		if n.Type == html.ElementNode && n.Data == "form" {
			form = true
		}
	})
	return form
}

func getLinkToHtpassIDP(responseBytes []byte) string {
	root, _ := html.Parse(bytes.NewBuffer(responseBytes))
	link := ""
	visit(root, func(n *html.Node) {
		if n.Type == html.ElementNode && n.Data == "a" {
			href := getAttr(n, "href")
			url, _ := url.Parse(href)
			if url.Query().Get("idp") == "htpasswd_provider" {
				link = href
			}
		}
	})
	return link
}

func getLoginFormRequest(responseBytes []byte, currentURL *url.URL, username, password string) *http.Request {
	reqHeader := http.Header{}
	action := ""
	formData := url.Values{}
	root, _ := html.Parse(bytes.NewBuffer(responseBytes))
	visit(root, func(n *html.Node) {
		if n.Type == html.ElementNode && n.Data == "input" {
			inputType := getAttr(n, "type")
			if inputType == "hidden" {
				name := getAttr(n, "name")
				value := getAttr(n, "value")
				formData.Add(name, value)
			}
		}
		if n.Type == html.ElementNode && n.Data == "form" {
			action = getAttr(n, "action")
		}
	})

	formData.Add("username", username)
	formData.Add("password", password)
	reqHeader.Set("Content-Type", "application/x-www-form-urlencoded")
	reqBody := strings.NewReader(formData.Encode())
	reqURL, _ := currentURL.Parse(action)
	req, _ := http.NewRequest("POST", reqURL.String(), reqBody)
	req.Header = reqHeader
	return req
}

func submitGrantForm(response *http.Response) *http.Request {
	reqHeader := http.Header{}
	action := ""
	formData := url.Values{}
	currentURL := response.Request.URL
	responseBytes, err := ioutil.ReadAll(response.Body)
	if err != nil {
		return nil
	}
	root, err := html.Parse(bytes.NewBuffer(responseBytes))
	visit(root, func(n *html.Node) {
		if n.Type == html.ElementNode && n.Data == "input" {
			inputType := getAttr(n, "type")
			if inputType == "hidden" || inputType == "checkbox" || inputType == "radio" {
				name := getAttr(n, "name")
				value := getAttr(n, "value")
				formData.Add(name, value)
			}
		}
		if n.Type == html.ElementNode && n.Data == "form" {
			action = getAttr(n, "action")
		}
	})
	formData.Add("approve", "Allow selected permissions")
	reqHeader.Set("Content-Type", "application/x-www-form-urlencoded")
	reqBody := strings.NewReader(formData.Encode())
	reqURL, _ := currentURL.Parse(action)
	req, _ := http.NewRequest("POST", reqURL.String(), reqBody)
	req.Header = reqHeader
	return req
}

func getAttr(element *html.Node, attrName string) string {
	for _, attr := range element.Attr {
		if attr.Key == attrName {
			return attr.Val
		}
	}
	return ""
}

func visit(n *html.Node, visitor func(*html.Node)) {
	visitor(n)
	for c := n.FirstChild; c != nil; c = c.NextSibling {
		visit(c, visitor)
	}
}
