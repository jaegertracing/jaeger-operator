// +build tolerations

package e2e

import (
	goctx "context"
	"fmt"
	"testing"

	framework "github.com/operator-framework/operator-sdk/pkg/test"
	"github.com/operator-framework/operator-sdk/pkg/test/e2eutil"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"

	v1 "k8s.io/api/core/v1"
)

type TolerationsTestSuite struct {
	suite.Suite
}

func (suite *TolerationsTestSuite) SetupSuite() {
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

func (suite *TolerationsTestSuite) TearDownSuite() {
	handleSuiteTearDown()
}

func TestTolerationsTestSuite(t *testing.T) {
	suite.Run(t, new(TolerationsTestSuite))
}

func (suite *TolerationsTestSuite) SetupTest() {
	t = suite.T()
}

func (suite *TolerationsTestSuite) AfterTest(suiteName, testName string) {
	handleTestFailure()
}

func (suite *TolerationsTestSuite) TestAllInOneTolerations() {
	jaegerInstanceName := "all-in-one-tolerations"

	jaegerCR := GetJaegerAllInOneCR(jaegerInstanceName, namespace)

	// get tolerations sample sets
	tolerationsAllInOne := suite.getTolerations("all-in-one")

	// update tolerations to jaeger cr
	jaegerCR.Spec.AllInOne.Tolerations = tolerationsAllInOne

	logrus.Infof("Creating jaeger services for tolerations test. jaeger-cr:%s, namespace:%s", jaegerInstanceName, namespace)
	err := fw.Client.Create(goctx.TODO(), jaegerCR, &framework.CleanupOptions{TestContext: ctx, Timeout: timeout, RetryInterval: retryInterval})
	require.NoError(t, err, "Error deploying Jaeger-all-in-one")
	defer undeployJaegerInstance(jaegerCR)

	err = e2eutil.WaitForDeployment(t, fw.KubeClient, namespace, jaegerInstanceName, 1, retryInterval, timeout)
	require.NoError(t, err, "Error waiting for all-in-one deployment")

	AllInOneSmokeTest(jaegerInstanceName)

	// verify tolerations
	instanceLabel := fmt.Sprintf("app=jaeger,app.kubernetes.io/component=all-in-one,app.kubernetes.io/instance=%s", jaegerInstanceName)
	allInOneDeployments := getDeployments(namespace, instanceLabel)
	require.Equal(t, 1, len(allInOneDeployments), "AllInOne deployments count not matching")
	allInOneDeployment := allInOneDeployments[0]
	require.Equal(t, int32(1), allInOneDeployment.Status.ReadyReplicas, "AllInOne deployment replicas count not matching")
	suite.verifyTolerations(allInOneDeployment.Name, tolerationsAllInOne, allInOneDeployment.Spec.Template.Spec.Tolerations)
}

func (suite *TolerationsTestSuite) TestElasticsearchProdTolerations() {
	jaegerInstanceName := "simple-prod-tolerations"
	collectorReplicasCount := int32(1)
	queryReplicasCount := int32(1)
	esNodeCount := int32(1)

	jaegerCR := GetJaegerSelfProvSimpleProdCR(jaegerInstanceName, namespace, esNodeCount)

	// update replicas count
	jaegerCR.Spec.Collector.Replicas = &collectorReplicasCount
	jaegerCR.Spec.Query.Replicas = &queryReplicasCount

	// get tolerations sample sets
	tolerationsCollector := suite.getTolerations("collector")
	tolerationsQuery := suite.getTolerations("query")
	tolerationsES := suite.getTolerations("es")

	// update tolerations to jaeger cr
	jaegerCR.Spec.Collector.Tolerations = tolerationsCollector
	jaegerCR.Spec.Query.Tolerations = tolerationsQuery
	jaegerCR.Spec.Storage.Elasticsearch.Tolerations = tolerationsES

	logrus.Infof("Creating jaeger services for tolerations test. jaeger-cr:%s, namespace:%s", jaegerInstanceName, namespace)
	createESSelfProvDeployment(jaegerCR, jaegerInstanceName, namespace)
	defer undeployJaegerInstance(jaegerCR)

	ProductionSmokeTest(jaegerInstanceName)

	// verify tolerations

	collectorDeployments := getDeployments(namespace, fmt.Sprintf("app=jaeger,app.kubernetes.io/component=collector,app.kubernetes.io/instance=%s", jaegerInstanceName))
	require.Equal(t, 1, len(collectorDeployments), "Collector deployments count not matching")
	collectorDeployment := collectorDeployments[0]
	require.Equal(t, collectorReplicasCount, collectorDeployment.Status.ReadyReplicas, "Collector deployment replicas count not matching")
	suite.verifyTolerations(collectorDeployment.Name, tolerationsCollector, collectorDeployment.Spec.Template.Spec.Tolerations)

	queryDeployments := getDeployments(namespace, fmt.Sprintf("app=jaeger,app.kubernetes.io/component=query,app.kubernetes.io/instance=%s", jaegerInstanceName))
	require.Equal(t, 1, len(queryDeployments), "Query deployments count not matching")
	queryDeployment := queryDeployments[0]
	require.Equal(t, queryReplicasCount, queryDeployment.Status.ReadyReplicas, "Query deployment replicas count not matching")
	suite.verifyTolerations(queryDeployment.Name, tolerationsQuery, queryDeployment.Spec.Template.Spec.Tolerations)

	esDeployments := getDeployments(namespace, "component=elasticsearch")
	require.Equal(t, esNodeCount, int32(len(esDeployments)), "Elasticsearch deployments count not matching")
	for index := 0; index < len(esDeployments); index++ {
		esDeployment := esDeployments[index]
		require.Equal(t, int32(1), esDeployment.Status.ReadyReplicas, "Elasticsearch deployment replicas count not matching")
		suite.verifyTolerations(esDeployment.Name, tolerationsES, esDeployment.Spec.Template.Spec.Tolerations)
	}
}

func (suite *TolerationsTestSuite) getTolerations(prefix string) []v1.Toleration {
	tolerations := []v1.Toleration{
		{Key: fmt.Sprintf("%s_equal_key1", prefix), Operator: v1.TolerationOpEqual, Value: "value1", Effect: v1.TaintEffectNoExecute},
		{Key: fmt.Sprintf("%s_equal_key2", prefix), Operator: v1.TolerationOpEqual, Value: "value2", Effect: v1.TaintEffectNoSchedule},
		{Key: fmt.Sprintf("%s_equal_key3", prefix), Operator: v1.TolerationOpEqual, Value: "value3", Effect: v1.TaintEffectPreferNoSchedule},

		{Key: fmt.Sprintf("%s_exists_key1", prefix), Operator: v1.TolerationOpExists, Effect: v1.TaintEffectNoExecute},
		{Key: fmt.Sprintf("%s_exists_key2", prefix), Operator: v1.TolerationOpExists, Effect: v1.TaintEffectNoSchedule},
		{Key: fmt.Sprintf("%s_exists_key3", prefix), Operator: v1.TolerationOpExists, Effect: v1.TaintEffectPreferNoSchedule},
	}
	return tolerations
}

func (suite *TolerationsTestSuite) verifyTolerations(deploymentName string, expectedList []v1.Toleration, actualList []v1.Toleration) {
	logrus.Infof("tolerations, deploymentName:%s, expectedList:[%+v]", deploymentName, expectedList)
	logrus.Infof("tolerations, deploymentName:%s, actualList:[%+v]", deploymentName, actualList)
	require.True(t, len(expectedList) <= len(actualList), "actual tolerations count is less than the expected")
	for expectedIndex := 0; expectedIndex < len(expectedList); expectedIndex++ {
		expected := expectedList[expectedIndex]
		found := false
		for actualIndex := 0; actualIndex < len(actualList); actualIndex++ {
			actual := actualList[actualIndex]
			if expected.Key == actual.Key &&
				expected.Operator == actual.Operator &&
				expected.Value == actual.Value &&
				expected.Effect == actual.Effect {
				found = true
				break
			}
		}
		require.True(t, found, "toleration not found on the actual list, %+v", expected)
	}
}
