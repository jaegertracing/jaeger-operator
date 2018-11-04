package storage

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/jaegertracing/jaeger-operator/pkg/apis/io/v1alpha1"
)

func TestCassandraCreateSchemaDisabled(t *testing.T) {
	falseVar := false

	jaeger := v1alpha1.NewJaeger("TestCassandraCreateSchemaDisabled")
	jaeger.Spec.Storage.CassandraCreateSchema.Enabled = &falseVar

	assert.Len(t, cassandraDeps(jaeger), 0)
}

func TestCassandraCreateSchemaEnabled(t *testing.T) {
	trueVar := true

	jaeger := v1alpha1.NewJaeger("TestCassandraCreateSchemaEnabled")
	jaeger.Spec.Storage.CassandraCreateSchema.Enabled = &trueVar

	jobs := cassandraDeps(jaeger)
	assert.Len(t, jobs, 1)

	assert.Equal(t, "false", jobs[0].Spec.Template.Annotations["prometheus.io/scrape"])
	assert.Equal(t, "false", jobs[0].Spec.Template.Annotations["sidecar.istio.io/inject"])

	criticalpod, found := jobs[0].Spec.Template.Annotations["scheduler.alpha.kubernetes.io/critical-pod"]
	assert.True(t, found)
	assert.Equal(t, "", criticalpod)
}

func TestCassandraCreateSchemaEnabledNil(t *testing.T) {
	jaeger := v1alpha1.NewJaeger("TestCassandraCreateSchemaEnabledNil")

	assert.Nil(t, jaeger.Spec.Storage.CassandraCreateSchema.Enabled)
	assert.Len(t, cassandraDeps(jaeger), 1)
}
