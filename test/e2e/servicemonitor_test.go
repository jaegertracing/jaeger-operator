// +build servicemonitor

package e2e

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"testing"
	"time"

	framework "github.com/operator-framework/operator-sdk/pkg/test"
	"github.com/operator-framework/operator-sdk/pkg/test/e2eutil"
	monitoringv1 "github.com/prometheus-operator/prometheus-operator/pkg/apis/monitoring/v1"
	log "github.com/sirupsen/logrus"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
	corev1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/wait"

	v1 "github.com/jaegertracing/jaeger-operator/pkg/apis/jaegertracing/v1"
)

type ServiceMonitorTestSuite struct {
	suite.Suite
}

func (suite *ServiceMonitorTestSuite) SetupSuite() {
	t = suite.T()
	require.NoError(t, framework.AddToFrameworkScheme(monitoringv1.AddToScheme, &monitoringv1.PrometheusList{
		TypeMeta: metav1.TypeMeta{
			Kind:       "Prometheus",
			APIVersion: "monitoring.coreos.com/v1",
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
	namespace = ctx.GetID()
	require.NotNil(t, namespace, "GetID failed")

	addToFrameworkSchemeForSmokeTests(t)

	if isOpenShift(t) {
		esServerUrls = "http://elasticsearch." + storageNamespace + ".svc.cluster.local:9200"
	}
}

func (suite *ServiceMonitorTestSuite) TearDownSuite() {
	handleSuiteTearDown()
}

func TestServiceMonitorSuite(t *testing.T) {
	suite.Run(t, new(ServiceMonitorTestSuite))
}

func (suite *ServiceMonitorTestSuite) SetupTest() {
	t = suite.T()
}

func (suite *ServiceMonitorTestSuite) AfterTest(suiteName, testName string) {
	handleTestFailure()
}

func (suite *ServiceMonitorTestSuite) TestAllInOne() {
	prometheusInstanceName := "prometheus-allinone"
	jaegerInstanceName := "jaeger-allinone"

	jaeger := getJaegerAllInOneServiceMonitor(namespace, jaegerInstanceName)
	log.Infof("passing %v", jaeger)
	err := fw.Client.Create(context.TODO(), jaeger, &framework.CleanupOptions{TestContext: ctx, Timeout: timeout, RetryInterval: retryInterval})
	require.NoError(t, err, "Error deploying example Jaeger")
	defer undeployJaegerInstance(jaeger)

	err = e2eutil.WaitForDeployment(t, fw.KubeClient, namespace, jaegerInstanceName, 1, retryInterval, timeout)
	require.NoError(t, err, "Error waiting for deployment")

	// We deploy prometheus after jaeger, to avoid the 3 minute config reload interval after serviceMonitor is deployed
	deployPrometheus(t, namespace, prometheusInstanceName)
	err = WaitForStatefulset(t, fw.KubeClient, namespace, fmt.Sprintf("prometheus-%s", prometheusInstanceName), retryInterval, timeout)
	require.NoError(t, err, "Error waiting for prometheus")

	AllInOneSmokeTest(jaegerInstanceName)

	testPrometheusMetricCollector(t, prometheusInstanceName, jaegerInstanceName, "collector-admin")
}

func (suite *ServiceMonitorTestSuite) TestProduction() {
	waitForElasticSearch()

	prometheusInstanceName := "prometheus-prod"
	jaegerInstanceName := "jaeger-prod"

	jaeger := getJaegerProductionServiceMonitor(namespace, jaegerInstanceName)
	log.Infof("passing %v", jaeger)
	err := fw.Client.Create(context.TODO(), jaeger, &framework.CleanupOptions{TestContext: ctx, Timeout: timeout, RetryInterval: retryInterval})
	require.NoError(t, err, "Error deploying example Jaeger")
	defer undeployJaegerInstance(jaeger)

	err = e2eutil.WaitForDeployment(t, fw.KubeClient, namespace, jaegerInstanceName+"-collector", 1, retryInterval, timeout)
	require.NoError(t, err, "Error waiting for collector deployment")

	err = e2eutil.WaitForDeployment(t, fw.KubeClient, namespace, jaegerInstanceName+"-query", 1, retryInterval, timeout)
	require.NoError(t, err, "Error waiting for query deployment")

	// We deploy prometheus after jaeger, to avoid the 3 minute config reload interval after serviceMonitor is deployed
	deployPrometheus(t, namespace, prometheusInstanceName)
	err = WaitForStatefulset(t, fw.KubeClient, namespace, fmt.Sprintf("prometheus-%s", prometheusInstanceName), retryInterval, timeout)
	require.NoError(t, err, "Error waiting for prometheus")

	ProductionSmokeTest(jaegerInstanceName)

	testPrometheusMetricCollector(t, prometheusInstanceName, jaegerInstanceName, "collector-admin")
	testPrometheusMetricCollector(t, prometheusInstanceName, jaegerInstanceName, "query-admin")
}

func (suite *ServiceMonitorTestSuite) TestStreaming() {
	waitForElasticSearch()
	waitForKafkaInstance()

	prometheusInstanceName := "prometheus-streaming"
	jaegerInstanceName := "simple-streaming"
	j := getJaegerStreamingServiceMonitor(namespace, jaegerInstanceName)
	log.Infof("passing %v", j)
	err := fw.Client.Create(context.TODO(), j, &framework.CleanupOptions{TestContext: ctx, Timeout: timeout, RetryInterval: retryInterval})
	require.NoError(t, err, "Error deploying jaeger")
	defer undeployJaegerInstance(j)

	err = WaitForDeployment(t, fw.KubeClient, namespace, jaegerInstanceName+"-ingester", 1, retryInterval, timeout)
	require.NoError(t, err, "Error waiting for ingester deployment")

	err = WaitForDeployment(t, fw.KubeClient, namespace, jaegerInstanceName+"-collector", 1, retryInterval, timeout)
	require.NoError(t, err, "Error waiting for collector deployment")

	err = WaitForDeployment(t, fw.KubeClient, namespace, jaegerInstanceName+"-query", 1, retryInterval, timeout)
	require.NoError(t, err, "Error waiting for query deployment")

	// We deploy prometheus after jaeger, to avoid the 3 minute config reload interval after serviceMonitor is deployed
	deployPrometheus(t, namespace, prometheusInstanceName)
	err = WaitForStatefulset(t, fw.KubeClient, namespace, fmt.Sprintf("prometheus-%s", prometheusInstanceName), retryInterval, timeout)
	require.NoError(t, err, "Error waiting for prometheus")

	ProductionSmokeTest(jaegerInstanceName)

	testPrometheusMetricCollector(t, prometheusInstanceName, jaegerInstanceName, "collector-admin")
	testPrometheusMetricCollector(t, prometheusInstanceName, jaegerInstanceName, "query-admin")
	testPrometheusMetricCollector(t, prometheusInstanceName, jaegerInstanceName, "ingester-admin")
}

func testPrometheusMetricCollector(t *testing.T, prometheusInstanceName, jaegerInstanceName, service string) {
	serviceName := fmt.Sprintf("%s-%s", jaegerInstanceName, service)
	ports := []string{"9090"}
	portForward, closeChan := CreatePortForward(namespace, fmt.Sprintf("prometheus-%s", prometheusInstanceName), "prometheus", ports, fw.KubeConfig)
	defer portForward.Close()
	defer close(closeChan)
	forwardedPorts, err := portForward.GetPorts()
	require.NoError(t, err)
	queryPort := strconv.Itoa(int(forwardedPorts[0].Local))

	url := fmt.Sprintf("http://localhost:%s/api/v1/label/job/values", queryPort)
	c := http.Client{Timeout: 3 * time.Second}

	req, err := http.NewRequest(http.MethodGet, url, nil)
	require.NoError(t, err, "Failed to create httpRequest")

	// it takes some time for the prometheus to include our ServiceMonitor and get a result
	err = wait.Poll(retryInterval, timeout, func() (done bool, err error) {
		res, err := c.Do(req)
		if err != nil && strings.Contains(err.Error(), "Timeout exceeded") {
			log.Infof("Retrying request after error %v", err)
			return false, nil
		}
		require.NoError(t, err)

		if res.StatusCode != 200 {
			return false, fmt.Errorf("unexpected status code %d", res.StatusCode)
		}
		var jsonResult struct {
			Status string   `json:"status"`
			Data   []string `json:"data"`
		}

		json.NewDecoder(res.Body).Decode(&jsonResult)

		if contains(jsonResult.Data, serviceName) {
			return true, nil
		}
		return false, nil
	})
	require.NoError(t, err, "Prometheus was not able to collect metrics from serviceMonitor for service %s before timeout.", serviceName)
}

func contains(list []string, item string) bool {
	for _, v := range list {
		if item == v {
			return true
		}
	}
	return false
}

func getJaegerAllInOneServiceMonitor(namespace string, name string) *v1.Jaeger {
	serviceMonitorEnabled := true
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
			Strategy: v1.DeploymentStrategyAllInOne,
			ServiceMonitor: v1.JaegerServiceMonitorSpec{
				Enabled: &serviceMonitorEnabled,
			},
		},
	}
	return exampleJaeger
}

func getJaegerProductionServiceMonitor(namespace string, name string) *v1.Jaeger {
	serviceMonitorEnabled := true
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
			Strategy: v1.DeploymentStrategyProduction,
			ServiceMonitor: v1.JaegerServiceMonitorSpec{
				Enabled: &serviceMonitorEnabled,
			},
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

func getJaegerStreamingServiceMonitor(namespace string, name string) *v1.Jaeger {
	serviceMonitorEnabled := true
	kafkaClusterURL := fmt.Sprintf("my-cluster-kafka-brokers.%s:9092", kafkaNamespace)
	ingressEnabled := true
	collectorOptions := make(map[string]interface{})
	collectorOptions["kafka.producer.topic"] = "jaeger-spans"
	collectorOptions["kafka.producer.brokers"] = kafkaClusterURL

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
			Strategy: v1.DeploymentStrategyStreaming,
			Collector: v1.JaegerCollectorSpec{
				Options: v1.NewOptions(collectorOptions),
			},
			Ingester: v1.JaegerIngesterSpec{
				Options: v1.NewOptions(map[string]interface{}{
					"kafka.consumer.topic":   "jaeger-spans",
					"kafka.consumer.brokers": kafkaClusterURL,
				}),
			},
			Storage: v1.JaegerStorageSpec{
				Type: v1.JaegerESStorage,
				Options: v1.NewOptions(map[string]interface{}{
					"es.server-urls": esServerUrls,
				}),
			},
			ServiceMonitor: v1.JaegerServiceMonitorSpec{
				Enabled: &serviceMonitorEnabled,
			},
		},
	}

	return exampleJaeger
}

func deployPrometheus(t *testing.T, namespace string, name string) {
	err := fw.Client.Create(context.TODO(), getPrometheusServiceAccount(namespace, name),
		&framework.CleanupOptions{TestContext: ctx, Timeout: timeout, RetryInterval: retryInterval})
	require.NoError(t, err, "Error deploying prometheus serviceaccount")
	err = fw.Client.Create(context.TODO(), getPrometheusRole(namespace, name),
		&framework.CleanupOptions{TestContext: ctx, Timeout: timeout, RetryInterval: retryInterval})
	require.NoError(t, err, "Error deploying prometheus role")
	err = fw.Client.Create(context.TODO(), getPrometheusRoleBinding(namespace, name),
		&framework.CleanupOptions{TestContext: ctx, Timeout: timeout, RetryInterval: retryInterval})
	require.NoError(t, err, "Error deploying prometheus rolebinding")
	err = fw.Client.Create(context.TODO(), getPrometheus(namespace, name),
		&framework.CleanupOptions{TestContext: ctx, Timeout: timeout, RetryInterval: retryInterval})
	require.NoError(t, err, "Error deploying prometheus")
}

func getPrometheus(namespace string, name string) *monitoringv1.Prometheus {
	return &monitoringv1.Prometheus{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
		},
		Spec: monitoringv1.PrometheusSpec{
			ServiceMonitorSelector: &metav1.LabelSelector{},
			ServiceAccountName:     name,
		},
	}
}

func getPrometheusServiceAccount(namespace string, name string) *corev1.ServiceAccount {
	return &corev1.ServiceAccount{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
		},
	}
}

func getPrometheusRole(namespace string, name string) *rbacv1.Role {
	return &rbacv1.Role{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
		},
		Rules: []rbacv1.PolicyRule{
			{
				Verbs:     []string{"get", "list", "watch"},
				APIGroups: []string{""},
				Resources: []string{"endpoints", "pods", "services"},
			},
		},
	}
}

func getPrometheusRoleBinding(namespace string, name string) *rbacv1.RoleBinding {
	return &rbacv1.RoleBinding{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
		},
		RoleRef: rbacv1.RoleRef{
			APIGroup: "rbac.authorization.k8s.io",
			Kind:     "Role",
			Name:     name,
		},
		Subjects: []rbacv1.Subject{{
			Kind: "ServiceAccount",
			Name: name,
		}},
	}
}
