// +build elasticsearch

package e2e

import (
	"context"
	"fmt"
	"io/ioutil"
	"net/http"
	"strings"
	"testing"

	"k8s.io/apimachinery/pkg/types"
	framework "github.com/operator-framework/operator-sdk/pkg/test"
	"github.com/operator-framework/operator-sdk/pkg/test/e2eutil"
	log "github.com/sirupsen/logrus"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
	"k8s.io/apimachinery/pkg/util/wait"
	v12 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/jaegertracing/jaeger-operator/pkg/apis/jaegertracing/v1"
)

type ElasticSearchTestSuite struct {
	suite.Suite
}

var esEnabled = false

func(suite *ElasticSearchTestSuite) SetupSuite() {
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

	if isOpenShift(t) {
		esServerUrls = "http://elasticsearch." + storageNamespace + ".svc.cluster.local:9200"
	}
}

func (suite *ElasticSearchTestSuite) TearDownSuite() {
	log.Info("Entering TearDownSuite()")
	ctx.Cleanup()
}

func TestElasticSearchSuite(t *testing.T) {
	suite.Run(t, new(ElasticSearchTestSuite))
}

func (suite *ElasticSearchTestSuite) SetupTest() {
	t = suite.T()
}

func (suite *ElasticSearchTestSuite) TestSparkDependenciesES() {
	storage := v1.JaegerStorageSpec{
		Type: "elasticsearch",
		Options: v1.NewOptions(map[string]interface{}{
			"es.server-urls": esServerUrls,
		}),
	}
	err := sparkTest(t, framework.Global, ctx, storage)
	require.NoError(t, err, "SparkTest failed")
}

func (suite *ElasticSearchTestSuite) TestSimpleProd() {
	err := WaitForStatefulset(t, fw.KubeClient, storageNamespace, "elasticsearch", retryInterval, timeout)
	require.NoError(t, err, "Error waiting for elasticsearch")

	// create jaeger custom resource
	exampleJaeger := getJaegerSimpleProdWithServerUrls()
	err = fw.Client.Create(context.TODO(), exampleJaeger, &framework.CleanupOptions{TestContext: ctx, Timeout: timeout, RetryInterval: retryInterval})
	require.NoError(t, err, "Error deploying example Jaeger")
	defer undeployJaegerInstance(exampleJaeger)

	err = e2eutil.WaitForDeployment(t, fw.KubeClient, namespace, "simple-prod-collector", 1, retryInterval, timeout)
	require.NoError(t, err, "Error waiting for collector deployment")

	err = e2eutil.WaitForDeployment(t, fw.KubeClient, namespace, "simple-prod-query", 1, retryInterval, timeout)
	require.NoError(t, err, "Error waiting for query deployment")

	portForw, closeChan := CreatePortForward(namespace, "simple-prod-query", "jaegertracing/jaeger-query", []string{"16686"}, fw.KubeConfig)
	defer portForw.Close()
	defer close(closeChan)

	portForwColl, closeChanColl := CreatePortForward(namespace, "simple-prod-collector", "jaegertracing/jaeger-collector", []string{"14268"}, fw.KubeConfig)
	require.NoError(t, err, "Error creating port forward")

	defer portForwColl.Close()
	defer close(closeChanColl)
	err = SmokeTest("http://localhost:16686/api/traces", "http://localhost:14268/api/traces", "foobar", retryInterval, timeout)
	require.NoError(t, err, "Error running smoketest")
}

func (suite *ElasticSearchTestSuite) TestEsIndexCleaner() {
	name := "test-es-index-cleaner"
	j := getJaegerAllInOne(name)

	err := fw.Client.Create(context.Background(), j, &framework.CleanupOptions{TestContext: ctx, Timeout: timeout, RetryInterval: retryInterval})
	require.NoError(t, err, "Error deploying Jaeger")
	defer undeployJaegerInstance(j)

	err = e2eutil.WaitForDeployment(t, fw.KubeClient, namespace, name, 1, retryInterval, timeout)
	require.NoError(t, err, "Error waiting for deployment")

	// create span, otherwise index cleaner fails - there would not be indices
	portForw, closeChan := CreatePortForward(namespace, name, "jaegertracing/all-in-one", []string{"16686", "14268"}, fw.KubeConfig)
	defer portForw.Close()
	defer close(closeChan)

	err = SmokeTest("http://localhost:16686/api/traces", "http://localhost:14268/api/traces", "foo-bar", retryInterval, timeout)
	require.NoError(t, err, "Error running smoketest")

	// Once we've created a span with the smoke test, enable the index cleaer
	key := types.NamespacedName{Name:name, Namespace:namespace}
	err = fw.Client.Get(context.Background(), key, j)
	require.NoError(t, err)
	esEnabled = true
	err = fw.Client.Update(context.Background(), j)
	require.NoError(t, err)

	portForwES, closeChanES := CreatePortForward(storageNamespace, "elasticsearch", "elasticsearch", []string{"9200"}, fw.KubeConfig)
	defer portForwES.Close()
	defer close(closeChanES)

	flag, err := hasIndexWithPrefix("jaeger-")
	require.NoError(t, err, "Error searching for index")
	require.True(t, flag, "HasIndexWithPrefix returned false")

	err = WaitForCronJob(t, fw.KubeClient, namespace, fmt.Sprintf("%s-es-index-cleaner", name), retryInterval, timeout)
	require.NoError(t, err, "Error waiting for Cron Job")

	err = WaitForJobOfAnOwner(t, fw.KubeClient, namespace, fmt.Sprintf("%s-es-index-cleaner", name), retryInterval, timeout)
	require.NoError(t, err, "Error waiting for Cron Job")

	err = wait.Poll(retryInterval, timeout, func() (done bool, err error) {
		flag, err := hasIndexWithPrefix("jaeger-")
		return !flag, err
	})
	require.NoError(t, err, "TODO")
}

func getJaegerSimpleProdWithServerUrls() *v1.Jaeger {
	exampleJaeger := &v1.Jaeger{
		TypeMeta: metav1.TypeMeta{
			Kind:       "Jaeger",
			APIVersion: "jaegertracing.io/v1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      "simple-prod",
			Namespace: namespace,
		},
		Spec: v1.JaegerSpec{
			Strategy: "production",
			Storage: v1.JaegerStorageSpec{
				Type: "elasticsearch",
				Options: v1.NewOptions(map[string]interface{}{
					"es.server-urls": esServerUrls,
				}),
			},
		},
	}
	return exampleJaeger
}

func getJaegerAllInOne(name string) *v1.Jaeger {
	numberOfDays := 0
	j := &v1.Jaeger{
		TypeMeta: v12.TypeMeta{
			Kind:       "Jaeger",
			APIVersion: "jaegertracing.io/v1",
		},
		ObjectMeta: v12.ObjectMeta{
			Name:      name,
			Namespace: namespace,
		},
		Spec: v1.JaegerSpec{
			Strategy: "allInOne",
			Storage: v1.JaegerStorageSpec{
				Type: "elasticsearch",
				Options: v1.NewOptions(map[string]interface{}{
					"es.server-urls": esServerUrls,
				}),
				EsIndexCleaner: v1.JaegerEsIndexCleanerSpec{
					Enabled:	&esEnabled,
					Schedule:     "*/1 * * * *",
					NumberOfDays: &numberOfDays,
				},
			},
		},
	}
	return j
}

func hasIndexWithPrefix(prefix string) (bool, error) {
	c := http.Client{}
	req, err := http.NewRequest(http.MethodGet, "http://localhost:9200/_cat/indices", nil)
	if err != nil {
		return false, err
	}
	resp, err := c.Do(req)
	if err != nil {
		return false, err
	}
	defer resp.Body.Close()

	bodyBytes, err := ioutil.ReadAll(resp.Body)
	bodyString := string(bodyBytes)

	return strings.Contains(bodyString, prefix), nil
}
