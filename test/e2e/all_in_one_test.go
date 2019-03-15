package e2e

import (
	goctx "context"
	"fmt"
	"io/ioutil"
	"net/http"
	"strings"
	"testing"
	"time"

	framework "github.com/operator-framework/operator-sdk/pkg/test"
	"github.com/operator-framework/operator-sdk/pkg/test/e2eutil"
	log "github.com/sirupsen/logrus"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/wait"

	"github.com/jaegertracing/jaeger-operator/pkg/apis/jaegertracing/v1"
)

const TrackingID = "MyTrackingId"

func JaegerAllInOne(t *testing.T) {
	ctx := prepare(t)
	defer ctx.Cleanup()

	if err := allInOneTest(t, framework.Global, ctx); err != nil {
		t.Fatal(err)
	}

	if err := allInOneWithUIConfigTest(t, framework.Global, ctx); err != nil {
		t.Fatal(err)
	}
}

func allInOneTest(t *testing.T, f *framework.Framework, ctx *framework.TestCtx) error {
	namespace, err := ctx.GetNamespace()
	if err != nil {
		return fmt.Errorf("could not get namespace: %v", err)
	}

	// create jaeger custom resource
	exampleJaeger := &v1.Jaeger{
		TypeMeta: metav1.TypeMeta{
			Kind:       "Jaeger",
			APIVersion: "jaegertracing.io/v1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      "my-jaeger",
			Namespace: namespace,
		},
		Spec: v1.JaegerSpec{
			Strategy: "allInOne",
			AllInOne: v1.JaegerAllInOneSpec{
				Options: v1.NewOptions(map[string]interface{}{
					"log-level":         "debug",
					"memory.max-traces": 10000,
				}),
			},
		},
	}

	log.Infof("passing %v", exampleJaeger)
	err = f.Client.Create(goctx.TODO(), exampleJaeger, &framework.CleanupOptions{TestContext: ctx, Timeout: timeout, RetryInterval: retryInterval})
	if err != nil {
		return err
	}

	return e2eutil.WaitForDeployment(t, f.KubeClient, namespace, "my-jaeger", 1, retryInterval, timeout)
}

func allInOneWithUIConfigTest(t *testing.T, f *framework.Framework, ctx *framework.TestCtx) error {
	cleanupOptions := &framework.CleanupOptions{TestContext: ctx, Timeout: timeout, RetryInterval: retryInterval}
	namespace, err := ctx.GetNamespace()
	if err != nil {
		return fmt.Errorf("could not get namespace: %v", err)
	}

	basePath := "/jaeger"

	j := &v1.Jaeger{
		TypeMeta: metav1.TypeMeta{
			Kind:       "Jaeger",
			APIVersion: "jaegertracing.io/v1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      "all-in-one-with-ui-config",
			Namespace: namespace,
		},
		Spec: v1.JaegerSpec{
			Strategy: "allInOne",
			AllInOne: v1.JaegerAllInOneSpec{
				Options: v1.NewOptions(map[string]interface{}{
					"query.base-path": basePath,
				}),
			},
			UI: v1.JaegerUISpec{
				Options: v1.NewFreeForm(map[string]interface{}{
					"tracking": map[string]interface{}{
						"gaID": TrackingID,
					},
				}),
			},
		},
	}

	j.Spec.Annotations = map[string]string{
		"nginx.ingress.kubernetes.io/ssl-redirect": "false",
	}

	err = f.Client.Create(goctx.TODO(), j, cleanupOptions)
	if err != nil {
		return err
	}

	err = e2eutil.WaitForDeployment(t, f.KubeClient, namespace, "all-in-one-with-ui-config", 1, retryInterval, timeout)
	if err != nil {
		return err
	}

	queryPod, err := GetPod(namespace, "all-in-one-with-ui-config", "jaegertracing/all-in-one", f.KubeClient)
	if err != nil {
		return err
	}

	portForward, closeChan, err := CreatePortForward(namespace, queryPod.Name, []string{"16686"}, f.KubeConfig)
	if err != nil {
		return err
	}
	defer portForward.Close()
	defer close(closeChan)



	url := fmt.Sprintf("http://localhost:16686/%s/search", basePath)
	c := http.Client{Timeout: time.Second}

	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return err
	}

	return wait.Poll(retryInterval, timeout, func() (done bool, err error) {
		res, err := c.Do(req)
		if err != nil {
			return false, err
		}

		if res.StatusCode != 200 {
			return false, fmt.Errorf("unexpected status code %d", res.StatusCode)
		}

		body, err := ioutil.ReadAll(res.Body)
		if err != nil {
			return false, err
		}

		if len(body) == 0 {
			return false, fmt.Errorf("empty body")
		}

		if !strings.Contains(string(body), TrackingID) {
			return false, fmt.Errorf("body does not include tracking id: %s", TrackingID)
		}

		return true, nil
	})
}
