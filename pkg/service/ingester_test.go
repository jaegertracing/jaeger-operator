package service

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"k8s.io/apimachinery/pkg/types"

	v1 "github.com/jaegertracing/jaeger-operator/pkg/apis/jaegertracing/v1"
)

func TestIngesterService(t *testing.T) {
	name := "testingesterservice"
	jaegerInstance := v1.NewJaeger(types.NamespacedName{Name: name})
	selector := map[string]string{"app": "myapp", "jaeger": name, "jaeger-component": "ingester"}
	service := NewIngesterService(jaegerInstance, selector)

	assert.Equal(t, fmt.Sprintf("%s-ingester", name), service.Name)
}
