package e2e

import (
	"context"
	"fmt"
	"testing"

	framework "github.com/operator-framework/operator-sdk/pkg/test"
	"github.com/operator-framework/operator-sdk/pkg/test/e2eutil"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/jaegertracing/jaeger-operator/pkg/apis/io/v1alpha1"
)

func EsIndexCleaner(t *testing.T) {
	testCtx := prepare(t)
	defer testCtx.Cleanup()
	if err := esIndexCleanerTest(t, framework.Global, testCtx); err != nil {
		t.Fatal(err)
	}
}

func esIndexCleanerTest(t *testing.T, f *framework.Framework, testCtx *framework.TestCtx) error {
	namespace, err := testCtx.GetNamespace()
	if err != nil {
		return fmt.Errorf("could not get namespace: %v", err)
	}

	name := "test-es-index-cleaner"
	j := &v1alpha1.Jaeger{
		TypeMeta: metav1.TypeMeta{
			Kind:       "Jaeger",
			APIVersion: "io.jaegertracing/v1alpha1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
		},
		Spec: v1alpha1.JaegerSpec{
			Strategy: "allInOne",
			Storage: v1alpha1.JaegerStorageSpec{
				Type: "elasticsearch",
				Options: v1alpha1.NewOptions(map[string]interface{}{
					"es.server-urls": "http://elasticsearch.default.svc:9200",
				}),
				EsIndexCleaner: v1alpha1.JaegerEsIndexCleanerSpec{
					Schedule: "*/1 * * * *",
				},
			},
		},
	}

	err = f.Client.Create(context.Background(), j, &framework.CleanupOptions{TestContext: testCtx, Timeout: timeout, RetryInterval: retryInterval})
	if err != nil {
		return err
	}

	err = e2eutil.WaitForDeployment(t, f.KubeClient, namespace, name, 1, retryInterval, timeout)
	if err != nil {
		return nil
	}

	// create span, otherwise index cleaner fails - there would not be indices
	jaegerPod, err := GetPod(namespace, name, "jaegertracing/all-in-one", f.KubeClient)
	if err != nil {
		return err
	}
	portForw, closeChan, err := CreatePortForward(namespace, jaegerPod.Name, []string{"16686", "14268"}, f.KubeConfig)
	if err != nil {
		return err
	}
	defer portForw.Close()
	defer close(closeChan)
	err = SmokeTest("http://localhost:16686/api/traces", "http://localhost:14268/api/traces", "foo-bar", retryInterval, timeout)
	if err != nil {
		return err
	}

	err = WaitForCronJob(t, f.KubeClient, namespace, fmt.Sprintf("%s-es-index-cleaner", name), retryInterval, timeout)
	if err != nil {
		return err
	}

	err = WaitForJobOfAnOwner(t, f.KubeClient, namespace, fmt.Sprintf("%s-es-index-cleaner", name), retryInterval, timeout)
	if err != nil {
		return err
	}
	return nil
}
