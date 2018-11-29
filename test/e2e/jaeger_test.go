package e2e

import (
	"testing"
	"time"

	framework "github.com/operator-framework/operator-sdk/pkg/test"
	"github.com/operator-framework/operator-sdk/pkg/test/e2eutil"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/jaegertracing/jaeger-operator/pkg/apis"
	"github.com/jaegertracing/jaeger-operator/pkg/apis/io/v1alpha1"
)

var (
	retryInterval = time.Second * 5
	timeout       = time.Minute * 1
)

func TestJaeger(t *testing.T) {
	jaegerList := &v1alpha1.JaegerList{
		TypeMeta: metav1.TypeMeta{
			Kind:       "Jaeger",
			APIVersion: "io.jaegertracing/v1alpha1",
		},
	}

	err := framework.AddToFrameworkScheme(apis.AddToScheme, jaegerList)
	if err != nil {
		t.Fatalf("failed to add custom resource scheme to framework: %v", err)
	}

	t.Run("jaeger-group", func(t *testing.T) {
		t.Run("my-jaeger", JaegerAllInOne)
		t.Run("my-other-jaeger", JaegerAllInOne)

		t.Run("simplest", SimplestJaeger)
		t.Run("simple-prod", SimpleProd)

		t.Run("daemonset", DaemonSet)
		t.Run("sidecar", Sidecar)
		t.Run("cassandra", Cassandra)
		t.Run("spark-dependencies-es", SparkDependenciesElasticsearch)
		t.Run("spark-dependencies-cass", SparkDependenciesCassandra)
	})
}

func prepare(t *testing.T) *framework.TestCtx {
	ctx := framework.NewTestCtx(t)
	err := ctx.InitializeClusterResources(&framework.CleanupOptions{TestContext: ctx, Timeout: timeout, RetryInterval: retryInterval})
	if err != nil {
		t.Fatalf("failed to initialize cluster resources: %v", err)
	}
	namespace, err := ctx.GetNamespace()
	if err != nil {
		t.Fatal(err)
	}

	t.Log("Initialized cluster resources. Namespace: " + namespace)

	// get global framework variables
	f := framework.Global
	// wait for the operator to be ready
	err = e2eutil.WaitForDeployment(t, f.KubeClient, namespace, "jaeger-operator", 1, retryInterval, timeout)
	if err != nil {
		t.Fatal(err)
	}

	return ctx
}

type resp struct {
	Data []trace `json:"data"`
}

type trace struct {
	TraceID string `json:"traceID"`
	Spans   []span `json:"spans"`
}

type span struct {
	TraceID string `json:"traceID"`
	SpanID  string `json:"spanID"`
}
