// +build elasticsearchrollover

package e2e

import (
	"context"
	"fmt"
	"regexp"

	"testing"
	"time"

	framework "github.com/operator-framework/operator-sdk/pkg/test"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
	"k8s.io/apimachinery/pkg/types"

	v1 "github.com/jaegertracing/jaeger-operator/pkg/apis/jaegertracing/v1"
)

type ElasticSearchRolloverTestSuite struct {
	suite.Suite
	esNamespace string // default storage namespace location
}

func TestElasticSearchRolloverSuite(t *testing.T) {
	indexSuite := new(ElasticSearchRolloverTestSuite)
	suite.Run(t, indexSuite)
}

func (suite *ElasticSearchRolloverTestSuite) SetupSuite() {
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

func (suite *ElasticSearchRolloverTestSuite) TearDownSuite() {
	// if !skipESExternal {
	// 	DeleteEsIndices(suite.esNamespace)
	// }

	handleSuiteTearDown()
}

func (suite *ElasticSearchRolloverTestSuite) SetupTest() {
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

func (suite *ElasticSearchRolloverTestSuite) AfterTest(suiteName, testName string) {
	handleTestFailure()
}

func (suite *ElasticSearchRolloverTestSuite) TestRolloverESSmoke() {
	jaegerInstanceName := "test-es-rollover"

	// Get the Jaeger CR to create the service
	jaegerInstance := GetJaegerSelfProvSimpleProdCR(jaegerInstanceName, namespace, 1)
	defer undeployJaegerInstance(jaegerInstance)

	// If there is an external ES deployment use it instead of creating a self provisioned one
	if !skipESExternal {
		if isOpenShift(t) {
			esServerUrls = "http://elasticsearch." + storageNamespace + ".svc.cluster.local:9200"
		}
	}

	// Configure the ES Rollover feature
	jaegerInstance.Spec.Storage = v1.JaegerStorageSpec{
		Type: v1.JaegerESStorage,
		Options: v1.NewOptions(map[string]interface{}{
			"es.server-urls": esServerUrls,
			"es.use-aliases": true,
		}),
		EsRollover: v1.JaegerEsRolloverSpec{
			Schedule:   "* * * * *",
			Conditions: "{\"max_age\": \"1d\"}",
			ReadTTL:    "10h",
		},
	}

	createESSelfProvDeployment(jaegerInstance, jaegerInstanceName, namespace)

	suite.waitRolloverDeployment(jaegerInstanceName)
}

func (suite *ElasticSearchRolloverTestSuite) TestRolloverESIndex() {
	jaegerInstanceName := "test-es-rollover"
	jaegerInstance := suite.createJaegerDeployment(jaegerInstanceName)
	defer undeployJaegerInstance(jaegerInstance)

	// Enable Rollover
	suite.updateJaegerCR(jaegerInstance, "* * * * *", "{\"max_docs\": \"100\", \"max_age\": \"1d\"}", "1m")
	suite.waitRolloverDeployment(jaegerInstanceName)

	generatedSpans := 2
	GenerateSpansHistory(namespace, jaegerInstanceName, "span-rollover-test", ElasticSearchIndexDateLayout, generatedSpans)

	// The generation of the indices is not instantaneus. So, we wait some time until they are generated
	time.Sleep(time.Second * 5)
	serviceIndices, spansIndices := GetJaegerIndices(suite.esNamespace)

	// Find the the new created indices with the expected names
	reRolloutPrefix := regexp.MustCompile(`jaeger-(span|service)-\d{6}`)

	for _, esIndex := range spansIndices {
		require.Regexp(t, reRolloutPrefix, esIndex.IndexName)
	}

	for _, esIndex := range serviceIndices {
		require.Regexp(t, reRolloutPrefix, esIndex.IndexName)
	}
}

func (suite *ElasticSearchRolloverTestSuite) TestRolloverESEnable() {
	jaegerInstanceName := "test-es-rollover"
	jaegerInstance := suite.createJaegerDeployment(jaegerInstanceName)
	defer undeployJaegerInstance(jaegerInstance)

	// We generate some spans before enabling the Rollover feature
	generatedSpans := 2
	GenerateSpansHistory(namespace, jaegerInstanceName, "span-rollover-test", ElasticSearchIndexDateLayout, generatedSpans)

	// The generation of the indices is not instantaneus. So, we wait some time until they are generated
	time.Sleep(time.Second * 5)
	serviceIndicesBefore, spansIndicesBefore := GetJaegerIndices(suite.esNamespace)

	// Enable Rollover
	suite.updateJaegerCR(jaegerInstance, "* * * * *", "{\"max_docs\": \"100\", \"max_age\": \"1d\"}", "1m")
	suite.waitRolloverDeployment(jaegerInstanceName)

	// We add new spans after enabling the rollout feature
	GenerateSpansHistory(namespace, jaegerInstanceName, "span-rollover-test", ElasticSearchIndexDateLayout, generatedSpans)
	time.Sleep(time.Second * 5)
	serviceIndicesAfter, spansIndicesAfter := GetJaegerIndices(suite.esNamespace)

	// The old indices were not removed
	assert.Greater(t, len(serviceIndicesAfter), len(serviceIndicesBefore))
	assert.Greater(t, len(spansIndicesAfter), len(spansIndicesBefore))

	// Check the old indices were not modified
	for _, index := range serviceIndicesBefore {
		foundIndex, err := FindIndex(serviceIndicesAfter, index.IndexName)
		require.NoError(t, err)
		require.Equal(t, foundIndex, index)
	}

	for _, index := range spansIndicesBefore {
		foundIndex, err := FindIndex(spansIndicesAfter, index.IndexName)
		require.NoError(t, err)
		require.Equal(t, foundIndex, index)
	}

	// We remove all the indexes
	DeleteEsIndices(suite.esNamespace)
	GenerateSpansHistory(namespace, jaegerInstanceName, "span-rollover-test", ElasticSearchIndexDateLayout, generatedSpans)

	// After starting from scratch, we just have 1 index for spans
	time.Sleep(time.Second * 5)
	serviceIndices, spansIndices := GetJaegerIndices(suite.esNamespace)
	assert.Len(t, serviceIndices, 0)
	assert.Len(t, spansIndices, 1)
}

func (suite *ElasticSearchRolloverTestSuite) waitRolloverDeployment(jaegerInstanceName string) {
	// Wait until the cronjob is created
	err := WaitForCronJob(t, fw.KubeClient, namespace, fmt.Sprintf("%s-es-rollover", jaegerInstanceName), retryInterval, timeout+1*time.Minute)
	require.NoError(t, err, "Error waiting for Cron Job")

	// Wait until the cronjob is executed properly at least one time
	err = WaitForJobOfAnOwner(t, fw.KubeClient, namespace, fmt.Sprintf("%s-es-rollover", jaegerInstanceName), retryInterval, timeout)
	require.NoError(t, err, "Error waiting for Cron Job")
}

// Create and deploy a Jaeger deployment without the Rollver ES feature enabled
func (suite *ElasticSearchRolloverTestSuite) createJaegerDeployment(name string) *v1.Jaeger {
	// Get the Jaeger CR to create the service
	jaegerInstance := GetJaegerSelfProvSimpleProdCR(name, namespace, 1)

	// If there is an external ES deployment use it instead of creating a self provisioned one
	if !skipESExternal {
		if isOpenShift(t) {
			esServerUrls = "http://elasticsearch." + storageNamespace + ".svc.cluster.local:9200"
		}
	}

	// Configure the ES Rollover feature
	jaegerInstance.Spec.Storage = v1.JaegerStorageSpec{
		Type: v1.JaegerESStorage,
		Options: v1.NewOptions(map[string]interface{}{
			"es.server-urls": esServerUrls,
		}),
	}

	createESSelfProvDeployment(jaegerInstance, name, namespace)
	return jaegerInstance
}

// function to update jaeger CR
func (suite *ElasticSearchRolloverTestSuite) updateJaegerCR(jaegerInstance *v1.Jaeger, schedule string, conditions string, readTTL string) {
	// get existing values
	key := types.NamespacedName{Name: jaegerInstance.Name, Namespace: jaegerInstance.GetNamespace()}
	err := fw.Client.Get(context.Background(), key, jaegerInstance)
	require.NoError(t, err)

	// update values
	options := jaegerInstance.Spec.Storage.Options.GenericMap()
	options["es.use-aliases"] = true // Enable Rollover feature

	jaegerInstance.Spec.Storage.Options = v1.NewOptions(options)

	jaegerInstance.Spec.Storage.EsRollover.Schedule = schedule
	jaegerInstance.Spec.Storage.EsRollover.Conditions = conditions
	jaegerInstance.Spec.Storage.EsRollover.ReadTTL = readTTL

	err = fw.Client.Update(context.Background(), jaegerInstance)
	require.NoError(t, err)
}
