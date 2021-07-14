// +build elasticsearch

package e2e

import (
	"context"
	"fmt"
	"sort"
	"testing"
	"time"

	framework "github.com/operator-framework/operator-sdk/pkg/test"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"

	v1 "github.com/jaegertracing/jaeger-operator/pkg/apis/jaegertracing/v1"
)

type ElasticSearchIndexTestSuite struct {
	suite.Suite
	esIndexCleanerHistoryDays int    // generate spans and services history
	esNamespace               string // default storage namespace location
}

// esIndexData struct is used to keep index data in simple format
// will be useful for the validations

func TestElasticSearchIndexSuite(t *testing.T) {
	indexSuite := new(ElasticSearchIndexTestSuite)
	// update default values
	indexSuite.esIndexCleanerHistoryDays = 45
	suite.Run(t, indexSuite)
}

func (suite *ElasticSearchIndexTestSuite) SetupSuite() {
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

func (suite *ElasticSearchIndexTestSuite) TearDownSuite() {
	handleSuiteTearDown()
}

func (suite *ElasticSearchIndexTestSuite) SetupTest() {
	t = suite.T()
	// update storage namespace
	if skipESExternal {
		suite.esNamespace = namespace
	} else {
		suite.esNamespace = storageNamespace
	}
	// delete indices from external elasticsearch node
	if !skipESExternal {
		DeleteEsIndices(suite.esNamespace)
	}
}

func (suite *ElasticSearchIndexTestSuite) AfterTest(suiteName, testName string) {
	handleTestFailure()
}

// executes es index cleaner with default index prefix
func (suite *ElasticSearchIndexTestSuite) TestEsIndexCleaner() {
	suite.runIndexCleaner("", []int{45, 30, 7, 1, 0})
}

// executes es index cleaner tests with custom index prefix
func (suite *ElasticSearchIndexTestSuite) TestEsIndexCleanerWithIndexPrefix() {
	suite.runIndexCleaner("my-custom_prefix", []int{3, 1, 0})
}

// executes index cleaner tests
func (suite *ElasticSearchIndexTestSuite) runIndexCleaner(esIndexPrefix string, daysRange []int) {
	logrus.Infof("index cleaner test started. daysRange=%v, prefix=%s", daysRange, esIndexPrefix)
	jaegerInstanceName := "test-es-index-cleaner"
	if esIndexPrefix != "" {
		jaegerInstanceName = "test-es-index-cleaner-with-prefix"
	}
	// get jaeger CR to create jaeger services
	jaegerInstance := GetJaegerSelfProvSimpleProdCR(jaegerInstanceName, namespace, 1)

	// If there is an external es deployment use it instead of creating a self provision one
	if !skipESExternal {
		if isOpenShift(t) {
			esServerUrls = "http://elasticsearch." + storageNamespace + ".svc.cluster.local:9200"
		}
		jaegerInstance.Spec.Storage = v1.JaegerStorageSpec{
			Type: v1.JaegerESStorage,
			Options: v1.NewOptions(map[string]interface{}{
				"es.server-urls": esServerUrls,
			}),
		}
	}

	// update jaeger CR with index cleaner specifications
	// initially disable es index cleaner job
	esIndexCleanerEnabled := false
	esIndexCleanerNumberOfDays := suite.esIndexCleanerHistoryDays
	jaegerInstance.Spec.Storage.EsIndexCleaner.Enabled = &esIndexCleanerEnabled
	jaegerInstance.Spec.Storage.EsIndexCleaner.NumberOfDays = &esIndexCleanerNumberOfDays
	jaegerInstance.Spec.Storage.EsIndexCleaner.Schedule = "*/1 * * * *"
	// update es.index-prefix, if supplied
	if esIndexPrefix != "" {
		if jaegerInstance.Spec.Storage.Options.Map() == nil {
			jaegerInstance.Spec.Storage.Options = v1.NewOptions(map[string]interface{}{})
		}
		jaegerInstance.Spec.Storage.Options.Map()["es.index-prefix"] = esIndexPrefix
	}

	logrus.Infof("Creating jaeger services for es index cleaner test: %s", jaegerInstanceName)
	createESSelfProvDeployment(jaegerInstance, jaegerInstanceName, namespace)
	defer undeployJaegerInstance(jaegerInstance)

	GenerateSpansHistory(namespace, jaegerInstanceName, "span-index-cleaner", ElasticSearchIndexDateLayout, suite.esIndexCleanerHistoryDays)

	suite.triggerIndexCleanerAndVerifyIndices(jaegerInstance, esIndexPrefix, daysRange)

}

// function to validate indices
func (suite *ElasticSearchIndexTestSuite) assertIndex(esIndexPrefix string, indices []esIndexData, verifyDateAfter time.Time, count int) {
	// sort and print indices
	sort.Slice(indices, func(i, j int) bool {
		return indices[i].Date.After(indices[j].Date)
	})
	indicesSlice := make([]string, 0)
	for _, ind := range indices {
		indicesSlice = append(indicesSlice, ind.IndexName)
	}
	logrus.Infof("indices should be after %v, indices list: %v", verifyDateAfter, indicesSlice)
	require.Equal(t, count, len(indices), "number of available indices not matching, %v", indices)
	for _, index := range indices {
		require.True(t, index.Date.After(verifyDateAfter), "this index must removed by index cleaner job: %v", index)
		require.Equal(t, esIndexPrefix, index.Prefix, "index prefix not matching")
	}
}

// trigger the index cleaner job for the given day range and verifies the indices availability
func (suite *ElasticSearchIndexTestSuite) triggerIndexCleanerAndVerifyIndices(jaegerInstance *v1.Jaeger, esIndexPrefix string, daysRange []int) {
	for _, verifyDays := range daysRange {
		logrus.Infof("Scheduling index cleaner job for %d days", verifyDays)
		// update and trigger index cleaner job
		suite.turnOnEsIndexCleaner(jaegerInstance, verifyDays)

		// get services and spans indices
		servicesIndices, spanIndices := GetJaegerIndices(suite.esNamespace)

		// set valid index start date
		indexDateReference := time.Now().AddDate(0, 0, -1*verifyDays)
		// set hours, minutes, seconds, etc.. to 0
		indexDateReference = time.Date(indexDateReference.Year(), indexDateReference.Month(), indexDateReference.Day(), 0, 0, 0, 0, indexDateReference.Location())
		logrus.Infof("indices status on es node={numberOfDays:%d, services:%d, spans:%d}", verifyDays, len(servicesIndices), len(spanIndices))
		suite.assertIndex(esIndexPrefix, servicesIndices, indexDateReference, verifyDays)
		suite.assertIndex(esIndexPrefix, spanIndices, indexDateReference, verifyDays)
	}
}

func (suite *ElasticSearchIndexTestSuite) turnOnEsIndexCleaner(jaegerInstance *v1.Jaeger, indexCleanerNumOfDays int) {
	// enable index cleaner job
	suite.updateJaegerCR(jaegerInstance, indexCleanerNumOfDays, true)

	// wait till the cron job created
	err := WaitForCronJob(t, fw.KubeClient, namespace, fmt.Sprintf("%s-es-index-cleaner", jaegerInstance.Name), retryInterval, timeout+1*time.Minute)
	require.NoError(t, err, "Error waiting for Cron Job")

	// wait for the first successful cron job pod
	err = WaitForJobOfAnOwner(t, fw.KubeClient, namespace, fmt.Sprintf("%s-es-index-cleaner", jaegerInstance.Name), retryInterval, timeout)
	require.NoError(t, err, "Error waiting for Cron Job")

	// disable index cleaner job
	suite.updateJaegerCR(jaegerInstance, indexCleanerNumOfDays, false)

	// seeing inconsistency in minikube when immediately disabling and enabling index cleaner job
	// as a result index clear job is not triggering, so sleep for a while
	time.Sleep(time.Second * 5)

	// delete completed job pods
	err = fw.KubeClient.CoreV1().Pods(namespace).DeleteCollection(
		context.Background(),
		metav1.DeleteOptions{},
		metav1.ListOptions{LabelSelector: "app.kubernetes.io/component=cronjob-es-index-cleaner"})
	require.NoError(t, err, "Error on delete index cleaner pods")
}

// function to update jaeger CR
func (suite *ElasticSearchIndexTestSuite) updateJaegerCR(jaegerInstance *v1.Jaeger, indexCleanerNumOfDays int, indexCleanerEnabled bool) {
	// get existing values
	key := types.NamespacedName{Name: jaegerInstance.Name, Namespace: jaegerInstance.GetNamespace()}
	err := fw.Client.Get(context.Background(), key, jaegerInstance)
	require.NoError(t, err)

	// update values
	jaegerInstance.Spec.Storage.EsIndexCleaner.Enabled = &indexCleanerEnabled
	jaegerInstance.Spec.Storage.EsIndexCleaner.NumberOfDays = &indexCleanerNumOfDays
	err = fw.Client.Update(context.Background(), jaegerInstance)
	require.NoError(t, err)
}
