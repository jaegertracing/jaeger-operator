// +build self-provisioned-elasticsearch

package e2e

import (
	"testing"

	framework "github.com/operator-framework/operator-sdk/pkg/test"
	"github.com/stretchr/testify/assert"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/jaegertracing/jaeger-operator/pkg/apis"
	"github.com/jaegertracing/jaeger-operator/pkg/apis/jaegertracing/v1"
)

func TestSelfProvisionedElasticsearch(t *testing.T) {
	assert.NoError(t, framework.AddToFrameworkScheme(apis.AddToScheme, &v1.JaegerList{
		TypeMeta: metav1.TypeMeta{
			Kind:       "Jaeger",
			APIVersion: "jaegertracing.io/v1",
		},
	}))

	// Don't start tests until elasticsearch is ready
	err := WaitForStatefulset(t, framework.Global.KubeClient, storageNamespace, "elasticsearch", retryInterval, timeout)
	if err != nil {
		t.Fatal(err)
	}

	t.Run("elasticsearch", func(t *testing.T) {
		t.Run("spark-dependencies-es", SelfProvisionedElasticsearch)
	})
}
