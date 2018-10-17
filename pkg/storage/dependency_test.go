package storage

import (
	"testing"

	"github.com/jaegertracing/jaeger-operator/pkg/apis/io/v1alpha1"
	"github.com/stretchr/testify/assert"
)

func TestDefaultDependencies(t *testing.T) {
	jaeger := v1alpha1.NewJaeger("TestCassandraDependencies")
	assert.Len(t, Dependencies(jaeger), 0)
}

func TestCassandraDependencies(t *testing.T) {
	jaeger := v1alpha1.NewJaeger("TestCassandraDependencies")
	jaeger.Spec.Storage.Type = "CASSANDRA" // should be converted to lowercase
	assert.Len(t, Dependencies(jaeger), 1)
}
