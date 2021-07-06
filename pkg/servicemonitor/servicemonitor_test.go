package servicemonitor

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"k8s.io/apimachinery/pkg/types"

	v1 "github.com/jaegertracing/jaeger-operator/pkg/apis/jaegertracing/v1"
)

func TestServiceMonitor(t *testing.T) {
	const port int32 = 8888
	name := t.Name()

	jaegerInstance := v1.NewJaeger(types.NamespacedName{Namespace: name, Name: name})
	serviceMonitor := NewServiceMonitor(jaegerInstance)

	assert.Equal(t, fmt.Sprintf("%s-metrics", name), serviceMonitor.Name)
	assert.Equal(t, name, serviceMonitor.Namespace)
}
