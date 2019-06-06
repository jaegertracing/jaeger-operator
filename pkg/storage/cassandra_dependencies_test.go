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

func TestCassandraCreateSchemaTTLSecondsAfterFinished(t *testing.T) {
	trueVar := true

	jaeger := v1.NewJaeger("TestCassandraCreateSchemaTTLSecondsAfterFinished")
	jaeger.Spec.Storage.CassandraCreateSchema.Enabled = &trueVar
	ttlSecondsAfterFinished := int32(100)
	jaeger.Spec.Storage.CassandraCreateSchema.TTLSecondsAfterFinished = &ttlSecondsAfterFinished
	cjob := cassandraDeps(jaeger)
	assert.Equal(t, ttlSecondsAfterFinished, *cjob[0].Spec.TTLSecondsAfterFinished)
}
