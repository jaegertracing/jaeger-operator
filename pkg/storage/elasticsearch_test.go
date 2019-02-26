package storage

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/jaegertracing/jaeger-operator/pkg/apis/io/v1alpha1"
	esv1alpha1 "github.com/jaegertracing/jaeger-operator/pkg/storage/elasticsearch/v1alpha1"
)

func TestShouldDeployElasticsearch(t *testing.T) {
	tests := []struct {
		j        v1alpha1.JaegerStorageSpec
		expected bool
	}{
		{j: v1alpha1.JaegerStorageSpec{}},
		{j: v1alpha1.JaegerStorageSpec{Type: "cassandra"}},
		{j: v1alpha1.JaegerStorageSpec{Type: "elasticsearch", Options: v1alpha1.NewOptions(map[string]interface{}{"es.server-urls": "foo"})}},
		{j: v1alpha1.JaegerStorageSpec{Type: "elasticsearch"}, expected: true},
	}
	for _, test := range tests {
		assert.Equal(t, test.expected, ShouldDeployElasticsearch(test.j))
	}
}

func TestCreateElasticsearchCR(t *testing.T) {
	tests := []struct {
		jEsSpec v1alpha1.ElasticsearchSpec
		esSpec  esv1alpha1.ElasticsearchSpec
	}{
		{
			jEsSpec: v1alpha1.ElasticsearchSpec{
				NodeCount:        2,
				RedundancyPolicy: esv1alpha1.FullRedundancy,
				Storage: esv1alpha1.ElasticsearchStorageSpec{
					StorageClassName: "floppydisk",
				},
			},
			esSpec: esv1alpha1.ElasticsearchSpec{
				ManagementState:  esv1alpha1.ManagementStateManaged,
				RedundancyPolicy: esv1alpha1.FullRedundancy,
				Spec:             esv1alpha1.ElasticsearchNodeSpec{},
				Nodes: []esv1alpha1.ElasticsearchNode{
					{
						NodeCount: 2,
						Storage:   esv1alpha1.ElasticsearchStorageSpec{StorageClassName: "floppydisk"},
						Roles:     []esv1alpha1.ElasticsearchNodeRole{esv1alpha1.ElasticsearchRoleClient, esv1alpha1.ElasticsearchRoleData, esv1alpha1.ElasticsearchRoleMaster},
					},
				},
			},
		},
		{
			jEsSpec: v1alpha1.ElasticsearchSpec{
				NodeCount:        5,
				RedundancyPolicy: esv1alpha1.FullRedundancy,
				Storage: esv1alpha1.ElasticsearchStorageSpec{
					StorageClassName: "floppydisk",
				},
			},
			esSpec: esv1alpha1.ElasticsearchSpec{
				ManagementState:  esv1alpha1.ManagementStateManaged,
				RedundancyPolicy: esv1alpha1.FullRedundancy,
				Spec:             esv1alpha1.ElasticsearchNodeSpec{},
				Nodes: []esv1alpha1.ElasticsearchNode{
					{
						NodeCount: 3,
						Storage:   esv1alpha1.ElasticsearchStorageSpec{StorageClassName: "floppydisk"},
						Roles:     []esv1alpha1.ElasticsearchNodeRole{esv1alpha1.ElasticsearchRoleMaster},
					},
					{
						NodeCount: 2,
						Storage:   esv1alpha1.ElasticsearchStorageSpec{StorageClassName: "floppydisk"},
						Roles:     []esv1alpha1.ElasticsearchNodeRole{esv1alpha1.ElasticsearchRoleClient, esv1alpha1.ElasticsearchRoleData},
					},
				},
			},
		},
	}
	for _, test := range tests {
		j := v1alpha1.NewJaeger("foo")
		j.Namespace = "myproject"
		j.Spec.Storage.Elasticsearch = test.jEsSpec
		es := &ElasticsearchDeployment{Jaeger: j}
		cr := es.createCr()
		assert.Equal(t, "myproject", cr.Namespace)
		assert.Equal(t, "elasticsearch", cr.Name)
		trueVar := true
		assert.Equal(t, []metav1.OwnerReference{{Name: "foo", Controller: &trueVar}}, cr.OwnerReferences)
		assert.Equal(t, cr.Spec, test.esSpec)
	}
}

func TestInject(t *testing.T) {
	tests := []struct {
		pod      *v1.PodSpec
		extected *v1.PodSpec
	}{
		{pod: &v1.PodSpec{
			Containers: []v1.Container{{
				Args:         []string{"foo"},
				VolumeMounts: []v1.VolumeMount{{Name: "lol"}},
			}},
		},
			extected: &v1.PodSpec{
				Containers: []v1.Container{{
					Args: []string{
						"foo",
						"--es.server-urls=" + elasticsearchUrl,
						"--es.token-file=" + k8sTokenFile,
						"--es.tls.ca=" + caPath,
						"--es.num-shards=0",
						"--es.num-replicas=1",
					},
					VolumeMounts: []v1.VolumeMount{
						{Name: "lol"},
						{Name: volumeName, ReadOnly: true, MountPath: volumeMountPath},
					},
				}},
				Volumes: []v1.Volume{{Name: "certs", VolumeSource: v1.VolumeSource{
					Secret: &v1.SecretVolumeSource{
						SecretName: "hoo-jaeger-elasticsearch"}}},
				}},
		},
		{pod: &v1.PodSpec{
			Containers: []v1.Container{{
				Args: []string{"--es.num-shards=15"},
			}},
		},
			extected: &v1.PodSpec{
				Containers: []v1.Container{{
					Args: []string{
						"--es.num-shards=15",
						"--es.server-urls=" + elasticsearchUrl,
						"--es.token-file=" + k8sTokenFile,
						"--es.tls.ca=" + caPath,
						"--es.num-replicas=1",
					},
					VolumeMounts: []v1.VolumeMount{
						{Name: volumeName, ReadOnly: true, MountPath: volumeMountPath},
					},
				}},
				Volumes: []v1.Volume{{Name: "certs", VolumeSource: v1.VolumeSource{
					Secret: &v1.SecretVolumeSource{
						SecretName: "hoo-jaeger-elasticsearch"}}},
				}},
		},
	}

	for _, test := range tests {
		es := &ElasticsearchDeployment{Jaeger: v1alpha1.NewJaeger("hoo")}
		es.InjectStorageConfiguration(test.pod)
		assert.Equal(t, test.extected, test.pod)
	}

}

func TestCreateElasticsearchObjects(t *testing.T) {
	j := v1alpha1.NewJaeger("foo")
	es := &ElasticsearchDeployment{Jaeger: j}
	objs, err := es.CreateElasticsearchObjects()
	assert.Nil(t, objs)
	assert.EqualError(t, err, "failed to create Elasticsearch certificates: failed to get watch namespace: WATCH_NAMESPACE must be set")
}

func TestCalculateReplicaShards(t *testing.T) {
	tests := []struct {
		dataNodes int
		redType   esv1alpha1.RedundancyPolicyType
		shards    int
	}{
		{redType: esv1alpha1.ZeroRedundancy, dataNodes: 1, shards: 0},
		{redType: esv1alpha1.ZeroRedundancy, dataNodes: 1, shards: 0},
		{redType: esv1alpha1.SingleRedundancy, dataNodes: 1, shards: 1},
		{redType: esv1alpha1.SingleRedundancy, dataNodes: 20, shards: 1},
		{redType: esv1alpha1.MultipleRedundancy, dataNodes: 1, shards: 0},
		{redType: esv1alpha1.MultipleRedundancy, dataNodes: 20, shards: 9},
		{redType: esv1alpha1.FullRedundancy, dataNodes: 1, shards: 0},
		{redType: esv1alpha1.FullRedundancy, dataNodes: 20, shards: 19},
	}
	for _, test := range tests {
		assert.Equal(t, test.shards, calculateReplicaShards(test.redType, test.dataNodes))
	}
}
