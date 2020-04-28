package cronjob

import (
	"testing"
	"time"

	"github.com/jaegertracing/jaeger-operator/pkg/version"

	"github.com/spf13/viper"
	"github.com/stretchr/testify/assert"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
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
	historyLimits := int32(2)
	j.Spec.Storage.EsRollover.SuccessfulJobsHistoryLimit = &historyLimits

	cjob := rollover(j)
	assert.Equal(t, j.Namespace, cjob.Namespace)
	assert.Equal(t, []metav1.OwnerReference{util.AsOwner(j)}, cjob.OwnerReferences)
	assert.Equal(t, util.Labels("eevee-es-rollover", "cronjob-es-rollover", *j), cjob.Labels)
	assert.Equal(t, 1, len(cjob.Spec.JobTemplate.Spec.Template.Spec.Containers))
	assert.Equal(t, j.Spec.Storage.EsRollover.Image, cjob.Spec.JobTemplate.Spec.Template.Spec.Containers[0].Image)
	assert.Equal(t, []string{"rollover", "foo"}, cjob.Spec.JobTemplate.Spec.Template.Spec.Containers[0].Args)
	assert.Equal(t, []corev1.EnvVar{{Name: "INDEX_PREFIX", Value: "shortone"}, {Name: "CONDITIONS", Value: "weheee"}}, cjob.Spec.JobTemplate.Spec.Template.Spec.Containers[0].Env)
	assert.Equal(t, historyLimits, *cjob.Spec.SuccessfulJobsHistoryLimit)
}

func TestLookback(t *testing.T) {
	j := v1.NewJaeger(types.NamespacedName{Name: "squirtle"})
	j.Namespace = "kitchen"
	j.Spec.Storage.EsRollover.Image = "wohooo"
	j.Spec.Storage.EsRollover.ReadTTL = "2h"
	j.Spec.Storage.Options = v1.NewOptions(map[string]interface{}{"es.server-urls": "foo,bar", "es.index-prefix": "shortone"})
	historyLimits := int32(3)
	j.Spec.Storage.EsRollover.SuccessfulJobsHistoryLimit = &historyLimits

	cjob := lookback(j)
	assert.Equal(t, j.Namespace, cjob.Namespace)
	assert.Equal(t, []metav1.OwnerReference{util.AsOwner(j)}, cjob.OwnerReferences)
	assert.Equal(t, util.Labels("squirtle-es-lookback", "cronjob-es-lookback", *j), cjob.Labels)
	assert.Equal(t, 1, len(cjob.Spec.JobTemplate.Spec.Template.Spec.Containers))
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
				"es.tls":                  "true",
				"es.tls.ca":               "/etc/ca",
				"es.tls.key":              "/etc/key",
				"es.tls.cert":             "/etc/cert",
				"es.tls.skip-host-verify": "true",
			}),
			expected: []corev1.EnvVar{
				{Name: "INDEX_PREFIX", Value: "foo"},
				{Name: "ES_USERNAME", Value: "fredy"},
				{Name: "ES_PASSWORD", Value: "nopass"},
				{Name: "ES_TLS", Value: "true"},
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

	cjob := rollover(jaeger)

	assert.Equal(t, "operator", cjob.Spec.JobTemplate.Spec.Template.Annotations["name"])
	assert.Equal(t, "false", cjob.Spec.JobTemplate.Spec.Template.Annotations["sidecar.istio.io/inject"])
	assert.Equal(t, "world", cjob.Spec.JobTemplate.Spec.Template.Annotations["hello"])
	assert.Equal(t, "false", cjob.Spec.JobTemplate.Spec.Template.Annotations["prometheus.io/scrape"])
	assert.Equal(t, "disabled", cjob.Spec.JobTemplate.Spec.Template.Annotations["linkerd.io/inject"])
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

	cjob := rollover(jaeger)

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
		cjob := rollover(test.jaeger)
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

	cjob := lookback(jaeger)

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

	cjob := lookback(jaeger)

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
		cjob := lookback(test.jaeger)
		assert.Equal(t, test.expected, cjob.Spec.JobTemplate.Spec.Template.Spec.Containers[0].Resources)

	}
}

func TestDefaultEsRolloverImage(t *testing.T) {
	viper.SetDefault("jaeger-es-rollover-image", "jaegertracing/jaeger-es-rollover")

	jaeger := v1.NewJaeger(types.NamespacedName{Name: "TestDefaultEsRolloverImage"})

	cjob := lookback(jaeger)
	assert.Empty(t, jaeger.Spec.Storage.EsRollover.Image)
	assert.Equal(t, "jaegertracing/jaeger-es-rollover:"+version.Get().Jaeger, cjob.Spec.JobTemplate.Spec.Template.Spec.Containers[0].Image)
}

func TestCustomEsRolloverImage(t *testing.T) {
	viper.Set("jaeger-es-rollover-image", "org/custom-es-rollover-image")
	defer viper.Reset()

	jaeger := v1.NewJaeger(types.NamespacedName{Name: "TestDefaultEsRolloverImage"})

	cjob := lookback(jaeger)
	assert.Empty(t, jaeger.Spec.Storage.EsRollover.Image)
	assert.Equal(t, "org/custom-es-rollover-image:"+version.Get().Jaeger, cjob.Spec.JobTemplate.Spec.Template.Spec.Containers[0].Image)
}
