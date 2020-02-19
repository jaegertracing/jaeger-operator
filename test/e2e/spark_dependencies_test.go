package e2e

import (
	"context"
	"fmt"
	"testing"
	"time"

	framework "github.com/operator-framework/operator-sdk/pkg/test"
	"github.com/operator-framework/operator-sdk/pkg/test/e2eutil"
	"github.com/pkg/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	v1 "github.com/jaegertracing/jaeger-operator/pkg/apis/jaegertracing/v1"
	"github.com/jaegertracing/jaeger-operator/pkg/util"
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
			Strategy: v1.DeploymentStrategyAllInOne,
			AllInOne: v1.JaegerAllInOneSpec{},
			Storage:  storage,
		},
	}

	err := f.Client.Create(context.Background(), j, &framework.CleanupOptions{TestContext: testCtx, Timeout: timeout, RetryInterval: retryInterval})
	if err != nil {
		return errors.WithMessagef(err, "Failed on client create")
	}
	defer undeployJaegerInstance(j)

	if storage.Type == "cassandra" {
		jobName := util.Truncate("%s-cassandra-schema-job", 52, name)
		err = WaitForJob(t, fw.KubeClient, namespace, jobName, retryInterval, timeout+1*time.Minute)
		if err != nil {
			return errors.WithMessage(err, fmt.Sprintf("Failed waiting for cassandra schema job: %s", jobName))
		}
	}

	jobName := util.Truncate("%s-spark-dependencies", 52, name)
	err = WaitForCronJob(t, f.KubeClient, namespace, jobName, retryInterval, timeout+1*time.Minute)
	if err != nil {
		return errors.WithMessage(err, fmt.Sprintf("Failed waiting for cronjob: %s", jobName))
	}

	err = WaitForJobOfAnOwner(t, f.KubeClient, namespace, jobName, retryInterval, timeout+1*time.Minute)
	if err != nil {
		return errors.WithMessage(err, fmt.Sprintf("Failed waiting for job from owner %s", jobName))
	}

	err = e2eutil.WaitForDeployment(t, f.KubeClient, namespace, name, 1, retryInterval, timeout)
	if err != nil {
		return errors.WithMessage(err, fmt.Sprintf("Failed waiting for deployment %s", name))
	}

	return nil
}
