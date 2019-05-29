package storage

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/jaegertracing/jaeger-operator/pkg/apis/jaegertracing/v1"
)

func TestCassandraCreateSchemaDisabled(t *testing.T) {
	falseVar := false

	jaeger := v1.NewJaeger("TestCassandraCreateSchemaDisabled")
	jaeger.Spec.Storage.CassandraCreateSchema.Enabled = &falseVar

	assert.Len(t, cassandraDeps(jaeger), 0)
}

func TestCassandraCreateSchemaEnabled(t *testing.T) {
	trueVar := true

	jaeger := v1.NewJaeger("TestCassandraCreateSchemaEnabled")
	jaeger.Spec.Storage.CassandraCreateSchema.Enabled = &trueVar

	assert.Len(t, cassandraDeps(jaeger), 1)
}

func TestCassandraCreateSchemaEnabledNil(t *testing.T) {
	jaeger := v1.NewJaeger("TestCassandraCreateSchemaEnabledNil")

	assert.Nil(t, jaeger.Spec.Storage.CassandraCreateSchema.Enabled)
	assert.Len(t, cassandraDeps(jaeger), 1)
}

func TestCassandraCreateSchemaAfterCompletionTTL(t *testing.T) {
	trueVar := true

	jaeger := v1.NewJaeger("TestCassandraCreateSchemaAfterCompletionTTL")
	jaeger.Spec.Storage.CassandraCreateSchema.Enabled = &trueVar
	afterCompletionTTL := int32(100)
	jaeger.Spec.Storage.CassandraCreateSchema.AfterCompletionTTL = &afterCompletionTTL
	cjob := cassandraDeps(jaeger)
	assert.Equal(t, afterCompletionTTL, *cjob[0].Spec.TTLSecondsAfterFinished)
}
