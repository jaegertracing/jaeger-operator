package e2e

import (
	goctx "context"
	"fmt"
	"testing"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	framework "github.com/operator-framework/operator-sdk/pkg/test"
	"github.com/operator-framework/operator-sdk/pkg/test/e2eutil"

	"github.com/jaegertracing/jaeger-operator/pkg/apis/io/v1alpha1"
)

func SimpleProd(t *testing.T) {
	t.Parallel()
	ctx := prepare(t)
	defer ctx.Cleanup()

	if err := simpleProd(t, framework.Global, ctx); err != nil {
		t.Fatal(err)
	}
}

func simpleProd(t *testing.T, f *framework.Framework, ctx *framework.TestCtx) error {
	namespace, err := ctx.GetNamespace()
	if err != nil {
		return fmt.Errorf("could not get namespace: %v", err)
	}

	err = WaitForStatefulset(t, f.KubeClient, "default", "elasticsearch", retryInterval, timeout)
	if err != nil {
		return err
	}

	// create jaeger custom resource
	exampleJaeger := &v1alpha1.Jaeger{
		TypeMeta: metav1.TypeMeta{
			Kind:       "Jaeger",
			APIVersion: "io.jaegertracing/v1alpha1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      "simple-prod",
			Namespace: namespace,
		},
		Spec: v1alpha1.JaegerSpec{
			Strategy: "production",
			Storage: v1alpha1.JaegerStorageSpec{
				Type: "elasticsearch",
				Options: v1alpha1.NewOptions(map[string]interface{}{
					"es.server-urls": "http://elasticsearch.default.svc:9200",
					"es.username":    "elastic",
					"es.password":    "changeme",
				}),
			},
		},
	}
	err = f.Client.Create(goctx.TODO(), exampleJaeger, &framework.CleanupOptions{TestContext: ctx, Timeout: timeout, RetryInterval: retryInterval})
	if err != nil {
		return err
	}

	err = e2eutil.WaitForDeployment(t, f.KubeClient, namespace, "simple-prod-collector", 1, retryInterval, timeout)
	if err != nil {
		return err
	}

	return e2eutil.WaitForDeployment(t, f.KubeClient, namespace, "simple-prod-query", 1, retryInterval, timeout)
}
