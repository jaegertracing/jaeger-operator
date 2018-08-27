package e2e

import (
	goctx "context"
	"fmt"
	"testing"

	"github.com/jaegertracing/jaeger-operator/pkg/apis/io/v1alpha1"

	framework "github.com/operator-framework/operator-sdk/pkg/test"
	"github.com/operator-framework/operator-sdk/pkg/test/e2eutil"
	"github.com/sirupsen/logrus"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func JaegerAllInOne(t *testing.T) {
	ctx := prepare(t)
	defer ctx.Cleanup(t)

	if err := allInOneTest(t, framework.Global, ctx); err != nil {
		t.Fatal(err)
	}
}

func allInOneTest(t *testing.T, f *framework.Framework, ctx framework.TestCtx) error {
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
			Strategy: "all-in-one",
			AllInOne: v1alpha1.JaegerAllInOneSpec{
				Options: v1alpha1.NewOptions(map[string]interface{}{
					"log-level":         "debug",
					"memory.max-traces": 10000,
				}),
			},
		},
	}

	logrus.Infof("passing %v", exampleJaeger)
	err = f.DynamicClient.Create(goctx.TODO(), exampleJaeger)
	if err != nil {
		return err
	}

	return e2eutil.WaitForDeployment(t, f.KubeClient, namespace, "my-jaeger", 1, retryInterval, timeout)
}
