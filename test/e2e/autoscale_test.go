// +build autoscale

package e2e

import (
	"context"
	"fmt"
	"strconv"
	"testing"
	"time"

	framework "github.com/operator-framework/operator-sdk/pkg/test"
	"github.com/operator-framework/operator-sdk/pkg/test/e2eutil"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	v1 "github.com/jaegertracing/jaeger-operator/pkg/apis/jaegertracing/v1"
)

var (
	tracegenDurationInMinutes = getIntEnv("TRACEGEN_DURATION_IN_MINUTES", 30)
	quitOnFirstScale          = getBoolEnv("QUIT_ON_FIRST_SCALE", true)
	cpuResourceLimit          = "100m"
	memoryResourceLimit       = "128Mi"
	replicas                  = int32(getIntEnv("TRACEGEN_REPLICAS", 10))
)

type AutoscaleTestSuite struct {
	suite.Suite
}

func (suite *AutoscaleTestSuite) SetupSuite() {
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

func (suite *AutoscaleTestSuite) TearDownSuite() {
	handleSuiteTearDown()
}

func TestAutoscaleSuite(t *testing.T) {
	suite.Run(t, new(AutoscaleTestSuite))
}

func (suite *AutoscaleTestSuite) SetupTest() {
	t = suite.T()
}

func (suite *AutoscaleTestSuite) AfterTest(suiteName, testName string) {
	handleTestFailure()
}

func (suite *AutoscaleTestSuite) TestAutoScaleCollector() {
	if !isOpenShift(t) {
		t.Skip("This test is currently only supported on OpenShift")
	}
	waitForElasticSearch()

	jaegerInstanceName := "simple-prod"
	jaegerInstance := getSimpleProd(jaegerInstanceName, namespace, cpuResourceLimit, memoryResourceLimit)
	createAndWaitFor(jaegerInstance, jaegerInstanceName)
	defer undeployJaegerInstance(jaegerInstance)

	tracegen := createTracegenDeployment(jaegerInstanceName, namespace, tracegenDurationInMinutes, replicas)
	defer deleteTracegenDeployment(tracegen)

	waitUntilScales(jaegerInstanceName, "collector")
}

func (suite *AutoscaleTestSuite) TestAutoScaleIngester() {
	if !isOpenShift(t) {
		t.Skip("This test is currently only supported on OpenShift")
	}
	waitForElasticSearch()
	waitForKafkaInstance()

	jaegerInstanceName := "simple-streaming"
	jaegerInstance := getSimpleStreaming(jaegerInstanceName, namespace)
	createAndWaitFor(jaegerInstance, jaegerInstanceName)
	defer undeployJaegerInstance(jaegerInstance)

	tracegenReplicas := int32(1)
	tracegen := createTracegenDeployment(jaegerInstanceName, namespace, tracegenDurationInMinutes, tracegenReplicas)
	defer deleteTracegenDeployment(tracegen)

	waitUntilScales(jaegerInstanceName, "ingester")
}

func createAndWaitFor(jaegerInstance *v1.Jaeger, jaegerInstanceName string) {
	err := fw.Client.Create(context.TODO(), jaegerInstance, &framework.CleanupOptions{TestContext: ctx, Timeout: timeout, RetryInterval: retryInterval})
	require.NoError(t, err, "Error deploying example Jaeger")

	err = e2eutil.WaitForDeployment(t, fw.KubeClient, namespace, jaegerInstanceName+"-collector", 1, retryInterval, timeout)
	require.NoError(t, err, "Error waiting for collector deployment")

	err = e2eutil.WaitForDeployment(t, fw.KubeClient, namespace, jaegerInstanceName+"-query", 1, retryInterval, timeout)
	require.NoError(t, err, "Error waiting for query deployment")
	logrus.Infof("Jaeger instance %s finished deploying in %s", jaegerInstanceName, namespace)
}

func waitUntilScales(jaegerInstanceName, podSelector string) {
	maxPodCount := -1
	podListOptions := metav1.ListOptions{
		LabelSelector: "app.kubernetes.io/name=" + jaegerInstanceName + "-" + podSelector,
	}

	lastIterationTimestamp := time.Now()
	for i := 1; i <= tracegenDurationInMinutes; i++ {
		pods, err := fw.KubeClient.CoreV1().Pods(namespace).List(context.Background(), podListOptions)
		require.NoError(t, err)

		podCount := len(pods.Items)
		logrus.Infof("Iteration %d found %d pods", i, podCount)
		if podCount > maxPodCount {
			maxPodCount = podCount
			if quitOnFirstScale && i > 1 {
				break
			}
		}

		eventList, err := fw.KubeClient.CoreV1().Events(namespace).List(context.Background(), metav1.ListOptions{})
		require.NoError(t, err)
		var eventsFound = false
		for _, event := range eventList.Items {
			if event.LastTimestamp.After(lastIterationTimestamp) {
				logrus.Warnf("Event Type: %s Reason: %s Message: %s Time %v", event.Type, event.Reason, event.Message, event.LastTimestamp)
				eventsFound = true
				lastIterationTimestamp = event.LastTimestamp.Time
			}
		}
		if !eventsFound {
			lastIterationTimestamp = time.Now()
		}
		time.Sleep(1 * time.Minute)
	}
	require.Greater(t, maxPodCount, 1, "Collector never scaled")
}

func getSimpleStreaming(name, namespace string) *v1.Jaeger {
	kafkaClusterURL := fmt.Sprintf("my-cluster-kafka-brokers.%s:9092", kafkaNamespace)
	collectorOptions := make(map[string]interface{})
	collectorOptions["kafka.producer.topic"] = "jaeger-spans"
	collectorOptions["kafka.producer.brokers"] = kafkaClusterURL
	collectorOptions["kafka.producer.batch-linger"] = "1s"
	collectorOptions["kafka.producer.batch-size"] = "128000"
	collectorOptions["kafka.producer.batch-max-messages"] = "100"

	autoscale := true
	var minReplicas int32 = 1
	var maxReplicas int32 = 5

	jaeger := &v1.Jaeger{
		TypeMeta: metav1.TypeMeta{
			Kind:       "Jaeger",
			APIVersion: "jaegertracing.io/v1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
		},
		Spec: v1.JaegerSpec{
			Strategy: v1.DeploymentStrategyStreaming,
			Collector: v1.JaegerCollectorSpec{
				Options: v1.NewOptions(collectorOptions),
			},
			Ingester: v1.JaegerIngesterSpec{
				JaegerCommonSpec: v1.JaegerCommonSpec{
					Resources: corev1.ResourceRequirements{
						Limits: corev1.ResourceList{
							corev1.ResourceCPU:    resource.MustParse(cpuResourceLimit), // FIXME this might need to change
							corev1.ResourceMemory: resource.MustParse(memoryResourceLimit),
						},
					},
				},
				Options: v1.NewOptions(map[string]interface{}{
					"kafka.consumer.topic":      "jaeger-spans",
					"kafka.consumer.brokers":    kafkaClusterURL,
					"ingester.deadlockInterval": "0",
					"ingester.parallelism":      "6900",
				}),
				AutoScaleSpec: v1.AutoScaleSpec{
					Autoscale:   &autoscale,
					MinReplicas: &minReplicas,
					MaxReplicas: &maxReplicas,
				},
			},
			Storage: v1.JaegerStorageSpec{
				Type: "elasticsearch",
				Options: v1.NewOptions(map[string]interface{}{
					"es.server-urls": esServerUrls,
				}),
			},
		},
	}

	return jaeger
}

// Create a simple-prod instance with optional values for autoscaling the collector
func getSimpleProd(name, namespace, cpuResourceLimit, memoryResourceLimit string) *v1.Jaeger {
	autoscale := true
	var minReplicas int32 = 1
	var maxReplicas int32 = 5

	jaeger := &v1.Jaeger{
		TypeMeta: metav1.TypeMeta{
			Kind:       "Jaeger",
			APIVersion: "jaegertracing.io/v1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
		},
		Spec: v1.JaegerSpec{
			Collector: v1.JaegerCollectorSpec{
				JaegerCommonSpec: v1.JaegerCommonSpec{
					Resources: corev1.ResourceRequirements{
						Limits: corev1.ResourceList{
							corev1.ResourceCPU:    resource.MustParse(cpuResourceLimit),
							corev1.ResourceMemory: resource.MustParse(memoryResourceLimit),
						},
					},
				},
			},
			Strategy: v1.DeploymentStrategyProduction,
			Storage: v1.JaegerStorageSpec{
				Type: "elasticsearch",
				Options: v1.NewOptions(map[string]interface{}{
					"es.server-urls": esServerUrls,
				}),
			},
		},
	}

	autoscaleCommonSpec := v1.JaegerCommonSpec{
		Resources: corev1.ResourceRequirements{
			Limits: corev1.ResourceList{
				corev1.ResourceCPU:    resource.MustParse(cpuResourceLimit),
				corev1.ResourceMemory: resource.MustParse(memoryResourceLimit),
			},
		},
	}
	autoscaleCollectorSpec := v1.AutoScaleSpec{
		Autoscale:   &autoscale,
		MinReplicas: &minReplicas,
		MaxReplicas: &maxReplicas,
	}
	jaeger.Spec.Collector.JaegerCommonSpec = autoscaleCommonSpec
	jaeger.Spec.Collector.AutoScaleSpec = autoscaleCollectorSpec

	return jaeger
}

func createTracegenDeployment(jaegerInstanceName, namespace string, testDuration int, replicas int32) *appsv1.Deployment {
	annotations := make(map[string]string)
	annotations["sidecar.jaegertracing.io/inject"] = jaegerInstanceName
	matchLabels := make(map[string]string)
	matchLabels["app"] = "tracegen"

	serviceName := "tracegen" + strconv.Itoa(time.Now().Nanosecond())
	duration := strconv.Itoa(testDuration) + "m"
	tracegenArgs := []string{"-duration", duration, "-workers", "10", "-service", serviceName}

	tracegenInstance := &appsv1.Deployment{
		TypeMeta: metav1.TypeMeta{
			Kind:       "Deployment",
			APIVersion: "apps/v1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:        "tracegen",
			Namespace:   namespace,
			Annotations: annotations,
		},
		Spec: appsv1.DeploymentSpec{
			Replicas: &replicas,
			Selector: &metav1.LabelSelector{
				MatchLabels: matchLabels,
			},
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: matchLabels,
				},
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{
						{
							Name:  "tracegen",
							Image: "jaegertracing/jaeger-tracegen:1.19",
							Args:  tracegenArgs,
						},
					},
				},
			},
		},
	}

	logrus.Infof("Creating tracegen deployment")
	err := fw.Client.Create(context.TODO(), tracegenInstance, &framework.CleanupOptions{TestContext: ctx, Timeout: timeout, RetryInterval: retryInterval})
	require.NoError(t, err, "Error deploying tracegen")

	logrus.Infof("Waiting for  tracegen deployment")
	err = e2eutil.WaitForDeployment(t, fw.KubeClient, namespace, "tracegen", int(replicas), retryInterval, timeout)
	require.NoError(t, err, "Error waiting for tracegen deployment")
	logrus.Infof("Tracegen finished deploying in %s", namespace)

	return tracegenInstance
}

func deleteTracegenDeployment(tracegen *appsv1.Deployment) {
	err := fw.Client.Delete(context.TODO(), tracegen)
	require.NoError(t, err)
	err = e2eutil.WaitForDeletion(t, fw.Client.Client, tracegen, retryInterval, timeout)
	require.NoError(t, err)
}
