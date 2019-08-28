package cronjob

import (
	"testing"

	"github.com/stretchr/testify/assert"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"

	"github.com/jaegertracing/jaeger-operator/pkg/apis/jaegertracing/v1"
)

func TestCreateEsIndexCleaner(t *testing.T) {
	jaeger := &v1.Jaeger{Spec: v1.JaegerSpec{Storage: v1.JaegerStorageSpec{Options: v1.NewOptions(
		map[string]interface{}{"es.index-prefix": "tenant1", "es.server-urls": "http://nowhere:666,foo"})}}}
	days := 0
	jaeger.Spec.Storage.EsIndexCleaner.NumberOfDays = &days
	cronJob := CreateEsIndexCleaner(jaeger)
	assert.Equal(t, 2, len(cronJob.Spec.JobTemplate.Spec.Template.Spec.Containers[0].Args))
	// default number of days (7) is applied in normalize in controller
	assert.Equal(t, []string{"0", "http://nowhere:666"}, cronJob.Spec.JobTemplate.Spec.Template.Spec.Containers[0].Args)
	assert.Equal(t, 1, len(cronJob.Spec.JobTemplate.Spec.Template.Spec.Containers[0].Env))
	assert.Equal(t, "INDEX_PREFIX", cronJob.Spec.JobTemplate.Spec.Template.Spec.Containers[0].Env[0].Name)
	assert.Equal(t, "tenant1", cronJob.Spec.JobTemplate.Spec.Template.Spec.Containers[0].Env[0].Value)
}

func TestEsIndexCleanerSecrets(t *testing.T) {
	jaeger := v1.NewJaeger(types.NamespacedName{Name: "TestEsIndexCleanerSecrets"})
	secret := "mysecret"
	jaeger.Spec.Storage.SecretName = secret

	days := 0
	jaeger.Spec.Storage.EsIndexCleaner.NumberOfDays = &days
	cronJob := CreateEsIndexCleaner(jaeger)
	assert.Equal(t, secret, cronJob.Spec.JobTemplate.Spec.Template.Spec.Containers[0].EnvFrom[0].SecretRef.LocalObjectReference.Name)
}

func TestEsIndexCleanerEnvVars(t *testing.T) {
	tests := []struct {
		opts map[string]interface{}
		envs []corev1.EnvVar
	}{
		{
			// empty options do not add any env vars
		},
		{
			opts: map[string]interface{}{"es.index-prefix": "foo", "es.username": "joe", "es.password": "pass"},
			envs: []corev1.EnvVar{{Name: "INDEX_PREFIX", Value: "foo"}, {Name: "ES_USERNAME", Value: "joe"}, {Name: "ES_PASSWORD", Value: "pass"}},
		},
		{
			opts: map[string]interface{}{"es.index-prefix": "foo", "es.username": "joe", "es.password": "pass", "es.use-aliases": "false"},
			envs: []corev1.EnvVar{{Name: "INDEX_PREFIX", Value: "foo"}, {Name: "ES_USERNAME", Value: "joe"}, {Name: "ES_PASSWORD", Value: "pass"}},
		},
		{
			opts: map[string]interface{}{"es.index-prefix": "foo", "es.username": "joe", "es.password": "pass", "es.use-aliases": "true"},
			envs: []corev1.EnvVar{{Name: "INDEX_PREFIX", Value: "foo"}, {Name: "ES_USERNAME", Value: "joe"}, {Name: "ES_PASSWORD", Value: "pass"}, {Name: "ROLLOVER", Value: "true"}},
		},
		{
			opts: map[string]interface{}{"es.index-prefix": "foo", "es.username": "joe", "es.password": "pass", "es.use-aliases": "true", "es.tls": "true"},
			envs: []corev1.EnvVar{{Name: "INDEX_PREFIX", Value: "foo"}, {Name: "ES_USERNAME", Value: "joe"}, {Name: "ES_PASSWORD", Value: "pass"},
				{Name: "ES_TLS", Value: "true"}, {Name: "ROLLOVER", Value: "true"}},
		},
		{
			opts: map[string]interface{}{"es.index-prefix": "foo", "es.username": "joe", "es.password": "pass", "es.use-aliases": "true", "es.tls": "true", "es.tls.ca": "foo/bar/ca.crt"},
			envs: []corev1.EnvVar{{Name: "INDEX_PREFIX", Value: "foo"}, {Name: "ES_USERNAME", Value: "joe"}, {Name: "ES_PASSWORD", Value: "pass"},
				{Name: "ES_TLS", Value: "true"}, {Name: "ES_TLS_CA", Value: "foo/bar/ca.crt"}, {Name: "ROLLOVER", Value: "true"}},
		},
		{
			opts: map[string]interface{}{"es.index-prefix": "foo", "es.username": "joe", "es.password": "pass", "es.use-aliases": "true", "es.tls": "true",
				"es.tls.ca": "foo/bar/ca.crt", "es.tls.cert": "foo/bar/cert.crt"},
			envs: []corev1.EnvVar{{Name: "INDEX_PREFIX", Value: "foo"}, {Name: "ES_USERNAME", Value: "joe"}, {Name: "ES_PASSWORD", Value: "pass"},
				{Name: "ES_TLS", Value: "true"}, {Name: "ES_TLS_CA", Value: "foo/bar/ca.crt"}, {Name: "ES_TLS_CERT", Value: "foo/bar/cert.crt"}, {Name: "ROLLOVER", Value: "true"}},
		},
		{
			opts: map[string]interface{}{"es.index-prefix": "foo", "es.username": "joe", "es.password": "pass", "es.use-aliases": "true", "es.tls": "true",
				"es.tls.ca": "foo/bar/ca.crt", "es.tls.cert": "foo/bar/cert.crt", "es.tls.key": "foo/bar/cert.key"},
			envs: []corev1.EnvVar{{Name: "INDEX_PREFIX", Value: "foo"}, {Name: "ES_USERNAME", Value: "joe"}, {Name: "ES_PASSWORD", Value: "pass"},
				{Name: "ES_TLS", Value: "true"}, {Name: "ES_TLS_CA", Value: "foo/bar/ca.crt"}, {Name: "ES_TLS_CERT", Value: "foo/bar/cert.crt"},
				{Name: "ES_TLS_KEY", Value: "foo/bar/cert.key"}, {Name: "ROLLOVER", Value: "true"}},
		},
	}

	for _, test := range tests {
		jaeger := v1.NewJaeger(types.NamespacedName{Name: "TestEsIndexCleanerSecrets"})
		days := 0
		jaeger.Spec.Storage.EsIndexCleaner.NumberOfDays = &days
		jaeger.Spec.Storage.Options = v1.NewOptions(test.opts)
		cronJob := CreateEsIndexCleaner(jaeger)
		assert.Equal(t, test.envs, cronJob.Spec.JobTemplate.Spec.Template.Spec.Containers[0].Env)
	}
}
