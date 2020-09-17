// +build autoscale

package e2e

import (
	"context"
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
	testDurationInMinutes       = getIntEnv("TEST_DURATION_IN_MINUTES", 30)
	quitOnFirstScale            = getBoolEnv("QUIT_ON_FIRST_SCALE", false)
	cpuResourceLimit            = "100m"
	memoryResourceLimit         = "128Mi"
	replicas              int32 = 10
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
	// We probably don't need this but it doesn't hurt
	err := WaitForStatefulset(t, fw.KubeClient, storageNamespace, "elasticsearch", retryInterval, timeout)
	require.NoError(t, err, "Error waiting for elasticsearch")

	jaegerInstanceName := "simple-prod"
	exampleJaeger := getSimpleProd(jaegerInstanceName, namespace, cpuResourceLimit, memoryResourceLimit)
	err = fw.Client.Create(context.TODO(), exampleJaeger, &framework.CleanupOptions{TestContext: ctx, Timeout: timeout, RetryInterval: retryInterval})
	require.NoError(t, err, "Error deploying example Jaeger")
	defer undeployJaegerInstance(exampleJaeger)

	err = e2eutil.WaitForDeployment(t, fw.KubeClient, namespace, jaegerInstanceName+"-collector", 1, retryInterval, timeout)
	require.NoError(t, err, "Error waiting for collector deployment")

	err = e2eutil.WaitForDeployment(t, fw.KubeClient, namespace, jaegerInstanceName+"-query", 1, retryInterval, timeout)
	require.NoError(t, err, "Error waiting for query deployment")

	logrus.Infof("Jaeger deployfinished, deploying tracegen in %s", namespace)

	tracegen := getTracegenDeployment(jaegerInstanceName, namespace, testDurationInMinutes, replicas)
	err = fw.Client.Create(context.TODO(), tracegen, &framework.CleanupOptions{TestContext: ctx, Timeout: timeout, RetryInterval: retryInterval})
	require.NoError(t, err, "Error deploying tracegen")

	err = e2eutil.WaitForDeployment(t, fw.KubeClient, namespace, "tracegen", int(replicas), retryInterval, timeout)
	require.NoError(t, err, "Error waiting for tracegen deployment")

	logrus.Infof("Tracegen deployed in %s", namespace)
	maxCollectorCount := -1
	lastIterationTimestamp := time.Now()
	for i := 0; i < testDurationInMinutes; i++ {
		pods, err := fw.KubeClient.CoreV1().Pods(namespace).List(context.Background(), metav1.ListOptions{LabelSelector: "app.kubernetes.io/name=simple-prod-collector"}) //FIXME use jaegerInstanceName
		require.NoError(t, err)

		collectorCount := len(pods.Items)
		logrus.Infof("Iteration %d found %d pods", i, collectorCount)
		if collectorCount > maxCollectorCount {
			maxCollectorCount = collectorCount
			if quitOnFirstScale && i > 1 { // TODO is this hacky?  we could just check collector count
				break
			}
		}

		// Print events since last iteration.  TODO replace this with Events(namespace).Watch
		eventList, err := fw.KubeClient.CoreV1().Events(namespace).List(context.Background(), metav1.ListOptions{})
		require.NoError(t, err)
		for _, event := range eventList.Items {
			if event.LastTimestamp.After(lastIterationTimestamp) {
				logrus.Warnf("Event Type: %s Reason: %s Message: %s Time %v", event.Type, event.Reason, event.Message, event.LastTimestamp)
			}
		}

		// FIXME mucking about with hpa info.  Either get the correct info or remove this
		/*
			hpas, err := fw.KubeClient.AutoscalingV2beta2().HorizontalPodAutoscalers(namespace).List(context.Background(), metav1.ListOptions{
				LabelSelector: "app.kubernetes.io/name=" + jaegerInstanceName + "-collector",
			})
			require.NoError(t, err)
			require.Equal(t, len(hpas.Items), 1)
			currentMetrics := hpas.Items[0].Status.CurrentMetrics
			for _, f := range currentMetrics {
				if f.Resource == nil {
					logrus.Infof("Resource is nil")
				} else {
					logrus.Infof("Resource.Current is %v ", f.Resource)
				}
			}

		*/

		lastIterationTimestamp = time.Now()
		time.Sleep(1 * time.Minute)
	}

	require.Greater(t, maxCollectorCount, 1, "Collector never scaled")
}

// Create a simple-prod instance with autoscale, replicas, and resources set.
func getSimpleProd(name, namespace, cpuResourceLimit, memoryResourceLimit string) *v1.Jaeger {
	ingressEnabled := true
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
				AutoScaleSpec: v1.AutoScaleSpec{
					Autoscale:   &autoscale,
					MinReplicas: &minReplicas,
					MaxReplicas: &maxReplicas,
				},
				JaegerCommonSpec: v1.JaegerCommonSpec{
					Resources: corev1.ResourceRequirements{
						Limits: corev1.ResourceList{
							corev1.ResourceCPU:    resource.MustParse(cpuResourceLimit),
							corev1.ResourceMemory: resource.MustParse(memoryResourceLimit),
						},
					},
				},
			},
			Ingress: v1.JaegerIngressSpec{
				Enabled:  &ingressEnabled,
				Security: v1.IngressSecurityNoneExplicit,
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

	return jaeger
}

/*
+Tragegen options
+  -debug   Whether to set DEBUG flag on the spans to force sampling
+  -duration For how long to run the test
+  -firehose Whether to set FIREHOSE flag on the spans to skip indexing
+  -marshal Whether to marshal trace context via HTTP headers
+  -pause duration How long to pause before finishing trace (default 1µs)  -- must be a string because duration.SHortHumanDuration is stupid
+  -service string Service name to use (default "tracegen")
+  -traces int Number of traces to generate in each worker (ignored if duration is provided (default 1)
+  -workers int Number of workers (goroutines) to run (default 1)
+*/
// TODO pass a randomized service name to simplify checking traces?  What other parameters should there be?
func getTracegenDeployment(jaegerInstanceName, namespace string, testDuration int, replicas int32) *appsv1.Deployment {
	annotations := make(map[string]string)
	annotations["sidecar.jaegertracing.io/inject"] = jaegerInstanceName
	matchLabels := make(map[string]string)
	matchLabels["app"] = "tracegen"

	duration := strconv.Itoa(testDuration) + "m"
	tracegenArgs := []string{"-duration", duration, "-workers", "10"} // TODO pass in number of workers

	var jaegerAgentArgs []string
	jaegerAgentArgs = append(jaegerAgentArgs, "--reporter.grpc.host-port=dns:///"+jaegerInstanceName+"-collector-headless."+namespace+":14250")
	if isOpenShift(t) {
		jaegerAgentArgs = append(jaegerAgentArgs, "--reporter.grpc.tls.skip-host-verify")
		jaegerAgentArgs = append(jaegerAgentArgs, "--reporter.grpc.tls.enabled=true")
	}
	sidecarEnv := &corev1.EnvVar{
		Name:  "POD_NAME",
		Value: "",
		ValueFrom: &corev1.EnvVarSource{
			FieldRef: &corev1.ObjectFieldSelector{
				APIVersion: "v1",
				FieldPath:  "metadata.name",
			},
		},
	}
	sidecarEnvs := []corev1.EnvVar{*sidecarEnv}

	sidecarPort := &corev1.ContainerPort{
		Name:          "jg-compact-trft",
		HostPort:      0,
		ContainerPort: 6831,
		Protocol:      "UDP",
		HostIP:        "",
	}
	sidecarPorts := []corev1.ContainerPort{*sidecarPort}

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
							Image: "jaegertracing/jaeger-tracegen:1.19", // FIXME where should we get this from?
							Args:  tracegenArgs,
						},
						{
							Name:  "jaeger-agent",
							Image: "jaegertracing/jaeger-agent:1.19",
							Args:  jaegerAgentArgs,
							Env:   sidecarEnvs,
							Ports: sidecarPorts,
						},
					},
				},
			},
		},
	}

	return tracegenInstance
}
