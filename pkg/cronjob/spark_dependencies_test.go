package cronjob

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"k8s.io/api/core/v1"

	"github.com/jaegertracing/jaeger-operator/pkg/apis/io/v1alpha1"
)

func TestRemoveEmptyVars(t *testing.T) {
	tests := []struct {
		underTest []v1.EnvVar
		expected  []v1.EnvVar
	}{
		{},
		{underTest: []v1.EnvVar{{Name: "foo", Value: "bar"}, {Name: "foo3"}, {Name: "foo2", ValueFrom: &v1.EnvVarSource{}}},
			expected: []v1.EnvVar{{Name: "foo", Value: "bar"}, {Name: "foo2", ValueFrom: &v1.EnvVarSource{}}}},
		{underTest: []v1.EnvVar{{Name: "foo"}}},
	}
	for _, test := range tests {
		exp := removeEmptyVars(test.underTest)
		assert.Equal(t, test.expected, exp)
	}
}

func TestStorageEnvs(t *testing.T) {
	tests := []struct {
		storage  v1alpha1.JaegerStorageSpec
		expected []v1.EnvVar
	}{
		{storage: v1alpha1.JaegerStorageSpec{Type: "foo"}},
		{storage: v1alpha1.JaegerStorageSpec{Type: "cassandra",
			Options: v1alpha1.NewOptions(map[string]interface{}{"cassandra.servers": "lol:hol", "cassandra.keyspace": "haha",
				"cassandra.username": "jdoe", "cassandra.password": "none"})},
			expected: []v1.EnvVar{
				{Name: "CASSANDRA_CONTACT_POINTS", Value: "lol:hol"},
				{Name: "CASSANDRA_KEYSPACE", Value: "haha"},
				{Name: "CASSANDRA_USERNAME", Value: "jdoe"},
				{Name: "CASSANDRA_PASSWORD", Value: "none"},
				{Name: "CASSANDRA_USE_SSL", Value: "false"},
				{Name: "CASSANDRA_LOCAL_DC", Value: ""},
				{Name: "CASSANDRA_CLIENT_AUTH_ENABLED", Value: "false"},
			}},
		{storage: v1alpha1.JaegerStorageSpec{Type: "elasticsearch",
			Options: v1alpha1.NewOptions(map[string]interface{}{"es.server-urls": "lol:hol", "es.index-prefix": "haha",
				"es.username": "jdoe", "es.password": "none"})},
			expected: []v1.EnvVar{
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
	assert.NotNil(t, CreateSparkDependencies(&v1alpha1.Jaeger{Spec: v1alpha1.JaegerSpec{Storage: v1alpha1.JaegerStorageSpec{Type: "elasticsearch"}}}))
}
