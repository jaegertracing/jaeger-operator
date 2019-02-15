package storage

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/jaegertracing/jaeger-operator/pkg/apis/io/v1alpha1"
)

func TestShouldDeployElasticsearch(t *testing.T) {
	tests := []struct {
		j        v1alpha1.JaegerStorageSpec
		expected bool
	}{
		{j: v1alpha1.JaegerStorageSpec{}},
		{j: v1alpha1.JaegerStorageSpec{Type: "cassandra"}},
		{j: v1alpha1.JaegerStorageSpec{Type: "elasticsearch", Options: v1alpha1.NewOptions(map[string]interface{}{"es.server-urls": "foo"})}},
		{j: v1alpha1.JaegerStorageSpec{Type: "ElasticSearch"}, expected: true},
	}
	for _, test := range tests {
		assert.Equal(t, test.expected, ShouldDeployElasticsearch(test.j))
	}
}

func TestCreateElasticsearchCR(t *testing.T) {
	trueVar := true
	j := v1alpha1.NewJaeger("foo")
	j.Namespace = "myproject"
	cr := createCr(j)
	assert.Equal(t, "myproject", cr.Namespace)
	assert.Equal(t, "elasticsearch", cr.Name)
	assert.Equal(t, []metav1.OwnerReference{{Name: "foo", Controller: &trueVar}}, cr.OwnerReferences)
}

func TestInject(t *testing.T) {
	p := &v1.PodSpec{
		Containers: []v1.Container{{
			Args:         []string{"foo"},
			VolumeMounts: []v1.VolumeMount{{Name: "lol"}},
		}},
	}
	es := &ElasticsearchDeployment{}
	es.InjectStorageConfiguration(p)
	expVolumes := []v1.Volume{{Name: "certs", VolumeSource: v1.VolumeSource{
		Secret: &v1.SecretVolumeSource{
			SecretName: "jaeger-elasticsearch",
		},
	}}}
	assert.Equal(t, expVolumes, p.Volumes)
	expContainers := []v1.Container{{
		Args: []string{
			"foo",
			"--es.server-urls=https://elasticsearch:9200",
			"--es.token-file=" + k8sTokenFile,
			"--es.tls.ca=" + caPath,
		},
		VolumeMounts: []v1.VolumeMount{
			{Name: "lol"},
			{
				Name:      volumeName,
				ReadOnly:  true,
				MountPath: volumeMountPath,
			},
		},
	}}
	assert.Equal(t, expContainers, p.Containers)
}

func TestCreateElasticsearchObjects(t *testing.T) {
	j := v1alpha1.NewJaeger("foo")
	es := &ElasticsearchDeployment{Jaeger: j}
	objs, err := es.CreateElasticsearchObjects()
	assert.Nil(t, objs)
	assert.EqualError(t, err, "failed to create Elasticsearch certificates: failed to get watch namespace: WATCH_NAMESPACE must be set")
}
