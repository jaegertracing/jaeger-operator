package storage

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"k8s.io/apimachinery/pkg/types"

	v1 "github.com/jaegertracing/jaeger-operator/pkg/apis/jaegertracing/v1"
)

func TestDefaultDependencies(t *testing.T) {
	jaeger := v1.NewJaeger(types.NamespacedName{Name: "TestCassandraDependencies"})
	assert.Len(t, Dependencies(jaeger), 0)
}

func TestCassandraDependencies(t *testing.T) {
	jaeger := v1.NewJaeger(types.NamespacedName{Name: "TestCassandraDependencies"})
	jaeger.Spec.Storage.Type = "CASSANDRA" // should be converted to lowercase
	assert.Len(t, Dependencies(jaeger), 1)
}

func TestESDependencies(t *testing.T) {
	jaeger := v1.NewJaeger(types.NamespacedName{Name: "charmander"})
	jaeger.Spec.Storage.Type = "elasticsearch" // should be converted to lowercase
	jaeger.Spec.Storage.Options = v1.NewOptions(map[string]interface{}{"es.use-aliases": "true"})
	deps := Dependencies(jaeger)
	assert.Len(t, deps, 1)
	assert.Equal(t, "charmander-es-rollover-create-mapping", deps[0].Name)
}
