// +build elasticsearch

package e2e

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/opentracing/opentracing-go"
	framework "github.com/operator-framework/operator-sdk/pkg/test"
	"github.com/operator-framework/operator-sdk/pkg/test/e2eutil"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/tools/portforward"

	v1 "github.com/jaegertracing/jaeger-operator/pkg/apis/jaegertracing/v1"
)

type ElasticSearchTestSuite struct {
	suite.Suite
}

// EsIndex struct to map indices data from es rest api response
// es api: /_cat/indices?format=json
type EsIndex struct {
	UUID        string `json:"uuid"`
	Status      string `json:"status"`
	Index       string `json:"index"`
	Health      string `json:"health"`
	DocsCount   string `json:"docs.count"`
	DocsDeleted string `json:"docs.deleted"`
	StoreSize   string `json:"store.size"`
}

var (
	esIndexCleanerEnabled = false            // global variable used to enable/disable index cleaner (pass-by-reference in jaeger CR object)
	numberOfDays          = 0                // index cleaner number of days (pass-by-reference in jaeger CR object)
	esNamespace           = storageNamespace // default storage namespace location
	esUrl                 string             // es node url
)

func (suite *ElasticSearchTestSuite) SetupSuite() {
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

	if isOpenShift(t) {
		esServerUrls = "http://elasticsearch." + storageNamespace + ".svc.cluster.local:9200"
	}
}

func (suite *ElasticSearchTestSuite) TearDownSuite() {
	handleSuiteTearDown()
}

func TestElasticSearchSuite(t *testing.T) {
	suite.Run(t, new(ElasticSearchTestSuite))
}

func (suite *ElasticSearchTestSuite) SetupTest() {
	t = suite.T()
	// delete indices from external elasticsearch
	if !skipESExternal {
		deleteEsIndices()
	}
}

func (suite *ElasticSearchTestSuite) AfterTest(suiteName, testName string) {
	handleTestFailure()
}

func (suite *ElasticSearchTestSuite) TestSparkDependenciesES() {
	if skipESExternal {
		t.Skip("This test requires an insecure ElasticSearch instance")
	}
	storage := v1.JaegerStorageSpec{
		Type: v1.JaegerESStorage,
		Options: v1.NewOptions(map[string]interface{}{
			"es.server-urls": esServerUrls,
		}),
	}
	err := sparkTest(t, framework.Global, ctx, storage)
	require.NoError(t, err, "SparkTest failed")
}

func (suite *ElasticSearchTestSuite) TestSimpleProd() {
	if skipESExternal {
		t.Skip("This case is covered by the self_provisioned_elasticsearch_test")
	}
	err := WaitForStatefulset(t, fw.KubeClient, storageNamespace, string(v1.JaegerESStorage), retryInterval, timeout)
	require.NoError(t, err, "Error waiting for elasticsearch")

	// create jaeger custom resource
	name := "simple-prod"
	exampleJaeger := getJaegerSimpleProdWithServerUrls(name)
	err = fw.Client.Create(context.TODO(), exampleJaeger, &framework.CleanupOptions{TestContext: ctx, Timeout: timeout, RetryInterval: retryInterval})
	require.NoError(t, err, "Error deploying example Jaeger")
	defer undeployJaegerInstance(exampleJaeger)

	err = e2eutil.WaitForDeployment(t, fw.KubeClient, namespace, name+"-collector", 1, retryInterval, timeout)
	require.NoError(t, err, "Error waiting for collector deployment")

	err = e2eutil.WaitForDeployment(t, fw.KubeClient, namespace, name+"-query", 1, retryInterval, timeout)
	require.NoError(t, err, "Error waiting for query deployment")

	ProductionSmokeTest(name)

	// Make sure we were using the correct collector image
	verifyCollectorImage(name, namespace, specifyOtelImages)
}

// executes es index cleaner tests with custom index prefix
func (suite *ElasticSearchTestSuite) TestEsIndexCleanerWithIndexPrefix() {
	suite.runIndexCleaner("my-custom_prefix", []int{3, 1, 0})
}

// executes es index cleaner with default index prefix
func (suite *ElasticSearchTestSuite) TestEsIndexCleaner() {
	suite.runIndexCleaner("", []int{45, 30, 7, 1, 0})
}

// executes index cleaner tests
func (suite *ElasticSearchTestSuite) runIndexCleaner(esIndexPrefix string, daysRange []int) {
	logrus.Infof("index cleaner test started. daysRange=%v, prefix=%s", daysRange, esIndexPrefix)
	jaegerInstanceName := "test-es-index-cleaner"
	if esIndexPrefix != "" {
		jaegerInstanceName = "test-es-index-cleaner-with-prefix"
	}
	// get jaeger CR to create jaeger services
	jaegerInstance := getJaegerSelfProvSimpleProd(jaegerInstanceName, namespace, 1)

	// storage namespace
	esNamespace = namespace

	// update es node namespace and es node url into jaeger CR for external es deployment
	if !skipESExternal {
		esNamespace = storageNamespace
		jaegerInstance.Spec.Storage = v1.JaegerStorageSpec{
			Type: v1.JaegerESStorage,
			Options: v1.NewOptions(map[string]interface{}{
				"es.server-urls": esServerUrls,
			}),
		}
	}

	// update jaeger CR with index cleaner specifications
	indexHistoryDays := 45 // maximum number of days to generate spans and services
	numberOfDays = indexHistoryDays
	// initially disable es index cleaner job
	esIndexCleanerEnabled = false
	jaegerInstance.Spec.Storage.EsIndexCleaner.Enabled = &esIndexCleanerEnabled
	jaegerInstance.Spec.Storage.EsIndexCleaner.Schedule = "*/1 * * * *"
	jaegerInstance.Spec.Storage.EsIndexCleaner.NumberOfDays = &numberOfDays
	// update es.index-prefix, if supplied
	if esIndexPrefix != "" {
		if jaegerInstance.Spec.Storage.Options.Map() == nil {
			jaegerInstance.Spec.Storage.Options = v1.NewOptions(map[string]interface{}{})
		}
		jaegerInstance.Spec.Storage.Options.Map()["es.index-prefix"] = esIndexPrefix
	}

	// update otel specific change
	if specifyOtelImages {
		logrus.Infof("Using OTEL collector for %s", jaegerInstanceName)
		jaegerInstance.Spec.Collector.Image = otelCollectorImage
		jaegerInstance.Spec.Collector.Config = v1.NewFreeForm(getOtelConfigForHealthCheckPort("14269"))
	}

	// deploy jaeger services
	logrus.Infof("Creating jaeger services for es index cleaner test: %s", jaegerInstanceName)
	createESSelfProvDeployment(jaegerInstance, jaegerInstanceName, namespace)
	defer undeployJaegerInstance(jaegerInstance)

	// generate spans and service for last 45 days
	currentDate := time.Now()
	indexDateLayout := "2006-01-02"
	// enable port forward for collector port
	logrus.Info("Enabling collector port forward")
	fwdPortColl, closeChanColl := CreatePortForward(namespace, jaegerInstanceName+"-collector", "collector", []string{fmt.Sprintf(":%d", jaegerCollectorPort)}, fw.KubeConfig)
	defer fwdPortColl.Close()
	defer close(closeChanColl)
	// get localhost collector port
	colPorts, err := fwdPortColl.GetPorts()
	require.NoError(t, err)
	localPortColl := colPorts[0].Local

	logrus.Infof("Generating spans and services for the last %d days", indexHistoryDays)
	for day := 0; day < indexHistoryDays; day++ {
		spanDate := currentDate.AddDate(0, 0, -1*day)
		stringDate := spanDate.Format(indexDateLayout)
		// get tracing client
		serviceName := fmt.Sprintf("%s_%s", jaegerInstanceName, stringDate)
		tracer, closer, err := getTracerClientWithCollectorEndpoint(serviceName, fmt.Sprintf("http://localhost:%d/api/traces", localPortColl))
		require.NoError(t, err)
		// generate span
		tracer.StartSpan("span-index-cleaner", opentracing.StartTime(spanDate)).
			SetTag("jaeger-instance", jaegerInstanceName).
			SetTag("test-case", t.Name()).
			SetTag("string-date", stringDate).
			FinishWithOptions(opentracing.FinishOptions{FinishTime: spanDate.Add(time.Second)})
		closer.Close()
	}

	// esIndexData struct is used to keep index data in simple format
	// will be useful for the validations
	type esIndexData struct {
		IndexName string    // original index name
		Type      string    // index type. span or service?
		Prefix    string    // prefix of the index
		Date      time.Time // index day/date
	}

	// function to get indices
	// returns in order: serviceIndices, spansIndices
	getIndices := func() ([]esIndexData, []esIndexData) {
		// get indices from es node
		esIndices, err := getEsIndices()
		require.NoError(t, err)
		fmt.Println("indices found on rest api response:", len(esIndices))

		servicesIndices := make([]esIndexData, 0)
		spansIndices := make([]esIndexData, 0)

		// parse date, prefix, type from index
		re := regexp.MustCompile(`\d{4}-\d{2}-\d{2}`)
		for _, esIndex := range esIndices {
			indexName := esIndex.Index
			dateString := re.FindString(indexName)
			if dateString == "" { // assume this index not belongs to jaeger
				continue
			}

			indexName = strings.Replace(indexName, dateString, "", 1)

			indexDate, err := time.Parse(indexDateLayout, dateString)
			require.NoError(t, err)

			esData := esIndexData{
				IndexName: esIndex.Index,
				Date:      indexDate,
			}

			// reference
			// https://github.com/jaegertracing/jaeger/blob/6c2be456ca41cdb98ac4b81cb8d9a9a9044463cd/plugin/storage/es/spanstore/reader.go#L40
			if strings.Contains(indexName, "jaeger-span-") {
				esData.Type = "span"
				prefix := strings.Replace(indexName, "jaeger-span-", "", 1)
				if len(prefix) > 0 {
					esData.Prefix = prefix[:len(prefix)-1] // removes "-" at end
				}
				spansIndices = append(spansIndices, esData)
			} else if strings.Contains(indexName, "jaeger-service-") {
				esData.Type = "service"
				prefix := strings.Replace(indexName, "jaeger-service-", "", 1)
				if len(prefix) > 0 {
					esData.Prefix = prefix[:len(prefix)-1] // removes "-" at end
				}
				servicesIndices = append(servicesIndices, esData)
			}
		}
		return servicesIndices, spansIndices
	}

	// function to validate indices
	assertIndex := func(indices []esIndexData, verifyDateAfter time.Time, count int) {
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

	// trigger the index cleaner for multiple day range and verify indices
	for _, verifyDays := range daysRange {
		logrus.Infof("Scheduling index cleaner job for %d days", verifyDays)
		// update and trigger index cleaner job
		turnOnEsIndexCleaner(jaegerInstance, verifyDays)

		// get servies and spans indices
		servicesIndices, spanIndices := getIndices()
		// valid index start date
		indexDateReference := time.Now().AddDate(0, 0, -1*verifyDays)
		// set hours, minutes, seconds, etc.. to 0
		indexDateReference = time.Date(indexDateReference.Year(), indexDateReference.Month(), indexDateReference.Day(), 0, 0, 0, 0, indexDateReference.Location())
		logrus.Infof("indices found={numberOfDays:%d, services:%d, spans:%d}", verifyDays, len(servicesIndices), len(spanIndices))
		assertIndex(servicesIndices, indexDateReference, verifyDays)
		assertIndex(spanIndices, indexDateReference, verifyDays)
	}
}

func getJaegerSimpleProdWithServerUrls(name string) *v1.Jaeger {
	ingressEnabled := true
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
			Ingress: v1.JaegerIngressSpec{
				Enabled:  &ingressEnabled,
				Security: v1.IngressSecurityNoneExplicit,
			},
			Strategy: v1.DeploymentStrategyProduction,
			Storage: v1.JaegerStorageSpec{
				Type: v1.JaegerESStorage,
				Options: v1.NewOptions(map[string]interface{}{
					"es.server-urls": esServerUrls,
				}),
			},
		},
	}

	if specifyOtelImages {
		logrus.Infof("Using OTEL collector for %s", name)
		exampleJaeger.Spec.Collector.Image = otelCollectorImage
		exampleJaeger.Spec.Collector.Config = v1.NewFreeForm(getOtelConfigForHealthCheckPort("14269"))
	}

	return exampleJaeger
}

// return indices from es node
func getEsIndices() ([]EsIndex, error) {
	bodyBytes, err := executeEsRequest(http.MethodGet, "/_cat/indices?format=json")
	require.NoError(t, err)

	// convert json data to struct format
	esIndices := make([]EsIndex, 0)
	err = json.Unmarshal(bodyBytes, &esIndices)
	require.NoError(t, err)

	return esIndices, nil
}

// deletes all the indices on es node
func deleteEsIndices() {
	logrus.Info("deleting all es node indices")
	_, err := executeEsRequest(http.MethodDelete, "/_all?format=json")
	require.NoError(t, err)
}

// executes rest api request on es node
func executeEsRequest(httpMethod, api string) ([]byte, error) {
	// enable port forward
	fwdPortES, closeChanES, esPort := createEsPortForward(esNamespace)
	defer fwdPortES.Close()
	defer close(closeChanES)

	// update es node url
	urlScheme := "http"
	if skipESExternal {
		urlScheme = "https"
	}
	esUrl = fmt.Sprintf("%s://localhost:%s%s", urlScheme, esPort, api)

	// create rest client to access es node rest API
	transport := &http.Transport{}
	client := http.Client{Transport: transport}

	// update certificates, if the es node provided by jaeger-operator
	if skipESExternal {
		esSecret, err := fw.KubeClient.CoreV1().Secrets(namespace).Get(context.Background(), "elasticsearch", metav1.GetOptions{})
		require.NoError(t, err)
		pool := x509.NewCertPool()
		pool.AppendCertsFromPEM(esSecret.Data["admin-ca"])

		clientCert, err := tls.X509KeyPair(esSecret.Data["admin-cert"], esSecret.Data["admin-key"])
		require.NoError(t, err)

		transport.TLSClientConfig = &tls.Config{
			RootCAs:      pool,
			Certificates: []tls.Certificate{clientCert},
		}
	}

	req, err := http.NewRequest(httpMethod, esUrl, nil)
	require.NoError(t, err)

	resp, err := client.Do(req)
	require.NoError(t, err)
	defer resp.Body.Close()

	require.EqualValues(t, 200, resp.StatusCode)

	return ioutil.ReadAll(resp.Body)
}

func createEsPortForward(esNamespace string) (portForwES *portforward.PortForwarder, closeChanES chan struct{}, esPort string) {
	portForwES, closeChanES = CreatePortForward(esNamespace, string(v1.JaegerESStorage), string(v1.JaegerESStorage), []string{"0:9200"}, fw.KubeConfig)
	forwardedPorts, err := portForwES.GetPorts()
	require.NoError(t, err)
	return portForwES, closeChanES, strconv.Itoa(int(forwardedPorts[0].Local))
}

func turnOnEsIndexCleaner(jaegerInstance *v1.Jaeger, days int) {
	key := types.NamespacedName{Name: jaegerInstance.Name, Namespace: jaegerInstance.GetNamespace()}
	err := fw.Client.Get(context.Background(), key, jaegerInstance)
	require.NoError(t, err)

	// update values
	esIndexCleanerEnabled = true
	numberOfDays = days
	err = fw.Client.Update(context.Background(), jaegerInstance)
	require.NoError(t, err)

	err = WaitForCronJob(t, fw.KubeClient, namespace, fmt.Sprintf("%s-es-index-cleaner", jaegerInstance.Name), retryInterval, timeout+1*time.Minute)
	require.NoError(t, err, "Error waiting for Cron Job")

	err = WaitForJobOfAnOwner(t, fw.KubeClient, namespace, fmt.Sprintf("%s-es-index-cleaner", jaegerInstance.Name), retryInterval, timeout)
	require.NoError(t, err, "Error waiting for Cron Job")

	// disable index cleaner job
	esIndexCleanerEnabled = false
	err = fw.Client.Update(context.Background(), jaegerInstance)
	require.NoError(t, err)
}
