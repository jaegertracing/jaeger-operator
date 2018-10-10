package service

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/jaegertracing/jaeger-operator/pkg/apis/io/v1alpha1"
)

func TestQueryServiceNameAndPorts(t *testing.T) {
	name := "TestQueryServiceNameAndPorts"
	selector := map[string]string{"app": "myapp", "jaeger": name, "jaeger-component": "query"}

	jaeger := v1alpha1.NewJaeger(name)
	svc := NewQueryService(jaeger, selector)

	assert.Equal(t, fmt.Sprintf("%s-query", name), svc.ObjectMeta.Name)
	assert.Len(t, svc.Spec.Ports, 1)
	assert.Equal(t, int32(16686), svc.Spec.Ports[0].Port)
}

func TestQueryServiceAnnotations(t *testing.T) {
	name := "TestQueryServiceAnnotations"
	k, v := "some-annotation-name", "some-annotation-value"
	annotations := map[string]string{k: v}
	selector := map[string]string{"app": name}

	j := v1alpha1.NewJaeger(name)
	j.Spec.Query.Annotations = annotations

	svc := NewQueryService(j, selector)
	assert.Equal(t, len(annotations), len(svc.Annotations))
	assert.Equal(t, v, svc.Annotations[k])
}

func TestQueryServiceLabels(t *testing.T) {
	name := "TestQueryServiceLabels"
	k, v := "some-label-name", "some-label-value"
	labels := map[string]string{k: v}
	selector := map[string]string{"app": name}

	j := v1alpha1.NewJaeger(name)
	j.Spec.Query.Labels = labels

	svc := NewQueryService(j, selector)
	assert.Equal(t, len(labels)+len(selector), len(svc.Labels))
	assert.Equal(t, v, svc.Labels[k])
}
