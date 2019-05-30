package e2e

import (
	"fmt"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/operator-framework/operator-sdk/pkg/test/e2eutil"
	"github.com/pkg/errors"
	osv1 "github.com/openshift/api/route/v1"
	osv1sec "github.com/openshift/api/security/v1"
	framework "github.com/operator-framework/operator-sdk/pkg/test"
	"github.com/stretchr/testify/assert"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/discovery"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"

	"github.com/jaegertracing/jaeger-operator/pkg/apis"
	"github.com/jaegertracing/jaeger-operator/pkg/apis/jaegertracing/v1"
)

var (
	retryInterval        = time.Second * 5
	timeout              = time.Minute * 2
	storageNamespace     = os.Getenv("STORAGE_NAMESPACE")
	kafkaNamespace       = os.Getenv("KAFKA_NAMESPACE")
	esServerUrls         = "http://elasticsearch." + storageNamespace + ".svc:9200"
	cassandraServiceName = "cassandra." + storageNamespace + ".svc"
	ctx                  *framework.TestCtx
	fw                   *framework.Framework
	namespace            string
	t                    *testing.T
)

// GetPod returns pod name
func GetPod(namespace, namePrefix, containsImage string, kubeclient kubernetes.Interface) (corev1.Pod, error) {
	pods, err := kubeclient.CoreV1().Pods(namespace).List(metav1.ListOptions{})
	if err != nil {
		return corev1.Pod{}, err
	}
	for _, pod := range pods.Items {
		if strings.HasPrefix(pod.Name, namePrefix) {
			for _, c := range pod.Spec.Containers {
				if strings.Contains(c.Image, containsImage) {
					return pod, nil
				}
			}
		}
	}
	return corev1.Pod{}, fmt.Errorf("could not find pod with image %s", containsImage)
}

func prepare(t *testing.T) (*framework.TestCtx, error) {
	ctx := framework.NewTestCtx(t)
	err := ctx.InitializeClusterResources(&framework.CleanupOptions{TestContext: ctx, Timeout: timeout, RetryInterval: retryInterval})
	if err != nil {
		t.Fatalf("failed to initialize cluster resources: %v", err)
	}
	namespace, err := ctx.GetNamespace()
	if err != nil {
		t.Fatal(err)
	}

	t.Log("Initialized cluster resources. Namespace: " + namespace)

	// get global framework variables
	f := framework.Global
	// wait for the operator to be ready
	err = e2eutil.WaitForDeployment(t, f.KubeClient, namespace, "jaeger-operator", 1, retryInterval, timeout)
	if err != nil {
		return nil, err
	}

	return ctx, nil
}

func getJaegerOperatorImages(kubeclient kubernetes.Interface, namespace string) (map[string]string, error) {
	imageNamesMap := make(map[string]string)

	deployment, err := kubeclient.AppsV1().Deployments(namespace).Get("jaeger-operator", metav1.GetOptions{IncludeUninitialized: false})
	if err != nil {
		if !strings.Contains(err.Error(), "not found") {
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

	if len(imageNamesMap) == 0 {
		return imageNamesMap, errors.New("Could not find the operator image")
	} else {
		return imageNamesMap, nil
	}
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
