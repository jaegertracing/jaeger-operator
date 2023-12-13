package cronjob

import (
	"testing"
	"time"

	batchv1 "k8s.io/api/batch/v1"
	batchv1beta1 "k8s.io/api/batch/v1beta1"

	"github.com/jaegertracing/jaeger-operator/pkg/autodetect"
	"github.com/jaegertracing/jaeger-operator/pkg/version"

	"github.com/spf13/viper"
	"github.com/stretchr/testify/assert"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"

	v1 "github.com/jaegertracing/jaeger-operator/apis/v1"
	"github.com/jaegertracing/jaeger-operator/pkg/util"
)

func init() {
	// Always test with v1.  It is available at compile time and is exactly the same as v1beta1
	viper.SetDefault(v1.FlagCronJobsVersion, v1.FlagCronJobsVersionBatchV1)
}

func TestCreateRollover(t *testing.T) {
	cj := CreateRollover(v1.NewJaeger(types.NamespacedName{Name: "pikachu"}))
	assert.Len(t, cj, 2)
}

func TestCreateRolloverTypeMeta(t *testing.T) {
	testData := []struct {
		Name string
		flag string
	}{
		{Name: "Test batch/v1beta1", flag: v1.FlagCronJobsVersionBatchV1Beta1},
		{Name: "Test batch/v1", flag: v1.FlagCronJobsVersionBatchV1},
	}
	for _, td := range testData {
		if td.flag == v1.FlagCronJobsVersionBatchV1Beta1 {
			viper.SetDefault(v1.FlagCronJobsVersion, v1.FlagCronJobsVersionBatchV1Beta1)
		}
		cjs := CreateRollover(v1.NewJaeger(types.NamespacedName{Name: "pikachu"}))
		assert.Len(t, cjs, 2)
		for _, cj := range cjs {
			switch tt := cj.(type) {
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
}

func TestRollover(t *testing.T) {
	j := v1.NewJaeger(types.NamespacedName{Name: "eevee"})
	j.Namespace = "kitchen"
	j.Spec.Storage.EsRollover.Image = "wohooo"
	j.Spec.Storage.EsRollover.Conditions = "weheee"
	j.Spec.Storage.Options = v1.NewOptions(map[string]interface{}{"es.server-urls": "foo,bar", "es.index-prefix": "shortone"})
	historyLimits := int32(2)
	j.Spec.Storage.EsRollover.SuccessfulJobsHistoryLimit = &historyLimits

	cjob := rollover(j).(*batchv1.CronJob)
	assert.Equal(t, j.Namespace, cjob.Namespace)
	assert.Equal(t, []metav1.OwnerReference{util.AsOwner(j)}, cjob.OwnerReferences)
	assert.Equal(t, util.Labels("eevee-es-rollover", "cronjob-es-rollover", *j), cjob.Labels)
	assert.Len(t, cjob.Spec.JobTemplate.Spec.Template.Spec.Containers, 1)
	assert.Equal(t, j.Spec.Storage.EsRollover.Image, cjob.Spec.JobTemplate.Spec.Template.Spec.Containers[0].Image)
	assert.Equal(t, []string{"rollover", "foo"}, cjob.Spec.JobTemplate.Spec.Template.Spec.Containers[0].Args)
	assert.Equal(t, []corev1.EnvVar{{Name: "INDEX_PREFIX", Value: "shortone"}, {Name: "CONDITIONS", Value: "weheee"}}, cjob.Spec.JobTemplate.Spec.Template.Spec.Containers[0].Env)
	assert.Equal(t, historyLimits, *cjob.Spec.SuccessfulJobsHistoryLimit)

	// Test openshift settings
	autodetect.OperatorConfiguration.SetPlatform(autodetect.OpenShiftPlatform)
	defer autodetect.OperatorConfiguration.SetPlatform(autodetect.KubernetesPlatform)
	cjob = rollover(j).(*batchv1.CronJob)
	assert.Equal(t,
		[]corev1.EnvVar{
			{Name: "INDEX_PREFIX", Value: "shortone"},
			{Name: "SKIP_DEPENDENCIES", Value: "true"},
			{Name: "CONDITIONS", Value: "weheee"},
		}, cjob.Spec.JobTemplate.Spec.Template.Spec.Containers[0].Env,
	)

	j.Spec.Storage.Options = v1.NewOptions(map[string]interface{}{
		"es.server-urls":    "foo,bar",
		"es.index-prefix":   "shortone",
		"skip-dependencies": "skip",
	})
	cjob = rollover(j).(*batchv1.CronJob)
	assert.Equal(t,
		[]corev1.EnvVar{
			{Name: "INDEX_PREFIX", Value: "shortone"},
			{Name: "SKIP_DEPENDENCIES", Value: "skip"},
			{Name: "CONDITIONS", Value: "weheee"},
		}, cjob.Spec.JobTemplate.Spec.Template.Spec.Containers[0].Env,
	)
}

func TestLookback(t *testing.T) {
	j := v1.NewJaeger(types.NamespacedName{Name: "squirtle"})
	j.Namespace = "kitchen"
	j.Spec.Storage.EsRollover.Image = "wohooo"
	j.Spec.Storage.EsRollover.ReadTTL = "2h"
	j.Spec.Storage.Options = v1.NewOptions(map[string]interface{}{"es.server-urls": "foo,bar", "es.index-prefix": "shortone"})
	historyLimits := int32(3)
	j.Spec.Storage.EsRollover.SuccessfulJobsHistoryLimit = &historyLimits

	cjob := lookback(j).(*batchv1.CronJob)
	assert.Equal(t, j.Namespace, cjob.Namespace)
	assert.Equal(t, []metav1.OwnerReference{util.AsOwner(j)}, cjob.OwnerReferences)
	assert.Equal(t, util.Labels("squirtle-es-lookback", "cronjob-es-lookback", *j), cjob.Labels)
	assert.Len(t, cjob.Spec.JobTemplate.Spec.Template.Spec.Containers, 1)
	assert.Equal(t, j.Spec.Storage.EsRollover.Image, cjob.Spec.JobTemplate.Spec.Template.Spec.Containers[0].Image)
	assert.Equal(t, []string{"lookback", "foo"}, cjob.Spec.JobTemplate.Spec.Template.Spec.Containers[0].Args)
	assert.Equal(t, []corev1.EnvVar{{Name: "INDEX_PREFIX", Value: "shortone"}, {Name: "UNIT", Value: "hours"}, {Name: "UNIT_COUNT", Value: "2"}}, cjob.Spec.JobTemplate.Spec.Template.Spec.Containers[0].Env)
	assert.Equal(t, historyLimits, *cjob.Spec.SuccessfulJobsHistoryLimit)
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
				"es.index-prefix":         "foo",
				"es.password":             "nopass",
				"es.username":             "fredy",
				"es.tls.enabled":          "true",
				"es.tls.ca":               "/etc/ca",
				"es.tls.key":              "/etc/key",
				"es.tls.cert":             "/etc/cert",
				"es.tls.skip-host-verify": "true",
			}),
			expected: []corev1.EnvVar{
				{Name: "INDEX_PREFIX", Value: "foo"},
				{Name: "ES_USERNAME", Value: "fredy"},
				{Name: "ES_PASSWORD", Value: "nopass"},
				{Name: "ES_TLS_ENABLED", Value: "true"},
				{Name: "ES_TLS_CA", Value: "/etc/ca"},
				{Name: "ES_TLS_CERT", Value: "/etc/cert"},
				{Name: "ES_TLS_KEY", Value: "/etc/key"},
				{Name: "ES_TLS_SKIP_HOST_VERIFY", Value: "true"},
			},
		},
	}
	for _, test := range tests {
		assert.EqualValues(t, test.expected, EsScriptEnvVars(test.opts))
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

func TestEsRolloverAnnotations(t *testing.T) {
	jaeger := v1.NewJaeger(types.NamespacedName{Name: "TestEsRolloverAnnotations"})
	jaeger.Spec.Annotations = map[string]string{
		"name":  "operator",
		"hello": "jaeger",
	}
	jaeger.Spec.Storage.EsRollover.Annotations = map[string]string{
		"hello":                "world", // Override top level annotation
		"prometheus.io/scrape": "false", // Override implicit value
	}

	cjob := rollover(jaeger).(*batchv1.CronJob)

	assert.Equal(t, "operator", cjob.Spec.JobTemplate.Spec.Template.Annotations["name"])
	assert.Equal(t, "false", cjob.Spec.JobTemplate.Spec.Template.Annotations["sidecar.istio.io/inject"])
	assert.Equal(t, "world", cjob.Spec.JobTemplate.Spec.Template.Annotations["hello"])
	assert.Equal(t, "false", cjob.Spec.JobTemplate.Spec.Template.Annotations["prometheus.io/scrape"])
	assert.Equal(t, "disabled", cjob.Spec.JobTemplate.Spec.Template.Annotations["linkerd.io/inject"])
}

func TestEsRolloverBackoffLimit(t *testing.T) {
	jaeger := v1.NewJaeger(types.NamespacedName{Name: "TestEsIndexCleanerAnnotations"})

	BackoffLimit := int32(3)
	jaeger.Spec.Storage.EsRollover.BackoffLimit = &BackoffLimit

	cjob := rollover(jaeger).(*batchv1.CronJob)
	assert.Equal(t, &BackoffLimit, cjob.Spec.JobTemplate.Spec.BackoffLimit)
}

func TestEsRolloverLabels(t *testing.T) {
	jaeger := v1.NewJaeger(types.NamespacedName{Name: "TestEsRolloverLabels"})
	jaeger.Spec.Labels = map[string]string{
		"name":  "operator",
		"hello": "jaeger",
	}
	jaeger.Spec.Storage.EsRollover.Labels = map[string]string{
		"hello":   "world", // Override top level label
		"another": "false",
	}

	cjob := rollover(jaeger).(*batchv1.CronJob)

	assert.Equal(t, "operator", cjob.Spec.JobTemplate.Spec.Template.Labels["name"])
	assert.Equal(t, "world", cjob.Spec.JobTemplate.Spec.Template.Labels["hello"])
	assert.Equal(t, "false", cjob.Spec.JobTemplate.Spec.Template.Labels["another"])
}

func TestEsRolloverResources(t *testing.T) {
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
					EsRollover: v1.JaegerEsRolloverSpec{
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
		cjob := rollover(test.jaeger).(*batchv1.CronJob)
		assert.Equal(t, test.expected, cjob.Spec.JobTemplate.Spec.Template.Spec.Containers[0].Resources)

	}
}

func TestEsRolloverLookbackAnnotations(t *testing.T) {
	jaeger := v1.NewJaeger(types.NamespacedName{Name: "TestEsRolloverLookbackAnnotations"})
	jaeger.Spec.Annotations = map[string]string{
		"name":  "operator",
		"hello": "jaeger",
	}
	jaeger.Spec.Storage.EsRollover.Annotations = map[string]string{
		"hello":                "world", // Override top level annotation
		"prometheus.io/scrape": "false", // Override implicit value
	}

	cjob := lookback(jaeger).(*batchv1.CronJob)

	assert.Equal(t, "operator", cjob.Spec.JobTemplate.Spec.Template.Annotations["name"])
	assert.Equal(t, "false", cjob.Spec.JobTemplate.Spec.Template.Annotations["sidecar.istio.io/inject"])
	assert.Equal(t, "world", cjob.Spec.JobTemplate.Spec.Template.Annotations["hello"])
	assert.Equal(t, "false", cjob.Spec.JobTemplate.Spec.Template.Annotations["prometheus.io/scrape"])
	assert.Equal(t, "disabled", cjob.Spec.JobTemplate.Spec.Template.Annotations["linkerd.io/inject"])
}

func TestEsRolloverLookbackLabels(t *testing.T) {
	jaeger := v1.NewJaeger(types.NamespacedName{Name: "TestEsRolloverLookbackLabels"})
	jaeger.Spec.Labels = map[string]string{
		"name":  "operator",
		"hello": "jaeger",
	}
	jaeger.Spec.Storage.EsRollover.Labels = map[string]string{
		"hello":   "world", // Override top level label
		"another": "false",
	}

	cjob := lookback(jaeger).(*batchv1.CronJob)

	assert.Equal(t, "operator", cjob.Spec.JobTemplate.Spec.Template.Labels["name"])
	assert.Equal(t, "world", cjob.Spec.JobTemplate.Spec.Template.Labels["hello"])
	assert.Equal(t, "false", cjob.Spec.JobTemplate.Spec.Template.Labels["another"])
}

func TestEsRolloverLookbackResources(t *testing.T) {
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
					EsRollover: v1.JaegerEsRolloverSpec{
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
		cjob := lookback(test.jaeger).(*batchv1.CronJob)
		assert.Equal(t, test.expected, cjob.Spec.JobTemplate.Spec.Template.Spec.Containers[0].Resources)

	}
}

func TestDefaultEsRolloverImage(t *testing.T) {
	viper.SetDefault("jaeger-es-rollover-image", "jaegertracing/jaeger-es-rollover")

	jaeger := v1.NewJaeger(types.NamespacedName{Name: "TestDefaultEsRolloverImage"})

	cjob := lookback(jaeger).(*batchv1.CronJob)
	assert.Empty(t, jaeger.Spec.Storage.EsRollover.Image)
	assert.Equal(t, "jaegertracing/jaeger-es-rollover:"+version.Get().Jaeger, cjob.Spec.JobTemplate.Spec.Template.Spec.Containers[0].Image)
}

func TestCustomEsRolloverImage(t *testing.T) {
	viper.Set("jaeger-es-rollover-image", "org/custom-es-rollover-image")
	defer viper.Reset()

	jaeger := v1.NewJaeger(types.NamespacedName{Name: "TestDefaultEsRolloverImage"})

	cjob := lookback(jaeger).(*batchv1.CronJob)
	assert.Empty(t, jaeger.Spec.Storage.EsRollover.Image)
	assert.Equal(t, "org/custom-es-rollover-image:"+version.Get().Jaeger, cjob.Spec.JobTemplate.Spec.Template.Spec.Containers[0].Image)
}

func TestEsRolloverImagePullSecrets(t *testing.T) {
	jaeger := v1.NewJaeger(types.NamespacedName{Name: "TestEsRolloverImagePullSecrets"})
	const pullSecret = "mysecret"
	jaeger.Spec.ImagePullSecrets = []corev1.LocalObjectReference{
		{
			Name: pullSecret,
		},
	}

	cjob := lookback(jaeger).(*batchv1.CronJob)

	assert.Equal(t, pullSecret, cjob.Spec.JobTemplate.Spec.Template.Spec.ImagePullSecrets[0].Name)
}

func TestEsRolloverContainerSecurityContext(t *testing.T) {
	trueVar := true
	falseVar := false
	idVar := int64(1234)
	securityContextVar := corev1.SecurityContext{
		RunAsNonRoot:             &trueVar,
		RunAsGroup:               &idVar,
		RunAsUser:                &idVar,
		Capabilities:             &corev1.Capabilities{Drop: []corev1.Capability{"ALL"}},
		Privileged:               &falseVar,
		AllowPrivilegeEscalation: &falseVar,
		SeccompProfile:           &corev1.SeccompProfile{Type: "RuntimeDefault"},
	}
	jaeger := v1.NewJaeger(types.NamespacedName{Name: "TestEsRolloverContainerSecurityContext"})
	jaeger.Spec.Storage.EsRollover.ContainerSecurityContext = &securityContextVar
	cjob := lookback(jaeger).(*batchv1.CronJob)

	assert.Equal(t, securityContextVar, *cjob.Spec.JobTemplate.Spec.Template.Spec.Containers[0].SecurityContext)
}

func TestEsRolloverContainerSecurityContextOverride(t *testing.T) {
	trueVar := true
	falseVar := false
	idVar1 := int64(1234)
	idVar2 := int64(4321)
	securityContextVar := corev1.SecurityContext{
		RunAsNonRoot:             &trueVar,
		RunAsGroup:               &idVar1,
		RunAsUser:                &idVar1,
		Capabilities:             &corev1.Capabilities{Drop: []corev1.Capability{"ALL"}},
		Privileged:               &falseVar,
		AllowPrivilegeEscalation: &falseVar,
		SeccompProfile:           &corev1.SeccompProfile{Type: "RuntimeDefault"},
	}
	overrideSecurityContextVar := corev1.SecurityContext{
		RunAsNonRoot:             &trueVar,
		RunAsGroup:               &idVar2,
		RunAsUser:                &idVar2,
		Capabilities:             &corev1.Capabilities{Drop: []corev1.Capability{"ALL"}},
		Privileged:               &falseVar,
		AllowPrivilegeEscalation: &falseVar,
		SeccompProfile:           &corev1.SeccompProfile{Type: "RuntimeDefault"},
	}
	jaeger := v1.NewJaeger(types.NamespacedName{Name: "TestEsRolloverContainerSecurityContext"})
	jaeger.Spec.JaegerCommonSpec.ContainerSecurityContext = &securityContextVar
	jaeger.Spec.Storage.EsRollover.ContainerSecurityContext = &overrideSecurityContextVar
	cjob := lookback(jaeger).(*batchv1.CronJob)

	assert.Equal(t, overrideSecurityContextVar, *cjob.Spec.JobTemplate.Spec.Template.Spec.Containers[0].SecurityContext)
}
