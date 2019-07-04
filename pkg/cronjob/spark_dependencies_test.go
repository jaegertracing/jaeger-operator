package cronjob

import (
	"testing"

	"github.com/stretchr/testify/assert"
	corev1 "k8s.io/api/core/v1"

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
