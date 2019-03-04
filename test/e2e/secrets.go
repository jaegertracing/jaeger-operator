package e2e

import (
	goctx "context"
	"errors"
	"fmt"
	"testing"

	framework "github.com/operator-framework/operator-sdk/pkg/test"
	"github.com/operator-framework/operator-sdk/pkg/test/e2eutil"
	"github.com/sirupsen/logrus"
	"k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/jaegertracing/jaeger-operator/pkg/apis/io/v1alpha1"
)

// Secrets tests if secrets are mounted properly.
func Secrets(t *testing.T) {
	ctx := prepare(t)
	defer ctx.Cleanup()

	if err := secretTest(t, framework.Global, ctx); err != nil {
		t.Fatal(err)
	}
}

func secretTest(t *testing.T, f *framework.Framework, ctx *framework.TestCtx) error {
	cleanupOptions := &framework.CleanupOptions{TestContext: ctx, Timeout: timeout, RetryInterval: retryInterval}
	namespace, err := ctx.GetNamespace()
	if err != nil {
		return fmt.Errorf("could not get namespace: %v", err)
	}

	data := map[string]string{
		"hello": "world",
	}

	secret := v1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name: "mysecret",
		},
		StringData: data,
	}

	logrus.Infof("passing %v", secret)
	_, err = f.KubeClient.CoreV1().Secrets(namespace).Create(&secret)
	if err != nil {
		return err
	}

	j := &v1alpha1.Jaeger{
		TypeMeta: metav1.TypeMeta{
			Kind:       "Jaeger",
			APIVersion: "io.jaegertracing/v1alpha1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      "with-secret",
			Namespace: namespace,
		},
		Spec: v1alpha1.JaegerSpec{
			Strategy: "allInOne",
			AllInOne: v1alpha1.JaegerAllInOneSpec{},
			Storage: v1alpha1.JaegerStorageSpec{
				Type:       "cassandra",
				Options:    v1alpha1.NewOptions(map[string]interface{}{"cassandra.servers": "cassandra.default.svc"}),
				SecretName: secret.Name,
			},
		},
	}

	logrus.Infof("passing %v", j)
	err = f.Client.Create(goctx.TODO(), j, cleanupOptions)
	if err != nil {
		return err
	}

	err = WaitForJob(t, f.KubeClient, namespace, "with-secret-job", retryInterval, timeout)
	if err != nil {
		return err
	}

	err = e2eutil.WaitForDeployment(t, f.KubeClient, namespace, "with-secret-deployment", 1, retryInterval, timeout)
	if err != nil {
		return err
	}

	i, err := f.KubeClient.ExtensionsV1beta1().Deployments(namespace).Get("with-secret-deployment", metav1.GetOptions{})
	if err != nil {
		return err
	}

	for _, c := range i.Spec.Template.Spec.Containers {
		for _, e := range c.Env {
			if e.Name == "hello" && e.Value == "world" {
				return nil
			}
		}
	}

	return errors.New("Env variable not set correctly")
}
