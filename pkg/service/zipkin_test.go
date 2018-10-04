package service

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/jaegertracing/jaeger-operator/pkg/apis/io/v1alpha1"
)

func TestZipkinServiceNameAndPort(t *testing.T) {
	name := "TestZipkinServiceNameAndPort"
	selector := map[string]string{"app": "myapp", "jaeger": name, "jaeger-component": "zipkin"}

	jaeger := v1alpha1.NewJaeger(name)
	svc := NewZipkinService(jaeger, selector)

	assert.Equal(t, fmt.Sprintf("%s-zipkin", name), svc.ObjectMeta.Name)
	assert.Len(t, svc.Spec.Ports, 1)
	assert.Equal(t, int32(9411), svc.Spec.Ports[0].Port)
}

func TestZipkinServiceAnnotations(t *testing.T) {
	name := "TestZipkinServiceAnnotations"
	k, v := "some-annotation-name", "some-annotation-value"
	annotations := map[string]string{k: v}
	selector := map[string]string{"app": name}

	j := v1alpha1.NewJaeger(name)
	j.Spec.Collector.Annotations = annotations

	svc := NewZipkinService(j, selector)
	assert.Equal(t, len(annotations), len(svc.Annotations))
	assert.Equal(t, v, svc.Annotations[k])
}

func TestZipkinServiceLabels(t *testing.T) {
	name := "TestZipkinServiceLabels"
	k, v := "some-label-name", "some-label-value"
	labels := map[string]string{k: v}
	selector := map[string]string{"app": name}

	j := v1alpha1.NewJaeger(name)
	j.Spec.Collector.Labels = labels

	svc := NewZipkinService(j, selector)
	assert.Equal(t, len(labels)+len(selector), len(svc.Labels))
	assert.Equal(t, v, svc.Labels[k])
}
