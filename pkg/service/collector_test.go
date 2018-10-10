package service

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/jaegertracing/jaeger-operator/pkg/apis/io/v1alpha1"
)

func TestCollectorServiceNameAndPorts(t *testing.T) {
	name := "TestCollectorServiceNameAndPorts"
	selector := map[string]string{"app": "myapp", "jaeger": name, "jaeger-component": "collector"}

	jaeger := v1alpha1.NewJaeger(name)
	svc := NewCollectorService(jaeger, selector)
	assert.Equal(t, svc.ObjectMeta.Name, fmt.Sprintf("%s-collector", name))

	ports := map[int32]bool{
		9411:  false,
		14267: false,
		14268: false,
	}

	for _, port := range svc.Spec.Ports {
		ports[port.Port] = true
	}

	for k, v := range ports {
		assert.Equal(t, v, true, "Expected port %v to be specified, but wasn't", k)
	}

}

func TestCollectorServiceAnnotations(t *testing.T) {
	name := "TestCollectorServiceAnnotations"
	k, v := "some-annotation-name", "some-annotation-value"
	annotations := map[string]string{k: v}
	selector := map[string]string{"app": name}

	j := v1alpha1.NewJaeger(name)
	j.Spec.Collector.Annotations = annotations

	svc := NewCollectorService(j, selector)
	assert.Equal(t, len(annotations), len(svc.Annotations))
	assert.Equal(t, v, svc.Annotations[k])
}

func TestCollectorServiceLabels(t *testing.T) {
	name := "TestCollectorServiceLabels"
	k, v := "some-label-name", "some-label-value"
	labels := map[string]string{k: v}
	selector := map[string]string{"app": name}

	j := v1alpha1.NewJaeger(name)
	j.Spec.Collector.Labels = labels

	svc := NewCollectorService(j, selector)
	assert.Equal(t, len(labels)+len(selector), len(svc.Labels))
	assert.Equal(t, v, svc.Labels[k])
}
