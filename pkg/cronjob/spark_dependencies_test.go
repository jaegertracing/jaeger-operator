package cronjob

import (
	"testing"

	"github.com/stretchr/testify/assert"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"

	v1 "github.com/jaegertracing/jaeger-operator/pkg/apis/jaegertracing/v1"
)

func TestRemoveEmptyVars(t *testing.T) {
	tests := []struct {
		underTest []corev1.EnvVar
		expected  []corev1.EnvVar
	}{
		{},
		{underTest: []corev1.EnvVar{{Name: "foo", Value: "bar"}, {Name: "foo3"}, {Name: "foo2", ValueFrom: &corev1.EnvVarSource{}}},
			expected: []corev1.EnvVar{{Name: "foo", Value: "bar"}, {Name: "foo2", ValueFrom: &corev1.EnvVarSource{}}}},
		{underTest: []corev1.EnvVar{{Name: "foo"}}},
	}
	for _, test := range tests {
		exp := removeEmptyVars(test.underTest)
		assert.Equal(t, test.expected, exp)
	}
}

func TestStorageEnvs(t *testing.T) {
	tests := []struct {
		storage  v1.JaegerStorageSpec
		expected []corev1.EnvVar
	}{
		{storage: v1.JaegerStorageSpec{Type: "foo"}},
		{storage: v1.JaegerStorageSpec{Type: "cassandra",
			Options: v1.NewOptions(map[string]interface{}{"cassandra.servers": "lol:hol", "cassandra.keyspace": "haha",
				"cassandra.username": "jdoe", "cassandra.password": "none"})},
			expected: []corev1.EnvVar{
				{Name: "CASSANDRA_CONTACT_POINTS", Value: "lol:hol"},
				{Name: "CASSANDRA_KEYSPACE", Value: "haha"},
				{Name: "CASSANDRA_USERNAME", Value: "jdoe"},
				{Name: "CASSANDRA_PASSWORD", Value: "none"},
				{Name: "CASSANDRA_USE_SSL", Value: ""},
				{Name: "CASSANDRA_LOCAL_DC", Value: ""},
				{Name: "CASSANDRA_CLIENT_AUTH_ENABLED", Value: "false"},
			}},
		{storage: v1.JaegerStorageSpec{Type: "cassandra",
			Options: v1.NewOptions(map[string]interface{}{"cassandra.servers": "lol:hol", "cassandra.keyspace": "haha",
				"cassandra.username": "jdoe", "cassandra.password": "none", "cassandra.tls": "ofcourse!", "cassandra.local-dc": "no-remote"})},
			expected: []corev1.EnvVar{
				{Name: "CASSANDRA_CONTACT_POINTS", Value: "lol:hol"},
				{Name: "CASSANDRA_KEYSPACE", Value: "haha"},
				{Name: "CASSANDRA_USERNAME", Value: "jdoe"},
				{Name: "CASSANDRA_PASSWORD", Value: "none"},
				{Name: "CASSANDRA_USE_SSL", Value: "ofcourse!"},
				{Name: "CASSANDRA_LOCAL_DC", Value: "no-remote"},
				{Name: "CASSANDRA_CLIENT_AUTH_ENABLED", Value: "false"},
			}},
		{storage: v1.JaegerStorageSpec{Type: "elasticsearch",
			Options: v1.NewOptions(map[string]interface{}{"es.server-urls": "lol:hol", "es.index-prefix": "haha",
				"es.username": "jdoe", "es.password": "none"})},
			expected: []corev1.EnvVar{
				{Name: "ES_NODES", Value: "lol:hol"},
				{Name: "ES_INDEX_PREFIX", Value: "haha"},
				{Name: "ES_USERNAME", Value: "jdoe"},
				{Name: "ES_PASSWORD", Value: "none"},
				{Name: "ES_CLIENT_NODE_ONLY", Value: "false"},
				{Name: "ES_NODES_WAN_ONLY", Value: "false"},
			}},
	}
	for _, test := range tests {
		envVars := getStorageEnvs(test.storage)
		assert.Equal(t, test.expected, envVars)
	}
}

func TestCreate(t *testing.T) {
	assert.NotNil(t, CreateSparkDependencies(&v1.Jaeger{Spec: v1.JaegerSpec{Storage: v1.JaegerStorageSpec{Type: "elasticsearch"}}}))
}

func TestSparkDependencies(t *testing.T) {
	j := &v1.Jaeger{Spec: v1.JaegerSpec{Storage: v1.JaegerStorageSpec{Type: "elasticsearch"}}}

	cjob := CreateSparkDependencies(j)
	assert.Equal(t, j.Namespace, cjob.Namespace)
}

func TestSparkDependenciesResources(t *testing.T) {

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

	dependencyResources := corev1.ResourceRequirements{
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
					Dependencies: v1.JaegerDependenciesSpec{
						Resources: dependencyResources,
					},
				},
				JaegerCommonSpec: v1.JaegerCommonSpec{
					Resources: parentResources,
				},
			}},
			expected: dependencyResources,
		},
	}
	for _, test := range tests {
		cjob := CreateSparkDependencies(test.jaeger)
		assert.Equal(t, test.expected, cjob.Spec.JobTemplate.Spec.Template.Spec.Containers[0].Resources)

	}
}
