package cronjob

import (
	"testing"

	"github.com/jaegertracing/jaeger-operator/pkg/version"

	"github.com/spf13/viper"
	"github.com/stretchr/testify/assert"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	"k8s.io/apimachinery/pkg/types"

	v1 "github.com/jaegertracing/jaeger-operator/pkg/apis/jaegertracing/v1"
)

func TestCreateEsIndexCleaner(t *testing.T) {
	jaeger := &v1.Jaeger{Spec: v1.JaegerSpec{Storage: v1.JaegerStorageSpec{Options: v1.NewOptions(
		map[string]interface{}{"es.index-prefix": "tenant1", "es.server-urls": "http://nowhere:666,foo"})}}}
	days := 0
	jaeger.Spec.Storage.EsIndexCleaner.NumberOfDays = &days
	historyLimits := int32(1)
	jaeger.Spec.Storage.EsIndexCleaner.SuccessfulJobsHistoryLimit = &historyLimits
	cronJob := CreateEsIndexCleaner(jaeger)
	assert.Equal(t, 2, len(cronJob.Spec.JobTemplate.Spec.Template.Spec.Containers[0].Args))
	// default number of days (7) is applied in normalize in controller
	assert.Equal(t, []string{"0", "http://nowhere:666"}, cronJob.Spec.JobTemplate.Spec.Template.Spec.Containers[0].Args)
	assert.Equal(t, 1, len(cronJob.Spec.JobTemplate.Spec.Template.Spec.Containers[0].Env))
	assert.Equal(t, "INDEX_PREFIX", cronJob.Spec.JobTemplate.Spec.Template.Spec.Containers[0].Env[0].Name)
	assert.Equal(t, "tenant1", cronJob.Spec.JobTemplate.Spec.Template.Spec.Containers[0].Env[0].Value)
	assert.Equal(t, historyLimits, *cronJob.Spec.SuccessfulJobsHistoryLimit)
}

func TestEsIndexCleanerSecrets(t *testing.T) {
	jaeger := v1.NewJaeger(types.NamespacedName{Name: "TestEsIndexCleanerSecrets"})
	secret := "mysecret"
	jaeger.Spec.Storage.SecretName = secret

	days := 0
	jaeger.Spec.Storage.EsIndexCleaner.NumberOfDays = &days
	historyLimits := int32(0)
	jaeger.Spec.Storage.EsIndexCleaner.SuccessfulJobsHistoryLimit = &historyLimits
	cronJob := CreateEsIndexCleaner(jaeger)
	assert.Equal(t, secret, cronJob.Spec.JobTemplate.Spec.Template.Spec.Containers[0].EnvFrom[0].SecretRef.LocalObjectReference.Name)
	assert.Equal(t, historyLimits, *cronJob.Spec.SuccessfulJobsHistoryLimit)
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

func TestEsIndexCleanerAnnotations(t *testing.T) {
	jaeger := v1.NewJaeger(types.NamespacedName{Name: "TestEsIndexCleanerAnnotations"})
	jaeger.Spec.Annotations = map[string]string{
		"name":  "operator",
		"hello": "jaeger",
	}
	jaeger.Spec.Storage.EsIndexCleaner.Annotations = map[string]string{
		"hello":                "world", // Override top level annotation
		"prometheus.io/scrape": "false", // Override implicit value
	}

	days := 0
	jaeger.Spec.Storage.EsIndexCleaner.NumberOfDays = &days

	cjob := CreateEsIndexCleaner(jaeger)

	assert.Equal(t, "operator", cjob.Spec.JobTemplate.Spec.Template.Annotations["name"])
	assert.Equal(t, "false", cjob.Spec.JobTemplate.Spec.Template.Annotations["sidecar.istio.io/inject"])
	assert.Equal(t, "world", cjob.Spec.JobTemplate.Spec.Template.Annotations["hello"])
	assert.Equal(t, "false", cjob.Spec.JobTemplate.Spec.Template.Annotations["prometheus.io/scrape"])
	assert.Equal(t, "disabled", cjob.Spec.JobTemplate.Spec.Template.Annotations["linkerd.io/inject"])
}

func TestEsIndexCleanerLabels(t *testing.T) {
	jaeger := v1.NewJaeger(types.NamespacedName{Name: "TestEsIndexCleanerLabels"})
	jaeger.Spec.Labels = map[string]string{
		"name":  "operator",
		"hello": "jaeger",
	}
	jaeger.Spec.Storage.EsIndexCleaner.Labels = map[string]string{
		"hello":   "world", // Override top level label
		"another": "false",
	}

	days := 0
	jaeger.Spec.Storage.EsIndexCleaner.NumberOfDays = &days

	cjob := CreateEsIndexCleaner(jaeger)

	assert.Equal(t, "operator", cjob.Spec.JobTemplate.Spec.Template.Labels["name"])
	assert.Equal(t, "world", cjob.Spec.JobTemplate.Spec.Template.Labels["hello"])
	assert.Equal(t, "false", cjob.Spec.JobTemplate.Spec.Template.Labels["another"])

	// Check if the labels of cronjob pod template equal to the labels of cronjob.
	assert.Equal(t, cjob.ObjectMeta.Labels, cjob.Spec.JobTemplate.Spec.Template.ObjectMeta.Labels)
}

func TestEsIndexCleanerResources(t *testing.T) {

	parentResources := corev1.ResourceRequirements{
		Limits: corev1.ResourceList{
			corev1.ResourceLimitsCPU:              *resource.NewQuantity(1024, resource.BinarySI),
			corev1.ResourceLimitsEphemeralStorage: *resource.NewQuantity(512, resource.DecimalSI),
		},
		Requests: corev1.ResourceList{
			corev1.ResourceRequestsCPU:              *resource.NewQuantity(1024, resource.BinarySI),
			corev1.ResourceRequestsEphemeralStorage: *resource.NewQuantity(512, resource.DecimalSI),
		},
	}

	childResources := corev1.ResourceRequirements{
		Limits: corev1.ResourceList{
			corev1.ResourceLimitsCPU:              *resource.NewQuantity(2048, resource.BinarySI),
			corev1.ResourceLimitsEphemeralStorage: *resource.NewQuantity(1024, resource.DecimalSI),
		},
		Requests: corev1.ResourceList{
			corev1.ResourceRequestsCPU:              *resource.NewQuantity(2048, resource.BinarySI),
			corev1.ResourceRequestsEphemeralStorage: *resource.NewQuantity(1024, resource.DecimalSI),
		},
	}

	days := 0

	tests := []struct {
		jaeger   *v1.Jaeger
		expected corev1.ResourceRequirements
	}{
		{
			jaeger:   &v1.Jaeger{Spec: v1.JaegerSpec{Storage: v1.JaegerStorageSpec{Type: "elasticsearch"}}},
			expected: corev1.ResourceRequirements{},
		},
		{
			jaeger: &v1.Jaeger{Spec: v1.JaegerSpec{
				Storage: v1.JaegerStorageSpec{Type: "elasticsearch"},
				JaegerCommonSpec: v1.JaegerCommonSpec{
					Resources: parentResources,
				},
			}},
			expected: parentResources,
		},
		{
			jaeger: &v1.Jaeger{Spec: v1.JaegerSpec{
				Storage: v1.JaegerStorageSpec{
					Type: "elasticsearch",
					EsIndexCleaner: v1.JaegerEsIndexCleanerSpec{
						JaegerCommonSpec: v1.JaegerCommonSpec{
							Resources: childResources,
						},
					},
				},
				JaegerCommonSpec: v1.JaegerCommonSpec{
					Resources: parentResources,
				},
			}},
			expected: childResources,
		},
	}
	for _, test := range tests {
		test.jaeger.Spec.Storage.EsIndexCleaner.NumberOfDays = &days

		cjob := CreateEsIndexCleaner(test.jaeger)
		assert.Equal(t, test.expected, cjob.Spec.JobTemplate.Spec.Template.Spec.Containers[0].Resources)

	}
}

func TestDefaultEsIndexCleanerImage(t *testing.T) {
	viper.SetDefault("jaeger-es-index-cleaner-image", "jaegertracing/jaeger-es-index-cleaner")

	days := 0

	jaeger := v1.NewJaeger(types.NamespacedName{Name: "TestDefaultEsIndexCleanerImage"})
	jaeger.Spec.Storage.EsIndexCleaner.NumberOfDays = &days

	cjob := CreateEsIndexCleaner(jaeger)
	assert.Empty(t, jaeger.Spec.Storage.EsIndexCleaner.Image)
	assert.Equal(t, "jaegertracing/jaeger-es-index-cleaner:"+version.Get().Jaeger, cjob.Spec.JobTemplate.Spec.Template.Spec.Containers[0].Image)
}

func TestCustomEsIndexCleanerImage(t *testing.T) {
	viper.Set("jaeger-es-index-cleaner-image", "org/custom-es-index-cleaner-image")
	defer viper.Reset()

	days := 0

	jaeger := v1.NewJaeger(types.NamespacedName{Name: "TestDefaultEsIndexCleanerImage"})
	jaeger.Spec.Storage.EsIndexCleaner.NumberOfDays = &days

	cjob := CreateEsIndexCleaner(jaeger)
	assert.Empty(t, jaeger.Spec.Storage.EsIndexCleaner.Image)
	assert.Equal(t, "org/custom-es-index-cleaner-image:"+version.Get().Jaeger, cjob.Spec.JobTemplate.Spec.Template.Spec.Containers[0].Image)
}
