package storage

import (
	"testing"

	"github.com/jaegertracing/jaeger-operator/pkg/version"

	"github.com/spf13/viper"
	"github.com/stretchr/testify/assert"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"

	v1 "github.com/jaegertracing/jaeger-operator/pkg/apis/jaegertracing/v1"
)

func TestCassandraCustomImage(t *testing.T) {
	jaeger := v1.NewJaeger(types.NamespacedName{Name: "my-instance"})
	jaeger.Spec.Storage.CassandraCreateSchema.Image = "mynamespace/image:version"

	b := cassandraDeps(jaeger)
	assert.Len(t, b, 1)
	assert.Len(t, b[0].Spec.Template.Spec.Containers, 1)
	assert.Equal(t, "mynamespace/image:version", b[0].Spec.Template.Spec.Containers[0].Image)
}

func TestCassandraCustomTraceTTL(t *testing.T) {
	jaeger := v1.NewJaeger(types.NamespacedName{Name: "my-instance"})
	jaeger.Spec.Storage.CassandraCreateSchema.TraceTTL = "168h" // 7d

	b := cassandraDeps(jaeger)
	assert.Len(t, b, 1)
	assert.Len(t, b[0].Spec.Template.Spec.Containers, 1)
	foundValue := ""
	for _, e := range b[0].Spec.Template.Spec.Containers[0].Env {
		if e.Name == "TRACE_TTL" {
			foundValue = e.Value
		}
	}
	assert.Equal(t, "604800", foundValue, "unexpected TRACE_TTL environment var value")
}

func TestCassandraCustomTraceTTLParseError(t *testing.T) {
	jaeger := v1.NewJaeger(types.NamespacedName{Name: "my-instance"})
	// this does not work. time.ParseDuration can not handle "days"
	// TRACE_TTL should fallback to default value
	jaeger.Spec.Storage.CassandraCreateSchema.TraceTTL = "7d"

	b := cassandraDeps(jaeger)
	assert.Len(t, b, 1)
	assert.Len(t, b[0].Spec.Template.Spec.Containers, 1)
	foundValue := ""
	for _, e := range b[0].Spec.Template.Spec.Containers[0].Env {
		if e.Name == "TRACE_TTL" {
			foundValue = e.Value
		}
	}
	assert.Equal(t, "172800", foundValue, "unexpected TRACE_TTL environment var value")
}

func TestDefaultImage(t *testing.T) {
	viper.Set("jaeger-cassandra-schema-image", "jaegertracing/theimage")
	defer viper.Reset()

	jaeger := v1.NewJaeger(types.NamespacedName{Name: "my-instance"})

	b := cassandraDeps(jaeger)
	assert.Len(t, b, 1)
	assert.Len(t, b[0].Spec.Template.Spec.Containers, 1)
	assert.Equal(t, "jaegertracing/theimage:"+version.Get().Jaeger, b[0].Spec.Template.Spec.Containers[0].Image)
}

func TestCassandraCreateSchemaDisabled(t *testing.T) {
	falseVar := false

	jaeger := v1.NewJaeger(types.NamespacedName{Name: "TestCassandraCreateSchemaDisabled"})
	jaeger.Spec.Storage.CassandraCreateSchema.Enabled = &falseVar

	assert.Len(t, cassandraDeps(jaeger), 0)
}

func TestCassandraCreateSchemaEnabled(t *testing.T) {
	trueVar := true

	jaeger := v1.NewJaeger(types.NamespacedName{Name: "TestCassandraCreateSchemaEnabled"})
	jaeger.Spec.Storage.CassandraCreateSchema.Enabled = &trueVar

	assert.Len(t, cassandraDeps(jaeger), 1)
}

func TestCassandraCreateSchemaEnabledNil(t *testing.T) {
	jaeger := v1.NewJaeger(types.NamespacedName{Name: "TestCassandraCreateSchemaEnabledNil"})

	assert.Nil(t, jaeger.Spec.Storage.CassandraCreateSchema.Enabled)
	assert.Len(t, cassandraDeps(jaeger), 1)
}

func TestCassandraCreateSchemaCustomTimeout(t *testing.T) {
	jaeger := v1.NewJaeger(types.NamespacedName{Name: "TestCassandraCreateSchemaCustomTimeout"})

	jaeger.Spec.Storage.CassandraCreateSchema.Timeout = "3m"

	b := cassandraDeps(jaeger)
	assert.Len(t, b, 1)
	assert.Equal(t, int64(180), *b[0].Spec.ActiveDeadlineSeconds)
}

func TestCassandraCreateSchemaDefaultTimeout(t *testing.T) {
	jaeger := v1.NewJaeger(types.NamespacedName{Name: "TestCassandraCreateSchemaDefaultTimeout"})

	b := cassandraDeps(jaeger)
	assert.Len(t, b, 1)
	assert.Equal(t, int64(86400), *b[0].Spec.ActiveDeadlineSeconds)
}

func TestCassandraCreateSchemaInvalidTimeout(t *testing.T) {
	jaeger := v1.NewJaeger(types.NamespacedName{Name: "TestCassandraCreateSchemaInvalidTimeout"})

	jaeger.Spec.Storage.CassandraCreateSchema.Timeout = "3mm"

	b := cassandraDeps(jaeger)
	assert.Len(t, b, 1)
	assert.Equal(t, int64(86400), *b[0].Spec.ActiveDeadlineSeconds)
}

func TestCassandraCreateSchemaSecurityContext(t *testing.T) {
	var user, group int64 = 111, 222
	expectedSecurityContext := &corev1.PodSecurityContext{RunAsUser: &user, RunAsGroup: &group}

	jaeger := v1.NewJaeger(types.NamespacedName{Name: "TestCassandraCreateSchemaSecurityContext"})
	jaeger.Spec.JaegerCommonSpec.SecurityContext = expectedSecurityContext

	b := cassandraDeps(jaeger)

	assert.Len(t, b, 1)
	assert.Equal(t, b[0].Spec.Template.Spec.SecurityContext, expectedSecurityContext)
}
