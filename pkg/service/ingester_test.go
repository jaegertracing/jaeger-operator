package service

import (
	"testing"

	"github.com/stretchr/testify/assert"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"

	v1 "github.com/jaegertracing/jaeger-operator/pkg/apis/jaegertracing/v1"
)

func TestIngesterAdminService(t *testing.T) {
	name := "TestIngesterAdminService"
	jaegerInstance := v1.NewJaeger(types.NamespacedName{Name: name})

	selector := map[string]string{"app": "myapp", "jaeger": name, "jaeger-component": "ingester"}
	svc := NewIngesterAdminService(jaegerInstance, selector)

	assert.Contains(t, svc.Name, "ingester-admin")
	assert.Contains(t, svc.Spec.Ports, corev1.ServicePort{Name: "admin", Port: 14270})
}
