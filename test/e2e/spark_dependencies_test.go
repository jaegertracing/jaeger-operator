package e2e

import (
	"context"
	"fmt"
	"testing"

	framework "github.com/operator-framework/operator-sdk/pkg/test"
	"github.com/operator-framework/operator-sdk/pkg/test/e2eutil"
	"github.com/pkg/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	v1 "github.com/jaegertracing/jaeger-operator/pkg/apis/jaegertracing/v1"
)

func sparkTest(t *testing.T, f *framework.Framework, testCtx *framework.TestCtx, storage v1.JaegerStorageSpec) error {
	storage.Dependencies = v1.JaegerDependenciesSpec{
		// run immediately
		Schedule: "*/1 * * * *",
	}

	name := "test-spark-deps"
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
			AllInOne: v1.JaegerAllInOneSpec{},
			Storage:  storage,
		},
	}

	err := f.Client.Create(context.Background(), j, &framework.CleanupOptions{TestContext: testCtx, Timeout: timeout, RetryInterval: retryInterval})
	if err != nil {
		return errors.WithMessagef(err, "Failed on client create")
	}
	defer undeployJaegerInstance(j)

	err = WaitForCronJob(t, f.KubeClient, namespace, fmt.Sprintf("%s-spark-dependencies", name), retryInterval, timeout)
	if err != nil {
		return errors.WithMessage(err, "Failed waiting for cron job")
	}

	err = WaitForJobOfAnOwner(t, f.KubeClient, namespace, fmt.Sprintf("%s-spark-dependencies", name), retryInterval, timeout)
	if err != nil {
		return errors.WithMessage(err, "Failed waiting for Job Of An Owner")
	}

	err = e2eutil.WaitForDeployment(t, f.KubeClient, namespace, name, 1, retryInterval, timeout)
	if err != nil {
		return errors.WithMessage(err, "Failed waiting for deployment ")
	} else {
		return nil
	}
}
