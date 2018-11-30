package e2e

import (
	goctx "context"
	"fmt"
	"github.com/pkg/errors"
	"k8s.io/client-go/kubernetes"
	"strings"
	"testing"

	"github.com/jaegertracing/jaeger-operator/pkg/apis/io/v1alpha1"
	framework "github.com/operator-framework/operator-sdk/pkg/test"
	"github.com/operator-framework/operator-sdk/pkg/test/e2eutil"
	"github.com/sirupsen/logrus"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// Cassandra runs a test with Cassandra as the backing storage
func Cassandra(t *testing.T) {
	ctx := prepare(t)
	defer ctx.Cleanup()

	if err := cassandraTest(t, framework.Global, ctx); err != nil {
		t.Fatal(err)
	}
}

func cassandraTest(t *testing.T, f *framework.Framework, ctx *framework.TestCtx) error {
	cleanupOptions := &framework.CleanupOptions{TestContext: ctx, Timeout: timeout, RetryInterval: retryInterval}
	namespace, err := ctx.GetNamespace()
	if err != nil {
		return fmt.Errorf("could not get namespace: %v", err)
	}

	j := &v1alpha1.Jaeger{
		TypeMeta: metav1.TypeMeta{
			Kind:       "Jaeger",
			APIVersion: "io.jaegertracing/v1alpha1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      "with-cassandra",
			Namespace: namespace,
		},
		Spec: v1alpha1.JaegerSpec{
			Strategy: "allInOne",
			AllInOne: v1alpha1.JaegerAllInOneSpec{},
			Storage: v1alpha1.JaegerStorageSpec{
				Type:    "cassandra",
				Options: v1alpha1.NewOptions(map[string]interface{}{"cassandra.servers": "cassandra.default.svc"}),
			},
		},
	}

	logrus.Infof("passing %v", j)
	err = f.Client.Create(goctx.TODO(), j, cleanupOptions)
	if err != nil {
		return err
	}

	err = WaitForJob(t, f.KubeClient, namespace, "with-cassandra-cassandra-schema-job", retryInterval, timeout)
	if err != nil {
		return err
	}

	err = e2eutil.WaitForDeployment(t, f.KubeClient, namespace, "with-cassandra", 1, retryInterval, timeout)
	if err != nil {
		return err
	}

	podName, err := GetPodName(namespace, "with-cassandra", f.KubeClient)
	if err != nil {
		return err
	}
	portForw, closeChan, err := CreatePortForward(namespace, podName, []string{"16686:16686", "14268:14268"}, f.KubeConfig)
	if err != nil {
		return err
	}
	defer close(closeChan)
	defer portForw.Close()
	go func() { portForw.ForwardPorts() }()
	<- portForw.Ready
	return SmokeTest("http://localhost:16686/api/traces", "http://localhost:14268/api/traces", "foobar", retryInterval, timeout)
}

func GetPodName(namespace, ownerNamePrefix string, kubeclient kubernetes.Interface) (string, error) {
	pods, err := kubeclient.CoreV1().Pods(namespace).List(metav1.ListOptions{IncludeUninitialized:true})
	if err != nil {
		return "", err
	}
	for _, pod := range pods.Items {
		for _, or := range pod.OwnerReferences {

			if strings.HasPrefix(or.Name, ownerNamePrefix) {
				return pod.Name, nil
			}
		}
	}
	return "", errors.New(fmt.Sprintf("could not find pod of owner %s", ownerNamePrefix))
}
