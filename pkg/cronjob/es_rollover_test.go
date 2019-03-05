package cronjob

import (
	"testing"

	"github.com/stretchr/testify/assert"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/jaegertracing/jaeger-operator/pkg/apis/jaegertracing/v1"
	"github.com/jaegertracing/jaeger-operator/pkg/util"
)

func TestCreateRollover(t *testing.T) {
	cj := CreateRollover(v1.NewJaeger("pikachu"))
	assert.Equal(t, 2, len(cj))
}

func TestRollover(t *testing.T) {
	j := v1.NewJaeger("eevee")
	j.Namespace = "kitchen"
	j.Spec.Storage.Rollover.Image = "wohooo"
	j.Spec.Storage.Rollover.Conditions = "weheee"
	j.Spec.Storage.Options = v1.NewOptions(map[string]interface{}{"es.server-urls": "foo,bar", "es.index-prefix": "shortone"})

	cjob := rollover(j)
	assert.Equal(t, j.Namespace, cjob.Namespace)
	assert.Equal(t, []metav1.OwnerReference{util.AsOwner(j)}, cjob.OwnerReferences)
	assert.Equal(t, util.Labels("eevee-es-rollover", "cronjob-es-rollover", *j), cjob.Labels)
	assert.Equal(t, 1, len(cjob.Spec.JobTemplate.Spec.Template.Spec.Containers))
	assert.Equal(t, j.Spec.Storage.Rollover.Image, cjob.Spec.JobTemplate.Spec.Template.Spec.Containers[0].Image)
	assert.Equal(t, []string{"rollover", "foo"}, cjob.Spec.JobTemplate.Spec.Template.Spec.Containers[0].Args)
	assert.Equal(t, []corev1.EnvVar{{Name: "INDEX_PREFIX", Value: "shortone"}, {Name: "CONDITIONS", Value: "weheee"}}, cjob.Spec.JobTemplate.Spec.Template.Spec.Containers[0].Env)
}

func TestLookback(t *testing.T) {
	j := v1.NewJaeger("squirtle")
	j.Namespace = "kitchen"
	j.Spec.Storage.Rollover.Image = "wohooo"
	unitCount := 7
	j.Spec.Storage.Rollover.UnitCount = &unitCount
	j.Spec.Storage.Rollover.Unit = "minutes"
	j.Spec.Storage.Options = v1.NewOptions(map[string]interface{}{"es.server-urls": "foo,bar", "es.index-prefix": "shortone"})

	cjob := lookback(j)
	assert.Equal(t, j.Namespace, cjob.Namespace)
	assert.Equal(t, []metav1.OwnerReference{util.AsOwner(j)}, cjob.OwnerReferences)
	assert.Equal(t, util.Labels("squirtle-es-lookback", "cronjob-es-lookback", *j), cjob.Labels)
	assert.Equal(t, 1, len(cjob.Spec.JobTemplate.Spec.Template.Spec.Containers))
	assert.Equal(t, j.Spec.Storage.Rollover.Image, cjob.Spec.JobTemplate.Spec.Template.Spec.Containers[0].Image)
	assert.Equal(t, []string{"lookback", "foo"}, cjob.Spec.JobTemplate.Spec.Template.Spec.Containers[0].Args)
	assert.Equal(t, []corev1.EnvVar{{Name: "INDEX_PREFIX", Value: "shortone"}, {Name: "UNIT", Value: "minutes"}, {Name: "UNIT_COUNT", Value: "7"}}, cjob.Spec.JobTemplate.Spec.Template.Spec.Containers[0].Env)
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
				"es.password":     "nopass",
				"es.username":     "fredy"}),
			expected: []corev1.EnvVar{
				{Name: "INDEX_PREFIX", Value: "foo"},
				{Name: "ES_USERNAME", Value: "fredy"},
				{Name: "ES_PASSWORD", Value: "nopass"}},
		},
	}
	for _, test := range tests {
		assert.Equal(t, test.expected, esScriptEnvVars(test.opts))
	}
}
