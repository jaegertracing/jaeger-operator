package cronjob

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"

	v1 "github.com/jaegertracing/jaeger-operator/pkg/apis/jaegertracing/v1"
	"github.com/jaegertracing/jaeger-operator/pkg/util"
)

func TestCreateRollover(t *testing.T) {
	cj := CreateRollover(v1.NewJaeger(types.NamespacedName{Name: "pikachu"}))
	assert.Equal(t, 2, len(cj))
}

func TestRollover(t *testing.T) {
	j := v1.NewJaeger(types.NamespacedName{Name: "eevee"})
	j.Namespace = "kitchen"
	j.Spec.Storage.EsRollover.Image = "wohooo"
	j.Spec.Storage.EsRollover.Conditions = "weheee"
	j.Spec.Storage.Options = v1.NewOptions(map[string]interface{}{"es.server-urls": "foo,bar", "es.index-prefix": "shortone"})

	cjob := rollover(j)
	assert.Equal(t, j.Namespace, cjob.Namespace)
	assert.Equal(t, []metav1.OwnerReference{util.AsOwner(j)}, cjob.OwnerReferences)
	assert.Equal(t, util.Labels("eevee-es-rollover", "cronjob-es-rollover", *j), cjob.Labels)
	assert.Equal(t, 1, len(cjob.Spec.JobTemplate.Spec.Template.Spec.Containers))
	assert.Equal(t, j.Spec.Storage.EsRollover.Image, cjob.Spec.JobTemplate.Spec.Template.Spec.Containers[0].Image)
	assert.Equal(t, []string{"rollover", "foo"}, cjob.Spec.JobTemplate.Spec.Template.Spec.Containers[0].Args)
	assert.Equal(t, []corev1.EnvVar{{Name: "INDEX_PREFIX", Value: "shortone"}, {Name: "CONDITIONS", Value: "weheee"}}, cjob.Spec.JobTemplate.Spec.Template.Spec.Containers[0].Env)
}

func TestLookback(t *testing.T) {
	j := v1.NewJaeger(types.NamespacedName{Name: "squirtle"})
	j.Namespace = "kitchen"
	j.Spec.Storage.EsRollover.Image = "wohooo"
	j.Spec.Storage.EsRollover.ReadTTL = "2h"
	j.Spec.Storage.Options = v1.NewOptions(map[string]interface{}{"es.server-urls": "foo,bar", "es.index-prefix": "shortone"})

	cjob := lookback(j)
	assert.Equal(t, j.Namespace, cjob.Namespace)
	assert.Equal(t, []metav1.OwnerReference{util.AsOwner(j)}, cjob.OwnerReferences)
	assert.Equal(t, util.Labels("squirtle-es-lookback", "cronjob-es-lookback", *j), cjob.Labels)
	assert.Equal(t, 1, len(cjob.Spec.JobTemplate.Spec.Template.Spec.Containers))
	assert.Equal(t, j.Spec.Storage.EsRollover.Image, cjob.Spec.JobTemplate.Spec.Template.Spec.Containers[0].Image)
	assert.Equal(t, []string{"lookback", "foo"}, cjob.Spec.JobTemplate.Spec.Template.Spec.Containers[0].Args)
	assert.Equal(t, []corev1.EnvVar{{Name: "INDEX_PREFIX", Value: "shortone"}, {Name: "UNIT", Value: "hours"}, {Name: "UNIT_COUNT", Value: "2"}}, cjob.Spec.JobTemplate.Spec.Template.Spec.Containers[0].Env)
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

func TestParseUnits(t *testing.T) {
	tests := []struct {
		d     time.Duration
		units pythonUnits
	}{
		{d: time.Second, units: pythonUnits{units: seconds, count: 1}},
		{d: time.Second * 6, units: pythonUnits{units: seconds, count: 6}},
		{d: time.Minute, units: pythonUnits{units: minutes, count: 1}},
		{d: time.Minute * 2, units: pythonUnits{units: minutes, count: 2}},
		{d: time.Hour, units: pythonUnits{units: hours, count: 1}},
		{d: time.Hour * 2, units: pythonUnits{units: hours, count: 2}},
		{d: time.Hour*2 + time.Minute*2, units: pythonUnits{units: minutes, count: 122}},
		{d: time.Hour*2 + time.Minute*2 + time.Second*2, units: pythonUnits{units: seconds, count: 2*60*60 + 2*60 + 2}},
		{d: time.Hour*2 + time.Minute*2 + time.Second*2 + time.Millisecond*8, units: pythonUnits{units: seconds, count: 2*60*60 + 2*60 + 2}},
		{d: time.Millisecond * 8, units: pythonUnits{units: seconds, count: 0}},
		{d: time.Minute * 60, units: pythonUnits{units: hours, count: 1}},
	}
	for _, test := range tests {
		assert.Equal(t, test.units, parseToUnits(test.d))
	}
}
