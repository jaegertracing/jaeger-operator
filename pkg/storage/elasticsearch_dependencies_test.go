package storage

import (
	"testing"

	"github.com/stretchr/testify/assert"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"

	v1 "github.com/jaegertracing/jaeger-operator/pkg/apis/jaegertracing/v1"
	"github.com/jaegertracing/jaeger-operator/pkg/util"
)

func TestEnableRollover(t *testing.T) {
	tests := []struct {
		spec     v1.JaegerStorageSpec
		expected bool
	}{
		{
			spec:     v1.JaegerStorageSpec{Type: "googlephotos"},
			expected: false,
		},
		{
			spec:     v1.JaegerStorageSpec{Type: "cassandra"},
			expected: false,
		},
		{
			spec:     v1.JaegerStorageSpec{Type: "elasticsearch"},
			expected: false,
		},
		{
			spec:     v1.JaegerStorageSpec{Type: "elasticsearch", Options: v1.NewOptions(map[string]interface{}{"es.use-aliases": "false"})},
			expected: false,
		},
		{
			spec:     v1.JaegerStorageSpec{Type: "cassandra", Options: v1.NewOptions(map[string]interface{}{"es.use-aliases": "false"})},
			expected: false,
		},
		{
			spec:     v1.JaegerStorageSpec{Type: "elasticsearch", Options: v1.NewOptions(map[string]interface{}{"es.use-aliases": "true"})},
			expected: true,
		},
	}

	for _, test := range tests {
		assert.Equal(t, test.expected, EnableRollover(test.spec))
	}
}

func TestElasticsearchDependencies(t *testing.T) {
	j := v1.NewJaeger(types.NamespacedName{Name: "eevee"})
	j.Namespace = "kitchen"
	j.Spec.Storage.EsRollover.Image = "wohooo"
	j.Spec.Storage.Options = v1.NewOptions(map[string]interface{}{"es.server-urls": "foo,bar", "es.index-prefix": "shortone"})

	deps := elasticsearchDependencies(j)
	assert.Equal(t, 1, len(deps))
	job := deps[0]

	assert.Equal(t, j.Namespace, job.Namespace)
	assert.Equal(t, []metav1.OwnerReference{util.AsOwner(j)}, job.OwnerReferences)
	assert.Equal(t, util.Labels("eevee-es-rollover-create-mapping", "job-es-rollover-create-mapping", *j), job.Labels)
	assert.Equal(t, 1, len(job.Spec.Template.Spec.Containers))
	assert.Equal(t, j.Spec.Storage.EsRollover.Image, job.Spec.Template.Spec.Containers[0].Image)
	assert.Equal(t, []string{"init", "foo"}, job.Spec.Template.Spec.Containers[0].Args)
	assert.Equal(t, []corev1.EnvVar{{Name: "INDEX_PREFIX", Value: "shortone"}}, job.Spec.Template.Spec.Containers[0].Env)
}

func TestEnvVars(t *testing.T) {
	tests := []struct {
		opts     v1.Options
		expected []corev1.EnvVar
	}{
		{},
		{
			opts:     v1.NewOptions(map[string]interface{}{"es.index-prefix": "foo"}),
			expected: []corev1.EnvVar{{Name: "INDEX_PREFIX", Value: "foo"}},
		},
		{
			opts: v1.NewOptions(map[string]interface{}{
				"es.index-prefix": "foo",
				"es.num-shards":   "5",
				"es.num-replicas": "3",
				"es.password":     "nopass",
				"es.username":     "fredy"}),
			expected: []corev1.EnvVar{
				{Name: "INDEX_PREFIX", Value: "foo"},
				{Name: "SHARDS", Value: "5"},
				{Name: "REPLICAS", Value: "3"},
				{Name: "ES_USERNAME", Value: "fredy"},
				{Name: "ES_PASSWORD", Value: "nopass"}},
		},
	}
	for _, test := range tests {
		assert.Equal(t, test.expected, envVars(test.opts))
	}
}
