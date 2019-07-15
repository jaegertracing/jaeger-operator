package e2e

import (
	"fmt"
	goctx "context"
	"os"
	"runtime"
	"strings"
	"testing"
	"time"

	osv1 "github.com/openshift/api/route/v1"
	osv1sec "github.com/openshift/api/security/v1"
	framework "github.com/operator-framework/operator-sdk/pkg/test"
	"github.com/operator-framework/operator-sdk/pkg/test/e2eutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	rbac "k8s.io/api/rbac/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/discovery"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"

	"github.com/jaegertracing/jaeger-operator/pkg/apis"
	v1 "github.com/jaegertracing/jaeger-operator/pkg/apis/jaegertracing/v1"
)

var (
	retryInterval        = time.Second * 5
	timeout              = time.Minute * 2
	storageNamespace     = os.Getenv("STORAGE_NAMESPACE")
	kafkaNamespace       = os.Getenv("KAFKA_NAMESPACE")
	noSetup              = os.Getenv("NO_SETUP")
	esServerUrls         = "http://elasticsearch." + storageNamespace + ".svc:9200"
	cassandraServiceName = "cassandra." + storageNamespace + ".svc"
	ctx                  *framework.TestCtx
	fw                   *framework.Framework
	namespace            string
	t                    *testing.T
)

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
	ctx := framework.NewTestCtx(t)
	doSetup := len(noSetup) == 0
	if doSetup {
		err := ctx.InitializeClusterResources(&framework.CleanupOptions{TestContext: ctx, Timeout: timeout, RetryInterval: retryInterval})
		if err != nil {
			t.Fatalf("failed to initialize cluster resources: %v", err)
		}
	}
	namespace, err := ctx.GetNamespace()
	if err != nil {
		t.Fatal(err)
	}

	roleName := namespace + "-jaeger-operator-cluster-role-crbs"
	cr := &rbac.ClusterRole{
		ObjectMeta: metav1.ObjectMeta{
			Name: roleName,
		},
		Rules: []rbac.PolicyRule{{
			APIGroups: []string{"rbac.authorization.k8s.io"},
			Resources: []string{"clusterrolebindings"},
			Verbs:     []string{"*"},
		}},
	}
	if _, err := framework.Global.KubeClient.Rbac().ClusterRoles().Create(cr); err != nil {
		t.Fatal(err)
	}

	crb := &rbac.ClusterRoleBinding{
		ObjectMeta: metav1.ObjectMeta{
			Name: namespace + "-jaeger-operator-cluster-admin",
		},
		Subjects: []rbac.Subject{{
			Kind:      "ServiceAccount",
			Name:      "jaeger-operator",
			Namespace: namespace,
		}},
		RoleRef: rbac.RoleRef{Kind: "ClusterRole", Name: roleName},
	}

	if _, err := framework.Global.KubeClient.Rbac().ClusterRoleBindings().Create(crb); err != nil {
		t.Fatal(err)
	}

	t.Log("Initialized cluster resources. Namespace: " + namespace)

	// get global framework variables
	f := framework.Global
	// wait for the operator to be ready
	if doSetup {
		err = e2eutil.WaitForDeployment(t, f.KubeClient, namespace, "jaeger-operator", 1, retryInterval, timeout)
		if err != nil {
			return nil, err
		}
	}

	return ctx, nil
}

func getJaegerOperatorImages(kubeclient kubernetes.Interface, namespace string) (map[string]string, error) {
	imageNamesMap := make(map[string]string)

	deployment, err := kubeclient.AppsV1().Deployments(namespace).Get("jaeger-operator", metav1.GetOptions{IncludeUninitialized: false})
	if err != nil {
		if strings.HasSuffix(err.Error(), "not found") {
			return imageNamesMap, nil
		} else {
			return imageNamesMap, err
		}
	} else {
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

func undeployJaegerInstance(jaeger *v1.Jaeger) {
	err := fw.Client.Delete(goctx.TODO(), jaeger)
	require.NoError(t, err, "Error undeploying Jaeger")
	err = e2eutil.WaitForDeletion(t, fw.Client.Client, jaeger, retryInterval, timeout)
	require.NoError(t, err)
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
