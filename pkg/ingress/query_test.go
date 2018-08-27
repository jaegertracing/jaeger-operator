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
