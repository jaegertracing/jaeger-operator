package e2e

import (
	goctx "context"
	"crypto/tls"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"os/exec"
	"runtime"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/jaegertracing/jaeger-operator/pkg/util"

	osv1 "github.com/openshift/api/route/v1"
	osv1sec "github.com/openshift/api/security/v1"
	framework "github.com/operator-framework/operator-sdk/pkg/test"
	"github.com/operator-framework/operator-sdk/pkg/test/e2eutil"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"golang.org/x/net/context"
	corev1 "k8s.io/api/core/v1"
	rbac "k8s.io/api/rbac/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/discovery"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"

	"github.com/jaegertracing/jaeger-operator/pkg/apis"
	v1 "github.com/jaegertracing/jaeger-operator/pkg/apis/jaegertracing/v1"
)

var (
	retryInterval        = time.Second * 5
	timeout              = time.Duration(getIntEnv("TEST_TIMEOUT", 2)) * time.Minute
	storageNamespace     = os.Getenv("STORAGE_NAMESPACE")
	kafkaNamespace       = os.Getenv("KAFKA_NAMESPACE")
	debugMode            = getBoolEnv("DEBUG_MODE", false)
	usingOLM             = getBoolEnv("OLM", false)
	saveLogs             = getBoolEnv("SAVE_LOGS", false)
	skipCassandraTests   = getBoolEnv("SKIP_CASSANDRA_TESTS", false)
	esServerUrls         = "http://elasticsearch." + storageNamespace + ".svc:9200"
	cassandraServiceName = "cassandra." + storageNamespace + ".svc"
	cassandraKeyspace    = "jaeger_v1_datacenter1"
	cassandraDatacenter  = "datacenter1"
	ctx                  *framework.TestCtx
	fw                   *framework.Framework
	namespace            string
	t                    *testing.T
)

func getBoolEnv(key string, defaultValue bool) bool {
	if value, ok := os.LookupEnv(key); ok {
		boolValue, err := strconv.ParseBool(value)
		if err != nil {
			logrus.Warnf("Error [%v] received converting environment variable [%s] using [%v]", err, key, boolValue)
		}
		return boolValue
	}
	return defaultValue
}

func getIntEnv(key string, defaultValue int) int {
	if value, ok := os.LookupEnv(key); ok {
		intValue, err := strconv.Atoi(value)
		if err != nil {
			logrus.Warnf("Error [%v] received converting environment variable [%s] using [%v]", err, key, value)
		}
		return intValue
	}
	return defaultValue
}

// GetPod returns pod name
func GetPod(namespace, namePrefix, containsImage string, kubeclient kubernetes.Interface) corev1.Pod {
	pods, err := kubeclient.CoreV1().Pods(namespace).List(metav1.ListOptions{})
	if err != nil {
		printTestStackTrace()
		require.NoError(t, err)
	}
	for _, pod := range pods.Items {
		if strings.HasPrefix(pod.Name, namePrefix) {
			for _, c := range pod.Spec.Containers {
				if strings.Contains(c.Image, containsImage) {
					return pod
				}
			}
		}
	}

	errorMessage := fmt.Sprintf("could not find pod in namespace %s with prefix %s and image %s", namespace, namePrefix, containsImage)
	require.FailNow(t, errorMessage)

	// We should never get here, but go requires a return statement
	emptyPod := &corev1.Pod{}
	return *emptyPod
}

func prepare(t *testing.T) (*framework.TestCtx, error) {
	t.Logf("debug mode: %v", debugMode)
	ctx := framework.NewTestCtx(t)
	// Install jaeger-operator unless we've installed it from OperatorHub
	start := time.Now()
	if !usingOLM {
		err := ctx.InitializeClusterResources(&framework.CleanupOptions{TestContext: ctx, Timeout: 10 * time.Minute, RetryInterval: retryInterval})
		if err != nil {
			t.Errorf("failed to initialize cluster resources: %v", err)
		}
	}
	namespace, err := ctx.GetNamespace()
	if err != nil {
		t.Errorf("failed to get the operator's namespace: %v", err)
	}
	logrus.Infof("Using namespace %s", namespace)

	ns, err := framework.Global.KubeClient.CoreV1().Namespaces().Get(namespace, metav1.GetOptions{})
	if err != nil {
		t.Errorf("failed to get the namespaces details: %v", err)
	}

	crb := &rbac.ClusterRoleBinding{
		ObjectMeta: metav1.ObjectMeta{
			Name: namespace + "jaeger-operator",
			OwnerReferences: []metav1.OwnerReference{
				{
					Name:       ns.Name,
					Kind:       "Namespace",
					APIVersion: "v1",
					UID:        ns.UID,
				},
			},
		},
		Subjects: []rbac.Subject{{
			Kind:      "ServiceAccount",
			Name:      "jaeger-operator",
			Namespace: namespace,
		}},
		RoleRef: rbac.RoleRef{Kind: "ClusterRole", Name: "jaeger-operator"},
	}

	if _, err := framework.Global.KubeClient.RbacV1().ClusterRoleBindings().Create(crb); err != nil {
		t.Errorf("failed to create cluster role binding: %v", err)
	}

	t.Logf("initialized cluster resources on namespace %s", namespace)

	// get global framework variables
	f := framework.Global
	// wait for the operator to be ready
	if !usingOLM {
		err := e2eutil.WaitForDeployment(t, f.KubeClient, namespace, "jaeger-operator", 1, retryInterval, timeout)
		if err != nil {
			logrus.Errorf("WaitForDeployment returned error %v", err)
			return nil, err
		}
	}
	logrus.Infof("Creation of Jaeger Operator in namespace %s took %v", namespace, time.Since(start))

	return ctx, nil
}

func getJaegerOperatorImages(kubeclient kubernetes.Interface, namespace string) (map[string]string, error) {
	imageNamesMap := make(map[string]string)

	deployment, err := kubeclient.AppsV1().Deployments(namespace).Get("jaeger-operator", metav1.GetOptions{})
	if err != nil {
		if strings.HasSuffix(err.Error(), "not found") {
			return imageNamesMap, nil
		}
		return imageNamesMap, err
	}

	containers := deployment.Spec.Template.Spec.Containers
	for _, container := range containers {
		if container.Name == "jaeger-operator" {
			for _, env := range container.Env {
				if env.Name == "WATCH_NAMESPACE" {
					imageNamesMap[container.Image] = env.Value
				}
			}
		}
	}

	return imageNamesMap, nil
}

func isOpenShift(t *testing.T) bool {
	apiList, err := availableAPIs(framework.Global.KubeConfig)
	if err != nil {
		t.Logf("Error trying to find APIs: %v\n", err)
	}

	apiGroups := apiList.Groups
	for _, group := range apiGroups {
		if group.Name == "route.openshift.io" {
			return true
		}
	}
	return false
}

func availableAPIs(kubeconfig *rest.Config) (*metav1.APIGroupList, error) {
	discoveryClient, err := discovery.NewDiscoveryClientForConfig(kubeconfig)
	if err != nil {
		return nil, err
	}

	apiList, err := discoveryClient.ServerGroups()
	if err != nil {
		return nil, err
	}

	return apiList, nil
}

func addToFrameworkSchemeForSmokeTests(t *testing.T) {
	assert.NoError(t, framework.AddToFrameworkScheme(apis.AddToScheme, &v1.JaegerList{
		TypeMeta: metav1.TypeMeta{
			Kind:       "Jaeger",
			APIVersion: "jaegertracing.io/v1",
		},
	}))
	if isOpenShift(t) {
		assert.NoError(t, framework.AddToFrameworkScheme(osv1.AddToScheme, &osv1.Route{}))
		assert.NoError(t, framework.AddToFrameworkScheme(osv1sec.AddToScheme, &osv1sec.SecurityContextConstraints{}))
	}
}

// Print a stack trace to help analyze test failures.  This is shorter and easier to read than debug.printstack()
func printTestStackTrace() {
	i := 1
	for {
		_, filename, lineNumber, ok := runtime.Caller(i)
		if !ok || !strings.Contains(filename, "jaeger-operator") {
			break
		}
		fmt.Printf("\t%s#%d\n", filename, lineNumber)
		i++
	}
}

func undeployJaegerInstance(jaeger *v1.Jaeger) bool {
	if saveLogs {
		logFileName := strings.ReplaceAll(t.Name(), "/", "-") + ".log"
		writePodLogToFile(jaeger.Namespace, "app.kubernetes.io/part-of=jaeger", "jaeger", logFileName)
	}

	if !debugMode || !t.Failed() {
		err := fw.Client.Delete(goctx.TODO(), jaeger)
		if err := fw.Client.Delete(goctx.TODO(), jaeger); err != nil {
			return false
		}

		if err = e2eutil.WaitForDeletion(t, fw.Client.Client, jaeger, retryInterval, timeout); err != nil {
			return false
		}
		return true
	}
	// Always return true, we don't care
	return true
}

func writePodLogToFile(namespace, labelSelector, containerName, logFileName string) {
	pods, err := fw.KubeClient.CoreV1().Pods(namespace).List(metav1.ListOptions{LabelSelector: labelSelector})
	if err != nil {
		logrus.Warnf("Got error listing pods in namespace %s with selector %s: %v", namespace, labelSelector, err)
		return
	}

	for _, pod := range pods.Items {
		result := fw.KubeClient.CoreV1().Pods(namespace).GetLogs(pod.Name, &corev1.PodLogOptions{Container: containerName}).Do()
		if result.Error() != nil {
			logrus.Warnf("Error getting log content %v", result.Error())
		} else {
			log, _ := result.Raw()
			err := ioutil.WriteFile(logFileName, log, 0644)
			if err != nil {
				logrus.Warnf("Error writing log content to file %s: %v\n", logFileName, err)
			}
		}
	}
}

func getJaegerInstance(name, namespace string) *v1.Jaeger {
	jaegerInstance := &v1.Jaeger{}
	key := types.NamespacedName{Name: name, Namespace: namespace}
	err := fw.Client.Get(goctx.Background(), key, jaegerInstance)
	require.NoError(t, err)
	return jaegerInstance
}

// ValidateHTTPResponseFunc should determine whether the response contains the desired content
type ValidateHTTPResponseFunc func(response *http.Response) (done bool, err error)

// WaitAndPollForHTTPResponse will try the targetURL until it gets the desired response or times out
func WaitAndPollForHTTPResponse(targetURL string, condition ValidateHTTPResponseFunc) (err error) {
	client := http.Client{Timeout: 5 * time.Second}
	request, err := http.NewRequest(http.MethodGet, targetURL, nil)
	require.NoError(t, err)
	err = wait.Poll(retryInterval, timeout, func() (done bool, err error) {
		response, err := client.Do(request)
		require.NoError(t, err)
		defer response.Body.Close()

		return condition(response)
	})

	return err
}

func handleSuiteTearDown() {
	logrus.Info("Entering TearDownSuite()")
	if saveLogs && !usingOLM {
		i := strings.Index(t.Name(), "/")
		logFileName := t.Name()[:i] + "-operator.log"
		writePodLogToFile(namespace, "name=jaeger-operator", "jaeger-operator", logFileName)
	}

	if !debugMode || !t.Failed() {
		ctx.Cleanup()
	}
}

func handleTestFailure() {
	if debugMode && t.Failed() {
		logrus.Errorf("Test %s failed\n", t.Name())
		// FIXME find a better way to terminate tests than os.Exit(1)
	}
}

type resp struct {
	Data []trace `json:"data"`
}

type trace struct {
	TraceID string `json:"traceID"`
	Spans   []span `json:"spans"`
}

type span struct {
	TraceID string `json:"traceID"`
	SpanID  string `json:"spanID"`
}

type services struct {
	Data   []string    `json:"data"`
	total  int         `json:"total"`
	limit  int         `json:"limit"`
	offset int         `json:offset`
	errors interface{} `json:"errors"`
}

func createJaegerInstanceFromFile(name, filename string) *v1.Jaeger {
	// #nosec   G204: Subprocess launching should be audited
	cmd := exec.Command("kubectl", "create", "--namespace", namespace, "--filename", filename)
	output, err := cmd.CombinedOutput()
	if err != nil && !strings.Contains(string(output), "AlreadyExists") {
		require.NoError(t, err, "Failed creating Jaeger instance with: [%s]\n", string(output))
	}

	return getJaegerInstance(name, namespace)
}

func smokeTestAllInOneExample(name, yamlFileName string) {
	smokeTestAllInOneExampleWithTimeout(name, yamlFileName, timeout+1*time.Minute)
}

func smokeTestAllInOneExampleWithTimeout(name, yamlFileName string, to time.Duration) {
	jaegerInstance := createJaegerInstanceFromFile(name, yamlFileName)
	defer undeployJaegerInstance(jaegerInstance)

	err := WaitForDeployment(t, fw.KubeClient, namespace, name, 1, retryInterval, to)
	require.NoErrorf(t, err, "Error waiting for %s to deploy", name)

	AllInOneSmokeTest(name)
}

func smokeTestProductionExample(name, yamlFileName string) {
	jaegerInstance := createJaegerInstanceFromFile(name, yamlFileName)
	defer undeployJaegerInstance(jaegerInstance)

	queryDeploymentName := name + "-query"
	collectorDeploymentName := name + "-collector"

	if jaegerInstance.Spec.Strategy == v1.DeploymentStrategyStreaming {
		ingesterDeploymentName := name + "-ingester"
		err := WaitForDeployment(t, fw.KubeClient, namespace, ingesterDeploymentName, 1, retryInterval, timeout)
		require.NoErrorf(t, err, "Error waiting for %s to deploy", ingesterDeploymentName)
	}

	err := WaitForDeployment(t, fw.KubeClient, namespace, queryDeploymentName, 1, retryInterval, timeout)
	require.NoErrorf(t, err, "Error waiting for %s to deploy", queryDeploymentName)
	err = WaitForDeployment(t, fw.KubeClient, namespace, collectorDeploymentName, 1, retryInterval, timeout)
	require.NoErrorf(t, err, "Error waiting for %s to deploy", collectorDeploymentName)

	ProductionSmokeTest(name)
}

func findRoute(t *testing.T, f *framework.Framework, name, namespace string) *osv1.Route {
	routeList := &osv1.RouteList{}
	err := wait.Poll(retryInterval, timeout, func() (bool, error) {
		if err := f.Client.List(context.Background(), routeList); err != nil {
			return false, err
		}
		if len(routeList.Items) >= 1 {
			return true, nil
		}
		return false, nil
	})

	if err != nil {
		t.Fatalf("Failed waiting for route: %v", err)
	}

	// Truncate the namespace name and use that to find the route
	target := util.DNSName(util.Truncate(name, 62-len(namespace)))
	for _, r := range routeList.Items {
		if r.Namespace == namespace && strings.HasPrefix(r.Spec.Host, target) {
			return &r
		}
	}

	t.Fatal("Could not find route")
	return nil
}

func getQueryURL(jaegerInstanceName, namespace, urlPattern string) (url string) {
	if isOpenShift(t) {
		route := findRoute(t, fw, jaegerInstanceName, namespace)
		require.Len(t, route.Status.Ingress, 1, "Wrong number of ingresses.")
		url = fmt.Sprintf("https://"+urlPattern, route.Spec.Host)
	} else {
		ingress, err := WaitForIngress(t, fw.KubeClient, namespace, jaegerInstanceName+"-query", retryInterval, timeout)
		require.NoError(t, err, "Failed waiting for ingress")
		require.Len(t, ingress.Status.LoadBalancer.Ingress, 1, "Wrong number of ingresses.")

		address := ingress.Status.LoadBalancer.Ingress[0].IP
		url = fmt.Sprintf("http://"+urlPattern, address)
	}

	return url
}

func getHTTPCLient(insecure bool) (httpClient http.Client) {
	if isOpenShift(t) {
		transport := &http.Transport{
			// #nosec  G402: TLS InsecureSkipVerify set true
			TLSClientConfig: &tls.Config{InsecureSkipVerify: insecure},
		}
		httpClient = http.Client{Timeout: 30 * time.Second, Transport: transport}
	} else {
		httpClient = http.Client{Timeout: time.Second}
	}

	return httpClient
}

func getQueryURLAndHTTPClient(jaegerInstanceName, urlPattern string, insecure bool) (string, http.Client) {
	url := getQueryURL(jaegerInstanceName, namespace, urlPattern)
	httpClient := getHTTPCLient(insecure)

	return url, httpClient
}

func createSecret(secretName, secretNamespace string, secretData map[string][]byte) *corev1.Secret {
	secret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      secretName,
			Namespace: secretNamespace,
		},
		Data: secretData,
	}

	createdSecret, err := fw.KubeClient.CoreV1().Secrets(secretNamespace).Create(secret)
	require.NoError(t, err)
	WaitForSecret(secretName, secretNamespace)
	return createdSecret
}

func deletePersistentVolumeClaims(namespace string) {
	pvcs, err := fw.KubeClient.CoreV1().PersistentVolumeClaims(kafkaNamespace).List(metav1.ListOptions{})
	require.NoError(t, err)

	emptyDeleteOptions := metav1.DeleteOptions{}
	for _, pvc := range pvcs.Items {
		logrus.Infof("Deleting PVC %s from namespace %s", pvc.Name, namespace)
		fw.KubeClient.CoreV1().PersistentVolumeClaims(kafkaNamespace).Delete(pvc.Name, &emptyDeleteOptions)
	}
}
