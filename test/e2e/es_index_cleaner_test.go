// +build elasticsearch

package e2e

import (
	"context"
	"fmt"
	"io/ioutil"
	"net/http"
	"strings"
	"testing"

	"github.com/stretchr/testify/suite"
	"github.com/stretchr/testify/require"
	framework "github.com/operator-framework/operator-sdk/pkg/test"
	"github.com/operator-framework/operator-sdk/pkg/test/e2eutil"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/wait"

	"github.com/jaegertracing/jaeger-operator/pkg/apis/jaegertracing/v1"
)

type EsIndexCleanerTestSuite struct {
	suite.Suite
}

func(suite *EsIndexCleanerTestSuite) SetupSuite() {
	t = suite.T()
	var err error
	ctx, err = prepare(t)
	if (err != nil) {
		ctx.Cleanup()
		require.FailNow(t, "Failed in prepare")
	}
	fw = framework.Global
	namespace, _ = ctx.GetNamespace()
	require.NotNil(t, namespace, "GetNamespace failed")

	addToFrameworkSchemeForSmokeTests(t)
}

func (suite *EsIndexCleanerTestSuite) TearDownSuite() {
	ctx.Cleanup()
}

func TestEsIndexCleanerSuite(t *testing.T) {
	suite.Run(t, new(EsIndexCleanerTestSuite))
}


func (suite *EsIndexCleanerTestSuite) TestEsIndexCleaner()  {
	name := "test-es-index-cleaner"
	numberOfDays := 0
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
			Strategy: "allInOne",
			Storage: v1.JaegerStorageSpec{
				Type: "elasticsearch",
				Options: v1.NewOptions(map[string]interface{}{
					"es.server-urls": esServerUrls,
				}),
				EsIndexCleaner: v1.JaegerEsIndexCleanerSpec{
					Schedule:     "*/1 * * * *",
					NumberOfDays: &numberOfDays,
				},
			},
		},
	}

	err := fw.Client.Create(context.Background(), j, &framework.CleanupOptions{TestContext: ctx, Timeout: timeout, RetryInterval: retryInterval})
	require.NoError(t, err, "Error deploying Jaeger")

	err = e2eutil.WaitForDeployment(t, fw.KubeClient, namespace, name, 1, retryInterval, timeout)
	require.NoError(t, err, "Error waiting for deployment")

	// create span, otherwise index cleaner fails - there would not be indices
	jaegerPod, err := GetPod(namespace, name, "jaegertracing/all-in-one", fw.KubeClient)
	require.NoError(t, err, "Error getting Pod")

	portForw, closeChan, err := CreatePortForward(namespace, jaegerPod.Name, []string{"16686", "14268"}, fw.KubeConfig)
	require.NoError(t, err, "Error creating port forward")

	defer portForw.Close()
	defer close(closeChan)

	err = SmokeTest("http://localhost:16686/api/traces", "http://localhost:14268/api/traces", "foo-bar", retryInterval, timeout)
	require.NoError(t, err, "Error running smoketest")

	esPod, err := GetPod(storageNamespace, "elasticsearch", "elasticsearch", fw.KubeClient)
	require.NoError(t, err, "Error getting Pod")

	portForwES, closeChanES, err := CreatePortForward(esPod.Namespace, esPod.Name, []string{"9200"}, fw.KubeConfig)
	require.NoError(t, err, "Error creating port forward")

	defer portForwES.Close()
	defer close(closeChanES)

	flag, err := hasIndexWithPrefix("jaeger-")
	require.NoError(t, err, "Error searching for index")
	require.True(t, flag, "HasIndexWithPrefix returned false")

	err = WaitForCronJob(t, fw.KubeClient, namespace, fmt.Sprintf("%s-es-index-cleaner", name), retryInterval, timeout)
	require.NoError(t, err, "Error waiting for Cron Job")

	err = WaitForJobOfAnOwner(t, fw.KubeClient, namespace, fmt.Sprintf("%s-es-index-cleaner", name), retryInterval, timeout)
	require.NoError(t, err, "Error waiting for Cron Job")

	err =  wait.Poll(retryInterval, timeout, func() (done bool, err error) {
		flag, err := hasIndexWithPrefix("jaeger-")
		return !flag, err
	})
	require.NoError(t, err, "TODO")
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
