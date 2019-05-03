package e2e

import (
	goctx "context"
	"fmt"
	"testing"

	"github.com/stretchr/testify/require"
	framework "github.com/operator-framework/operator-sdk/pkg/test"
	"github.com/operator-framework/operator-sdk/pkg/test/e2eutil"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/jaegertracing/jaeger-operator/pkg/apis/jaegertracing/v1"
)

func SimplestJaeger(t *testing.T) {
	ctx, err := prepare(t)
	if (err != nil) {
		ctx.Cleanup()
		require.FailNow(t, "Failed in prepare")
	}
	defer ctx.Cleanup()

	if err := simplest(t, framework.Global, ctx); err != nil {
		t.Fatal(err)
	}
}

func simplest(t *testing.T, f *framework.Framework, ctx *framework.TestCtx) error {
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
		Spec: v1.JaegerSpec{},
	}
	err = f.Client.Create(goctx.TODO(), exampleJaeger, &framework.CleanupOptions{TestContext: ctx, Timeout: timeout, RetryInterval: retryInterval})
	if err != nil {
		return err
	}

	return e2eutil.WaitForDeployment(t, f.KubeClient, namespace, "my-jaeger", 1, retryInterval, timeout)
}
