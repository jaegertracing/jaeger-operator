// +build smoke

package e2e

import (
	"testing"

	framework "github.com/operator-framework/operator-sdk/pkg/test"
	"github.com/stretchr/testify/assert"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/jaegertracing/jaeger-operator/pkg/apis"
	"github.com/jaegertracing/jaeger-operator/pkg/apis/jaegertracing/v1"
)

func TestSmoke(t *testing.T) {
	assert.NoError(t, framework.AddToFrameworkScheme(apis.AddToScheme, &v1.JaegerList{
		TypeMeta: metav1.TypeMeta{
			Kind:       "Jaeger",
			APIVersion: "jaegertracing.io/v1",
		},
	}))

	t.Run("smoke", func(t *testing.T) {
		t.Run("my-jaeger", JaegerAllInOne)
		t.Run("daemonset", DaemonSet)
		t.Run("daemonset-with-ingress", DaemonSetWithIngress)
		t.Run("sidecar", Sidecar)
	})
}
