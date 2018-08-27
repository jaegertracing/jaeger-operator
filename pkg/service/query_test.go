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
