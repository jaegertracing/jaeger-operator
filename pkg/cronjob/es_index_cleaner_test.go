package cronjob

import (
	"testing"

	"github.com/jaegertracing/jaeger-operator/pkg/version"

	"github.com/spf13/viper"
	"github.com/stretchr/testify/assert"

	batchv1 "k8s.io/api/batch/v1"
	batchv1beta1 "k8s.io/api/batch/v1beta1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	"k8s.io/apimachinery/pkg/types"

	v1 "github.com/jaegertracing/jaeger-operator/apis/v1"
)

func init() {
	// Always test with v1.  It is available at compile time and is exactly the same as v1beta1
	viper.SetDefault(v1.FlagCronJobsVersion, v1.FlagCronJobsVersionBatchV1)
}

func TestCreateEsIndexCleaner(t *testing.T) {
	jaeger := &v1.Jaeger{Spec: v1.JaegerSpec{Storage: v1.JaegerStorageSpec{Options: v1.NewOptions(
		map[string]interface{}{"es.index-prefix": "tenant1", "es.server-urls": "http://nowhere:666,foo"})}}}
	days := 0
	jaeger.Spec.Storage.EsIndexCleaner.NumberOfDays = &days
	historyLimits := int32(1)
	jaeger.Spec.Storage.EsIndexCleaner.SuccessfulJobsHistoryLimit = &historyLimits
	cronJob := CreateEsIndexCleaner(jaeger).(*batchv1.CronJob)

	assert.Len(t, cronJob.Spec.JobTemplate.Spec.Template.Spec.Containers[0].Args, 2)
	// default number of days (7) is applied in normalize in controller
	assert.Equal(t, []string{"0", "http://nowhere:666"}, cronJob.Spec.JobTemplate.Spec.Template.Spec.Containers[0].Args)
	assert.Len(t, cronJob.Spec.JobTemplate.Spec.Template.Spec.Containers[0].Env, 1)
	assert.Equal(t, "INDEX_PREFIX", cronJob.Spec.JobTemplate.Spec.Template.Spec.Containers[0].Env[0].Name)
	assert.Equal(t, "tenant1", cronJob.Spec.JobTemplate.Spec.Template.Spec.Containers[0].Env[0].Value)
	assert.Equal(t, historyLimits, *cronJob.Spec.SuccessfulJobsHistoryLimit)
}

func TestCreateEsIndexCleanerTypeMeta(t *testing.T) {
	testData := []struct {
		Name string
		flag string
	}{
		{Name: "Test batch/v1beta1", flag: v1.FlagCronJobsVersionBatchV1Beta1},
		{Name: "Test batch/v1", flag: v1.FlagCronJobsVersionBatchV1},
	}

	jaeger := &v1.Jaeger{Spec: v1.JaegerSpec{Storage: v1.JaegerStorageSpec{Options: v1.NewOptions(
		map[string]interface{}{"es.index-prefix": "tenant1", "es.server-urls": "http://nowhere:666,foo"})}}}
	days := 0
	jaeger.Spec.Storage.EsIndexCleaner.NumberOfDays = &days
	historyLimits := int32(1)
	jaeger.Spec.Storage.EsIndexCleaner.SuccessfulJobsHistoryLimit = &historyLimits
	for _, td := range testData {
		if td.flag == v1.FlagCronJobsVersionBatchV1Beta1 {
			viper.SetDefault(v1.FlagCronJobsVersion, v1.FlagCronJobsVersionBatchV1Beta1)
		}
		cronJobs := CreateEsIndexCleaner(jaeger)
		switch tt := cronJobs.(type) {
		case *batchv1beta1.CronJob:
			assert.Equal(t, "CronJob", tt.Kind)
			assert.Equal(t, v1.FlagCronJobsVersionBatchV1Beta1, tt.APIVersion)
			viper.SetDefault(v1.FlagCronJobsVersion, v1.FlagCronJobsVersionBatchV1)
		case *batchv1.CronJob:
			assert.Equal(t, "CronJob", tt.Kind)
			assert.Equal(t, v1.FlagCronJobsVersionBatchV1, tt.APIVersion)
		}
	}
}

func TestEsIndexCleanerSecrets(t *testing.T) {
	jaeger := v1.NewJaeger(types.NamespacedName{Name: "TestEsIndexCleanerSecrets"})
	secret := "mysecret"
	jaeger.Spec.Storage.SecretName = secret

	days := 0
	jaeger.Spec.Storage.EsIndexCleaner.NumberOfDays = &days
	historyLimits := int32(0)
	jaeger.Spec.Storage.EsIndexCleaner.SuccessfulJobsHistoryLimit = &historyLimits

	cronJob := CreateEsIndexCleaner(jaeger).(*batchv1.CronJob)
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
		{
			opts: map[string]interface{}{"es.index-prefix": "foo", "es.index-date-separator": ".", "es.username": "joe", "es.password": "pass"},
			envs: []corev1.EnvVar{{Name: "INDEX_PREFIX", Value: "foo"}, {Name: "INDEX_DATE_SEPARATOR", Value: "."}, {Name: "ES_USERNAME", Value: "joe"}, {Name: "ES_PASSWORD", Value: "pass"}},
		},
	}

	for _, test := range tests {
		jaeger := v1.NewJaeger(types.NamespacedName{Name: "TestEsIndexCleanerSecrets"})
		days := 0
		jaeger.Spec.Storage.EsIndexCleaner.NumberOfDays = &days
		jaeger.Spec.Storage.Options = v1.NewOptions(test.opts)
		cronJob := CreateEsIndexCleaner(jaeger).(*batchv1.CronJob)
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

	cjob := CreateEsIndexCleaner(jaeger).(*batchv1.CronJob)
	assert.Equal(t, "operator", cjob.Spec.JobTemplate.Spec.Template.Annotations["name"])
	assert.Equal(t, "false", cjob.Spec.JobTemplate.Spec.Template.Annotations["sidecar.istio.io/inject"])
	assert.Equal(t, "world", cjob.Spec.JobTemplate.Spec.Template.Annotations["hello"])
	assert.Equal(t, "false", cjob.Spec.JobTemplate.Spec.Template.Annotations["prometheus.io/scrape"])
	assert.Equal(t, "disabled", cjob.Spec.JobTemplate.Spec.Template.Annotations["linkerd.io/inject"])
}

func TestEsIndexCleanerBackoffLimit(t *testing.T) {
	jaeger := v1.NewJaeger(types.NamespacedName{Name: "TestEsIndexCleanerAnnotations"})

	BackoffLimit := int32(3)
	jaeger.Spec.Storage.EsIndexCleaner.BackoffLimit = &BackoffLimit

	days := 0
	jaeger.Spec.Storage.EsIndexCleaner.NumberOfDays = &days

	cjob := CreateEsIndexCleaner(jaeger).(*batchv1.CronJob)
	assert.Equal(t, &BackoffLimit, cjob.Spec.JobTemplate.Spec.BackoffLimit)
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

	cjob := CreateEsIndexCleaner(jaeger).(*batchv1.CronJob)

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
			jaeger:   &v1.Jaeger{Spec: v1.JaegerSpec{Storage: v1.JaegerStorageSpec{Type: v1.JaegerESStorage}}},
			expected: corev1.ResourceRequirements{},
		},
		{
			jaeger: &v1.Jaeger{Spec: v1.JaegerSpec{
				Storage: v1.JaegerStorageSpec{Type: v1.JaegerESStorage},
				JaegerCommonSpec: v1.JaegerCommonSpec{
					Resources: parentResources,
				},
			}},
			expected: parentResources,
		},
		{
			jaeger: &v1.Jaeger{Spec: v1.JaegerSpec{
				Storage: v1.JaegerStorageSpec{
					Type: v1.JaegerESStorage,
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

		cjob := CreateEsIndexCleaner(test.jaeger).(*batchv1.CronJob)
		assert.Equal(t, test.expected, cjob.Spec.JobTemplate.Spec.Template.Spec.Containers[0].Resources)

	}
}

func TestDefaultEsIndexCleanerImage(t *testing.T) {
	viper.SetDefault("jaeger-es-index-cleaner-image", "jaegertracing/jaeger-es-index-cleaner")

	days := 0

	jaeger := v1.NewJaeger(types.NamespacedName{Name: "TestDefaultEsIndexCleanerImage"})
	jaeger.Spec.Storage.EsIndexCleaner.NumberOfDays = &days

	cjob := CreateEsIndexCleaner(jaeger).(*batchv1.CronJob)
	assert.Empty(t, jaeger.Spec.Storage.EsIndexCleaner.Image)
	assert.Equal(t, "jaegertracing/jaeger-es-index-cleaner:"+version.Get().Jaeger, cjob.Spec.JobTemplate.Spec.Template.Spec.Containers[0].Image)
}

func TestCustomEsIndexCleanerImage(t *testing.T) {
	viper.Set("jaeger-es-index-cleaner-image", "org/custom-es-index-cleaner-image")
	defer viper.Reset()

	days := 0

	jaeger := v1.NewJaeger(types.NamespacedName{Name: "TestDefaultEsIndexCleanerImage"})
	jaeger.Spec.Storage.EsIndexCleaner.NumberOfDays = &days

	cjob := CreateEsIndexCleaner(jaeger).(*batchv1.CronJob)
	assert.Empty(t, jaeger.Spec.Storage.EsIndexCleaner.Image)
	assert.Equal(t, "org/custom-es-index-cleaner-image:"+version.Get().Jaeger, cjob.Spec.JobTemplate.Spec.Template.Spec.Containers[0].Image)
}

// Test Case for PriorityClassName
func TestPriorityClassName(t *testing.T) {
	jaeger := v1.NewJaeger(types.NamespacedName{Name: "TestPriorityClassName"})

	priorityClassNameVal := ""

	assert.Equal(t, priorityClassNameVal, jaeger.Spec.Storage.EsIndexCleaner.PriorityClassName)
}

func TestEsIndexCleanerImagePullSecrets(t *testing.T) {
	jaeger := v1.NewJaeger(types.NamespacedName{Name: "TestEsIndexCleanerImagePullSecrets"})
	days := 0
	jaeger.Spec.Storage.EsIndexCleaner.NumberOfDays = &days

	const pullSecret = "mysecret"
	jaeger.Spec.ImagePullSecrets = []corev1.LocalObjectReference{
		{
			Name: pullSecret,
		},
	}

	esIndexCleaner := CreateEsIndexCleaner(jaeger).(*batchv1.CronJob)

	assert.Equal(t, pullSecret, esIndexCleaner.Spec.JobTemplate.Spec.Template.Spec.ImagePullSecrets[0].Name)
}

func TestEsIndexCleanerImagePullPolicy(t *testing.T) {
	jaeger := v1.NewJaeger(types.NamespacedName{Name: "TestEsIndexCleanerImagePullPolicy"})
	days := 0
	jaeger.Spec.Storage.EsIndexCleaner.NumberOfDays = &days

	const ImagePullPolicy = corev1.PullPolicy("Always")
	jaeger.Spec.Storage.EsIndexCleaner.NumberOfDays = &days
	jaeger.Spec.Storage.EsIndexCleaner.ImagePullPolicy = corev1.PullPolicy("Always")

	esIndexCleaner := CreateEsIndexCleaner(jaeger).(*batchv1.CronJob)

	assert.Equal(t, ImagePullPolicy, esIndexCleaner.Spec.JobTemplate.Spec.Template.Spec.Containers[0].ImagePullPolicy)
}

func TestEsIndexCleaneContainerSecurityContext(t *testing.T) {
	jaeger := v1.NewJaeger(types.NamespacedName{Name: "TestEsIndexCleanerContainerSecurityContext"})
	days := 0
	jaeger.Spec.Storage.EsIndexCleaner.NumberOfDays = &days

	true := true
	ContainerSecurityContext := &corev1.SecurityContext{
		ReadOnlyRootFilesystem: &true,
	}
	jaeger.Spec.Storage.EsIndexCleaner.NumberOfDays = &days
	jaeger.Spec.Storage.EsIndexCleaner.ContainerSecurityContext = ContainerSecurityContext

	esIndexCleaner := CreateEsIndexCleaner(jaeger).(*batchv1.CronJob)

	assert.Equal(t, ContainerSecurityContext, esIndexCleaner.Spec.JobTemplate.Spec.Template.Spec.Containers[0].SecurityContext)
}
