package ingress

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/jaegertracing/jaeger-operator/pkg/apis/io/v1alpha1"
)

func TestQueryIngress(t *testing.T) {
	jaeger := v1alpha1.NewJaeger("TestQueryIngress")
	ingress := NewQueryIngress(jaeger)
	assert.Contains(t, ingress.Spec.Backend.ServiceName, "query")
}

func TestQueryIngressLabels(t *testing.T) {
	name := "TestQueryIngressLabels"
	k, v := "some-label-name", "some-label-value"
	labels := map[string]string{k: v}

	j := v1alpha1.NewJaeger(name)
	j.Spec.Query.Labels = labels

	q := NewQueryIngress(j)
	assert.Equal(t, len(labels), len(q.Labels))
	assert.Equal(t, v, q.Labels[k])
}

func TestQueryIngressAnnotations(t *testing.T) {
	name := "TestQueryIngressAnnotations"
	k, v := "some-annotation-name", "some-annotation-value"
	annotations := map[string]string{k: v}

	j := v1alpha1.NewJaeger(name)
	j.Spec.Query.Annotations = annotations

	q := NewQueryIngress(j)
	assert.Equal(t, len(annotations), len(q.Annotations))
	assert.Equal(t, v, q.Annotations[k])
}
