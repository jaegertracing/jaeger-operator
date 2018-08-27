package deployment

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/jaegertracing/jaeger-operator/pkg/apis/io/v1alpha1"
)

func TestQueryNegativeSize(t *testing.T) {
	jaeger := v1alpha1.NewJaeger("TestQueryNegativeSize")
	jaeger.Spec.Query.Size = -1

	query := NewQuery(jaeger)
	dep := query.Get()
	assert.Equal(t, int32(1), *dep.Spec.Replicas)
}

func TestQueryDefaultSize(t *testing.T) {
	jaeger := v1alpha1.NewJaeger("TestQueryDefaultSize")
	jaeger.Spec.Query.Size = 0

	query := NewQuery(jaeger)
	dep := query.Get()
	assert.Equal(t, int32(1), *dep.Spec.Replicas)
}

func TestQueryImage(t *testing.T) {
	query := NewQuery(v1alpha1.NewJaeger("TestQueryImage"))
	dep := query.Get()
	containers := dep.Spec.Template.Spec.Containers

	assert.Len(t, containers, 1)
	assert.Contains(t, containers[0].Image, "jaeger-query")
}

func TestQueryPodName(t *testing.T) {
	name := "TestQueryPodName"
	query := NewQuery(v1alpha1.NewJaeger(name))
	dep := query.Get()

	assert.Contains(t, dep.ObjectMeta.Name, fmt.Sprintf("%s-query", name))
}

func TestQueryServices(t *testing.T) {
	query := NewQuery(v1alpha1.NewJaeger("TestQueryServices"))
	svcs := query.Services()

	assert.Len(t, svcs, 1)
}

func TestQueryIngresses(t *testing.T) {
	query := NewQuery(v1alpha1.NewJaeger("TestQueryIngresses"))
	svcs := query.Ingresses()

	assert.Len(t, svcs, 1)
}
