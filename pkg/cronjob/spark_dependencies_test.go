package cronjob

import (
	"testing"

	"github.com/spf13/viper"
	"github.com/stretchr/testify/assert"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	"k8s.io/apimachinery/pkg/types"

	v1 "github.com/jaegertracing/jaeger-operator/pkg/apis/jaegertracing/v1"
)

func TestStorageEnvs(t *testing.T) {
	trueVar := true
	falseVar := false
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
			}},
		{storage: v1.JaegerStorageSpec{Type: "elasticsearch",
			Options: v1.NewOptions(map[string]interface{}{"es.server-urls": "lol:hol", "es.index-prefix": "haha",
				"es.username": "jdoe", "es.password": "none"}),
			Dependencies: v1.JaegerDependenciesSpec{ElasticsearchClientNodeOnly: &trueVar, ElasticsearchNodesWanOnly: &falseVar}},
			expected: []corev1.EnvVar{
				{Name: "ES_NODES", Value: "lol:hol"},
				{Name: "ES_INDEX_PREFIX", Value: "haha"},
				{Name: "ES_USERNAME", Value: "jdoe"},
				{Name: "ES_PASSWORD", Value: "none"},
				{Name: "ES_NODES_WAN_ONLY", Value: "false"},
				{Name: "ES_CLIENT_NODE_ONLY", Value: "true"},
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

func TestSparkDependenciesSecrets(t *testing.T) {
	jaeger := v1.NewJaeger(types.NamespacedName{Name: "TestSparkDependenciesSecrets"})
	secret := "mysecret"
	jaeger.Spec.Storage.SecretName = secret

	days := 0
	jaeger.Spec.Storage.EsIndexCleaner.NumberOfDays = &days
	cronJob := CreateSparkDependencies(jaeger)
	assert.Len(t, cronJob.Spec.JobTemplate.Spec.Template.Spec.Containers, 1)
	assert.Len(t, cronJob.Spec.JobTemplate.Spec.Template.Spec.Containers[0].EnvFrom, 1)
	assert.Equal(t, secret, cronJob.Spec.JobTemplate.Spec.Template.Spec.Containers[0].EnvFrom[0].SecretRef.LocalObjectReference.Name)
}

func TestSparkDependencies(t *testing.T) {
	j := &v1.Jaeger{Spec: v1.JaegerSpec{Storage: v1.JaegerStorageSpec{Type: "elasticsearch"}}}
	historyLimits := int32(3)
	j.Spec.Storage.Dependencies.SuccessfulJobsHistoryLimit = &historyLimits
	cjob := CreateSparkDependencies(j)
	assert.Equal(t, j.Namespace, cjob.Namespace)
	assert.Equal(t, historyLimits, *cjob.Spec.SuccessfulJobsHistoryLimit)
}

func TestDependenciesAnnotations(t *testing.T) {
	jaeger := v1.NewJaeger(types.NamespacedName{Name: "TestDependenciesAnnotations"})
	jaeger.Spec.Annotations = map[string]string{
		"name":  "operator",
		"hello": "jaeger",
	}
	jaeger.Spec.Storage.Dependencies.Annotations = map[string]string{
		"hello":                "world", // Override top level annotation
		"prometheus.io/scrape": "false", // Override implicit value
	}

	cjob := CreateSparkDependencies(jaeger)

	assert.Equal(t, "operator", cjob.Spec.JobTemplate.Spec.Template.Annotations["name"])
	assert.Equal(t, "false", cjob.Spec.JobTemplate.Spec.Template.Annotations["sidecar.istio.io/inject"])
	assert.Equal(t, "world", cjob.Spec.JobTemplate.Spec.Template.Annotations["hello"])
	assert.Equal(t, "false", cjob.Spec.JobTemplate.Spec.Template.Annotations["prometheus.io/scrape"])
	assert.Equal(t, "disabled", cjob.Spec.JobTemplate.Spec.Template.Annotations["linkerd.io/inject"])
}

func TestDependenciesLabels(t *testing.T) {
	jaeger := v1.NewJaeger(types.NamespacedName{Name: "TestDependenciesLabels"})
	jaeger.Spec.Labels = map[string]string{
		"name":  "operator",
		"hello": "jaeger",
	}
	jaeger.Spec.Storage.Dependencies.Labels = map[string]string{
		"hello":   "world", // Override top level label
		"another": "false",
	}

	cjob := CreateSparkDependencies(jaeger)

	assert.Equal(t, "operator", cjob.Spec.JobTemplate.Spec.Template.Labels["name"])
	assert.Equal(t, "world", cjob.Spec.JobTemplate.Spec.Template.Labels["hello"])
	assert.Equal(t, "false", cjob.Spec.JobTemplate.Spec.Template.Labels["another"])

	// Check if the labels of cronjob pod template equal to the labels of cronjob.
	assert.Equal(t, cjob.ObjectMeta.Labels, cjob.Spec.JobTemplate.Spec.Template.ObjectMeta.Labels)
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
						JaegerCommonSpec: v1.JaegerCommonSpec{
							Resources: dependencyResources,
						},
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

func TestDefaultSparkDependenciesImage(t *testing.T) {
	viper.SetDefault("jaeger-spark-dependencies-image", "jaegertracing/spark-dependencies")

	jaeger := v1.NewJaeger(types.NamespacedName{Name: "TestDefaultSparkDependenciesImage"})

	cjob := CreateSparkDependencies(jaeger)
	assert.Empty(t, jaeger.Spec.Storage.Dependencies.Image)
	assert.Equal(t, "jaegertracing/spark-dependencies", cjob.Spec.JobTemplate.Spec.Template.Spec.Containers[0].Image)
}

func TestCustomSparkDependenciesImage(t *testing.T) {
	viper.Set("jaeger-spark-dependencies-image", "org/custom-spark-dependencies-image")
	defer viper.Reset()

	jaeger := v1.NewJaeger(types.NamespacedName{Name: "TestDefaultSparkDependenciesImage"})

	cjob := CreateSparkDependencies(jaeger)
	assert.Empty(t, jaeger.Spec.Storage.Dependencies.Image)
	assert.Equal(t, "org/custom-spark-dependencies-image", cjob.Spec.JobTemplate.Spec.Template.Spec.Containers[0].Image)
}
