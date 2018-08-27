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
