package cronjob

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"k8s.io/apimachinery/pkg/types"

	v1 "github.com/jaegertracing/jaeger-operator/pkg/apis/jaegertracing/v1"
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
