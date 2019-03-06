package storage

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/jaegertracing/jaeger-operator/pkg/apis/jaegertracing/v1"
)

func TestDefaultDependencies(t *testing.T) {
	jaeger := v1.NewJaeger("TestCassandraDependencies")
	assert.Len(t, Dependencies(jaeger), 0)
}

func TestCassandraDependencies(t *testing.T) {
	jaeger := v1.NewJaeger("TestCassandraDependencies")
	jaeger.Spec.Storage.Type = "CASSANDRA" // should be converted to lowercase
	assert.Len(t, Dependencies(jaeger), 1)
}

func TestESDependencies(t *testing.T) {
	jaeger := v1.NewJaeger("charmander")
	jaeger.Spec.Storage.Type = "elasticsearch" // should be converted to lowercase
	jaeger.Spec.Storage.Options = v1.NewOptions(map[string]interface{}{"es.use-aliases": "true"})
	deps := Dependencies(jaeger)
	assert.Len(t, deps, 1)
	assert.Equal(t, "charmander-es-rollover-create-mapping", deps[0].Name)
}
