package service

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/jaegertracing/jaeger-operator/pkg/apis/io/v1alpha1"
)

func TestAgentServiceNameAndPorts(t *testing.T) {
	name := "TestAgentServiceNameAndPorts"
	selector := map[string]string{"app": "myapp", "jaeger": name, "jaeger-component": "agent"}

	jaeger := v1alpha1.NewJaeger(name)
	svc := NewAgentService(jaeger, selector)
	assert.Equal(t, svc.ObjectMeta.Name, fmt.Sprintf("%s-agent", name))

	ports := map[int32]bool{
		5775: false,
		5778: false,
		6831: false,
		6832: false,
	}

	for _, port := range svc.Spec.Ports {
		ports[port.Port] = true
	}

	for k, v := range ports {
		assert.Equal(t, v, true, "Expected port %v to be specified, but wasn't", k)
	}

}

func TestAgentServiceAnnotations(t *testing.T) {
	name := "TestAgentServiceAnnotations"
	k, v := "some-annotation-name", "some-annotation-value"
	annotations := map[string]string{k: v}
	selector := map[string]string{"app": name}

	j := v1alpha1.NewJaeger(name)
	j.Spec.Agent.Annotations = annotations

	svc := NewAgentService(j, selector)
	assert.Equal(t, len(annotations), len(svc.Annotations))
	assert.Equal(t, v, svc.Annotations[k])
}

func TestAgentServiceLabels(t *testing.T) {
	name := "TestAgentServiceLabels"
	k, v := "some-label-name", "some-label-value"
	labels := map[string]string{k: v}
	selector := map[string]string{"app": name}

	j := v1alpha1.NewJaeger(name)
	j.Spec.Agent.Labels = labels

	svc := NewAgentService(j, selector)
	assert.Equal(t, len(labels)+len(selector), len(svc.Labels))
	assert.Equal(t, v, svc.Labels[k])
}
