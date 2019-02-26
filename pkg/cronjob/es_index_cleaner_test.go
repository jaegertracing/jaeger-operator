package cronjob

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/jaegertracing/jaeger-operator/pkg/apis/io/v1alpha1"
)

func TestGetEsHostname(t *testing.T) {
	tests := []struct {
		underTest map[string]string
		hostname  string
	}{
		{hostname: ""},
		{underTest: map[string]string{"": ""}, hostname: ""},
		{underTest: map[string]string{"es.server-urls": "goo:tar"}, hostname: "goo:tar"},
		{underTest: map[string]string{"es.server-urls": "http://es:9000,https://es2:9200"}, hostname: "http://es:9000"},
	}
	for _, test := range tests {
		assert.Equal(t, test.hostname, getEsHostname(test.underTest))
	}
}

func TestCreateEsIndexCleaner(t *testing.T) {
	jaeger := &v1alpha1.Jaeger{Spec: v1alpha1.JaegerSpec{Storage: v1alpha1.JaegerStorageSpec{Options: v1alpha1.NewOptions(
		map[string]interface{}{"es.index-prefix": "tenant1", "es.server-urls": "http://nowhere:666,foo"})}}}
	cronJob := CreateEsIndexCleaner(jaeger)
	assert.Equal(t, 2, len(cronJob.Spec.JobTemplate.Spec.Template.Spec.Containers[0].Args))
	// default number of days (7) is applied in normalize in controller
	assert.Equal(t, []string{"0", "http://nowhere:666"}, cronJob.Spec.JobTemplate.Spec.Template.Spec.Containers[0].Args)
	assert.Equal(t, 1, len(cronJob.Spec.JobTemplate.Spec.Template.Spec.Containers[0].Env))
	assert.Equal(t, "INDEX_PREFIX", cronJob.Spec.JobTemplate.Spec.Template.Spec.Containers[0].Env[0].Name)
	assert.Equal(t, "tenant1", cronJob.Spec.JobTemplate.Spec.Template.Spec.Containers[0].Env[0].Value)
}

func TestEsIndexCleanerSecrets(t *testing.T) {
	jaeger := v1alpha1.NewJaeger("TestEsIndexCleanerSecrets")
	secret := "mysecret"
	jaeger.Spec.Storage.SecretName = secret

	cronJob := CreateEsIndexCleaner(jaeger)
	assert.Equal(t, secret, cronJob.Spec.JobTemplate.Spec.Template.Spec.Containers[0].EnvFrom[0].SecretRef.LocalObjectReference.Name)
}
