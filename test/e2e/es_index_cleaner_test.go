package e2e

import (
	"context"
	"fmt"
	"io/ioutil"
	"net/http"
	"strings"
	"testing"

	framework "github.com/operator-framework/operator-sdk/pkg/test"
	"github.com/operator-framework/operator-sdk/pkg/test/e2eutil"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/wait"

	"github.com/jaegertracing/jaeger-operator/pkg/apis/jaegertracing/v1"
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
	numberOfDays := 0
	j := &v1.Jaeger{
		TypeMeta: metav1.TypeMeta{
			Kind:       "Jaeger",
			APIVersion: "jaegertracing.io/v1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
		},
		Spec: v1.JaegerSpec{
			Strategy: "allInOne",
			Storage: v1.JaegerStorageSpec{
				Type: "elasticsearch",
				Options: v1.NewOptions(map[string]interface{}{
					"es.server-urls": esServerUrls,
				}),
				EsIndexCleaner: v1.JaegerEsIndexCleanerSpec{
					Schedule:     "*/1 * * * *",
					NumberOfDays: &numberOfDays,
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

	esPod, err := GetPod("default", "elasticsearch", "elasticsearch", f.KubeClient)
	if err != nil {
		return err
	}
	portForwES, closeChanES, err := CreatePortForward(esPod.Namespace, esPod.Name, []string{"9200"}, f.KubeConfig)
	if err != nil {
		return err
	}
	defer portForwES.Close()
	defer close(closeChanES)

	err = SmokeTest("http://localhost:16686/api/traces", "http://localhost:14268/api/traces", "foo-bar", retryInterval, timeout)
	if err != nil {
		return err
	}

	if flag, err := hasIndex(t); !flag || err != nil {
		return fmt.Errorf("jaeger-span index not found prior to es-index-cleaner: err = %v", err)
	}

	err = WaitForCronJob(t, f.KubeClient, namespace, fmt.Sprintf("%s-es-index-cleaner", name), retryInterval, timeout)
	if err != nil {
		return err
	}

	err = WaitForJobOfAnOwner(t, f.KubeClient, namespace, fmt.Sprintf("%s-es-index-cleaner", name), retryInterval, timeout)
	if err != nil {
		return err
	}

	return wait.Poll(retryInterval, timeout, func() (done bool, err error) {
		flag, err := hasIndex(t)
		return !flag, err
	})
}

func hasIndex(t *testing.T) (bool, error) {
	c := http.Client{}
	req, err := http.NewRequest(http.MethodGet, "http://localhost:9200/_cat/indices", nil)
	if err != nil {
		return false, err
	}
	resp, err := c.Do(req)
	if err != nil {
		return false, err
	}
	defer resp.Body.Close()

	bodyBytes, err := ioutil.ReadAll(resp.Body)
	bodyString := string(bodyBytes)

	return strings.Contains(bodyString, "jaeger-span-"), nil
}
