package storage

import (
	"testing"

	"github.com/spf13/viper"
	"github.com/stretchr/testify/assert"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	"k8s.io/apimachinery/pkg/types"

	v1 "github.com/jaegertracing/jaeger-operator/pkg/apis/jaegertracing/v1"
)

func TestCassandraCustomImage(t *testing.T) {
	jaeger := v1.NewJaeger(types.NamespacedName{Name: "my-instance"})
	jaeger.Spec.Storage.CassandraCreateSchema.Image = "mynamespace/image:version"

	b := cassandraDeps(jaeger)
	assert.Len(t, b, 1)
	assert.Len(t, b[0].Spec.Template.Spec.Containers, 1)
	assert.Equal(t, "mynamespace/image:version", b[0].Spec.Template.Spec.Containers[0].Image)
}

func TestDefaultImage(t *testing.T) {
	viper.Set("jaeger-cassandra-schema-image", "jaegertracing/theimage")
	defer viper.Reset()

	jaeger := v1.NewJaeger(types.NamespacedName{Name: "my-instance"})

	b := cassandraDeps(jaeger)
	assert.Len(t, b, 1)
	assert.Len(t, b[0].Spec.Template.Spec.Containers, 1)
	assert.Equal(t, "jaegertracing/theimage:0.0.0", b[0].Spec.Template.Spec.Containers[0].Image)
}

func TestCassandraCreateSchemaDisabled(t *testing.T) {
	falseVar := false

	jaeger := v1.NewJaeger(types.NamespacedName{Name: "TestCassandraCreateSchemaDisabled"})
	jaeger.Spec.Storage.CassandraCreateSchema.Enabled = &falseVar

	assert.Len(t, cassandraDeps(jaeger), 0)
}

func TestCassandraCreateSchemaEnabled(t *testing.T) {
	trueVar := true

	jaeger := v1.NewJaeger(types.NamespacedName{Name: "TestCassandraCreateSchemaEnabled"})
	jaeger.Spec.Storage.CassandraCreateSchema.Enabled = &trueVar

	assert.Len(t, cassandraDeps(jaeger), 1)
}

func TestCassandraCreateSchemaEnabledNil(t *testing.T) {
	jaeger := v1.NewJaeger(types.NamespacedName{Name: "TestCassandraCreateSchemaEnabledNil"})

	assert.Nil(t, jaeger.Spec.Storage.CassandraCreateSchema.Enabled)
	assert.Len(t, cassandraDeps(jaeger), 1)
}

func TestCassandraCreateSchemaAnnotations(t *testing.T) {
	jaeger := v1.NewJaeger(types.NamespacedName{Name: "TestCassandraCreateSchemaAnnotations"})
	jaeger.Spec.Annotations = map[string]string{
		"name":  "operator",
		"hello": "jaeger",
	}
	jaeger.Spec.Storage.CassandraCreateSchema.Annotations = map[string]string{
		"hello":                "world", // Override top level annotation
		"prometheus.io/scrape": "false", // Override implicit value
	}

	cjob := cassandraDeps(jaeger)[0]

	assert.Equal(t, "operator", cjob.Spec.Template.Annotations["name"])
	assert.Equal(t, "false", cjob.Spec.Template.Annotations["sidecar.istio.io/inject"])
	assert.Equal(t, "world", cjob.Spec.Template.Annotations["hello"])
	assert.Equal(t, "false", cjob.Spec.Template.Annotations["prometheus.io/scrape"])
	assert.Equal(t, "disabled", cjob.Spec.Template.Annotations["linkerd.io/inject"])
}

func TestCassandraCreateSchemaLabels(t *testing.T) {
	jaeger := v1.NewJaeger(types.NamespacedName{Name: "TestCassandraCreateSchemaLabels"})
	jaeger.Spec.Labels = map[string]string{
		"name":  "operator",
		"hello": "jaeger",
	}
	jaeger.Spec.Storage.CassandraCreateSchema.Labels = map[string]string{
		"hello":   "world", // Override top level label
		"another": "false",
	}

	cjob := cassandraDeps(jaeger)[0]

	assert.Equal(t, "operator", cjob.Spec.Template.Labels["name"])
	assert.Equal(t, "world", cjob.Spec.Template.Labels["hello"])
	assert.Equal(t, "false", cjob.Spec.Template.Labels["another"])
}

func TestCassandraCreateSchemaResources(t *testing.T) {

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
					CassandraCreateSchema: v1.JaegerCassandraCreateSchemaSpec{
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
		cjob := cassandraDeps(test.jaeger)[0]
		assert.Equal(t, test.expected, cjob.Spec.Template.Spec.Containers[0].Resources)

	}
}
