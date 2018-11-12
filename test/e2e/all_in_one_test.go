package e2e

import (
	goctx "context"
	"fmt"
	"io/ioutil"
	"net/http"
	"testing"
	"time"

	"github.com/jaegertracing/jaeger-operator/pkg/apis/io/v1alpha1"
	framework "github.com/operator-framework/operator-sdk/pkg/test"
	"github.com/operator-framework/operator-sdk/pkg/test/e2eutil"
	"github.com/sirupsen/logrus"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/wait"
)

func JaegerAllInOne(t *testing.T) {
	t.Parallel()
	ctx := prepare(t)
	defer ctx.Cleanup()

	if err := allInOneTest(t, framework.Global, ctx); err != nil {
		t.Fatal(err)
	}

	if err := allInOneWithUIBasePathTest(t, framework.Global, ctx); err != nil {
		t.Fatal(err)
	}
}

func allInOneTest(t *testing.T, f *framework.Framework, ctx *framework.TestCtx) error {
	namespace, err := ctx.GetNamespace()
	if err != nil {
		return fmt.Errorf("could not get namespace: %v", err)
	}

	// create jaeger custom resource
	exampleJaeger := &v1alpha1.Jaeger{
		TypeMeta: metav1.TypeMeta{
			Kind:       "Jaeger",
			APIVersion: "io.jaegertracing/v1alpha1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      "my-jaeger",
			Namespace: namespace,
		},
		Spec: v1alpha1.JaegerSpec{
			Strategy: "allInOne",
			AllInOne: v1alpha1.JaegerAllInOneSpec{
				Options: v1alpha1.NewOptions(map[string]interface{}{
					"log-level":         "debug",
					"memory.max-traces": 10000,
				}),
			},
		},
	}

	logrus.Infof("passing %v", exampleJaeger)
	err = f.Client.Create(goctx.TODO(), exampleJaeger, &framework.CleanupOptions{TestContext: ctx, Timeout: timeout, RetryInterval: retryInterval})
	if err != nil {
		return err
	}

	return e2eutil.WaitForDeployment(t, f.KubeClient, namespace, "my-jaeger", 1, retryInterval, timeout)
}

func allInOneWithUIBasePathTest(t *testing.T, f *framework.Framework, ctx *framework.TestCtx) error {
	cleanupOptions := &framework.CleanupOptions{TestContext: ctx, Timeout: timeout, RetryInterval: retryInterval}
	namespace, err := ctx.GetNamespace()
	if err != nil {
		return fmt.Errorf("could not get namespace: %v", err)
	}

	basePath := "/jaeger"

	j := &v1alpha1.Jaeger{
		TypeMeta: metav1.TypeMeta{
			Kind:       "Jaeger",
			APIVersion: "io.jaegertracing/v1alpha1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      "all-in-one-with-base-path",
			Namespace: namespace,
		},
		Spec: v1alpha1.JaegerSpec{
			Strategy: "allInOne",
			AllInOne: v1alpha1.JaegerAllInOneSpec{
				Options: v1alpha1.NewOptions(map[string]interface{}{
					"query.base-path": basePath,
				}),
			},
		},
	}

	err = f.Client.Create(goctx.TODO(), j, cleanupOptions)
	if err != nil {
		return err
	}

	err = WaitForIngress(t, f.KubeClient, namespace, "all-in-one-with-base-path-query", retryInterval, timeout)
	if err != nil {
		return err
	}

	i, err := f.KubeClient.ExtensionsV1beta1().Ingresses(namespace).Get("all-in-one-with-base-path-query", metav1.GetOptions{})
	if err != nil {
		return err
	}

	if len(i.Status.LoadBalancer.Ingress) != 1 {
		return fmt.Errorf("Wrong number of ingresses. Expected 1, was %v", len(i.Status.LoadBalancer.Ingress))
	}

	address := i.Status.LoadBalancer.Ingress[0].IP
	url := fmt.Sprintf("http://%s%s/search", address, basePath)
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

		return len(body) > 0, nil
	})
}
