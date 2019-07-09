package storage

import (
	"testing"

	"github.com/stretchr/testify/assert"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"

	v1 "github.com/jaegertracing/jaeger-operator/pkg/apis/jaegertracing/v1"
	esv1 "github.com/jaegertracing/jaeger-operator/pkg/storage/elasticsearch/v1"
)

func TestShouldDeployElasticsearch(t *testing.T) {
	tests := []struct {
		j        v1.JaegerStorageSpec
		expected bool
	}{
		{j: v1.JaegerStorageSpec{}},
		{j: v1.JaegerStorageSpec{Type: "cassandra"}},
		{j: v1.JaegerStorageSpec{Type: "elasticsearch", Options: v1.NewOptions(map[string]interface{}{"es.server-urls": "foo"})}},
		{j: v1.JaegerStorageSpec{Type: "elasticsearch"}, expected: true},
	}
	for _, test := range tests {
		assert.Equal(t, test.expected, ShouldDeployElasticsearch(test.j))
	}
}

func TestCreateElasticsearchCR(t *testing.T) {
	storageClassName := "floppydisk"
	genuuid1 := "myprojectfoo"
	genuuidmaster1 := "myprojectfoomaster"
	genuuid2 := "myprojectfoobar"
	genuuidmaster2 := "myprojectfoobarmaster"
	tests := []struct {
		name      string
		namespace string
		jEsSpec   v1.ElasticsearchSpec
		esSpec    esv1.ElasticsearchSpec
	}{
		{
			name:      "foo",
			namespace: "myproject",
			jEsSpec: v1.ElasticsearchSpec{
				NodeCount:        2,
				RedundancyPolicy: esv1.FullRedundancy,
				Storage: esv1.ElasticsearchStorageSpec{
					StorageClassName: &storageClassName,
				},
			},
			esSpec: esv1.ElasticsearchSpec{
				ManagementState:  esv1.ManagementStateManaged,
				RedundancyPolicy: esv1.FullRedundancy,
				Spec:             esv1.ElasticsearchNodeSpec{},
				Nodes: []esv1.ElasticsearchNode{
					{
						NodeCount: 2,
						Storage:   esv1.ElasticsearchStorageSpec{StorageClassName: &storageClassName},
						Roles:     []esv1.ElasticsearchNodeRole{esv1.ElasticsearchRoleClient, esv1.ElasticsearchRoleData, esv1.ElasticsearchRoleMaster},
						GenUUID:   &genuuid1,
					},
				},
			},
		},
		{
			name:      "foo",
			namespace: "myproject",
			jEsSpec: v1.ElasticsearchSpec{
				NodeCount:        5,
				RedundancyPolicy: esv1.FullRedundancy,
				Storage: esv1.ElasticsearchStorageSpec{
					StorageClassName: &storageClassName,
				},
			},
			esSpec: esv1.ElasticsearchSpec{
				ManagementState:  esv1.ManagementStateManaged,
				RedundancyPolicy: esv1.FullRedundancy,
				Spec:             esv1.ElasticsearchNodeSpec{},
				Nodes: []esv1.ElasticsearchNode{
					{
						NodeCount: 3,
						Storage:   esv1.ElasticsearchStorageSpec{StorageClassName: &storageClassName},
						Roles:     []esv1.ElasticsearchNodeRole{esv1.ElasticsearchRoleMaster},
						GenUUID:   &genuuidmaster1,
					},
					{
						NodeCount: 2,
						Storage:   esv1.ElasticsearchStorageSpec{StorageClassName: &storageClassName},
						Roles:     []esv1.ElasticsearchNodeRole{esv1.ElasticsearchRoleClient, esv1.ElasticsearchRoleData},
						GenUUID:   &genuuid1,
					},
				},
			},
		},
		{
			name:      "foo-ba%r",
			namespace: "myproje&ct",
			jEsSpec: v1.ElasticsearchSpec{
				NodeCount:        5,
				RedundancyPolicy: esv1.FullRedundancy,
				Storage: esv1.ElasticsearchStorageSpec{
					StorageClassName: &storageClassName,
				},
			},
			esSpec: esv1.ElasticsearchSpec{
				ManagementState:  esv1.ManagementStateManaged,
				RedundancyPolicy: esv1.FullRedundancy,
				Spec:             esv1.ElasticsearchNodeSpec{},
				Nodes: []esv1.ElasticsearchNode{
					{
						NodeCount: 3,
						Storage:   esv1.ElasticsearchStorageSpec{StorageClassName: &storageClassName},
						Roles:     []esv1.ElasticsearchNodeRole{esv1.ElasticsearchRoleMaster},
						GenUUID:   &genuuidmaster2,
					},
					{
						NodeCount: 2,
						Storage:   esv1.ElasticsearchStorageSpec{StorageClassName: &storageClassName},
						Roles:     []esv1.ElasticsearchNodeRole{esv1.ElasticsearchRoleClient, esv1.ElasticsearchRoleData},
						GenUUID:   &genuuid2,
					},
				},
			},
		},
	}
	for _, test := range tests {
		j := v1.NewJaeger(types.NamespacedName{Name: test.name, Namespace: test.namespace})
		j.Spec.Storage.Elasticsearch = test.jEsSpec
		es := &ElasticsearchDeployment{Jaeger: j}
		cr := es.Elasticsearch()
		assert.Equal(t, test.namespace, cr.Namespace)
		assert.Equal(t, "elasticsearch", cr.Name)
		trueVar := true
		assert.Equal(t, []metav1.OwnerReference{{Name: test.name, Controller: &trueVar}}, cr.OwnerReferences)
		assert.Equal(t, cr.Spec, test.esSpec)
	}
}

func TestInject(t *testing.T) {
	tests := []struct {
		pod      *corev1.PodSpec
		expected *corev1.PodSpec
		es       v1.ElasticsearchSpec
	}{
		{pod: &corev1.PodSpec{
			Containers: []corev1.Container{{
				Args:         []string{"foo"},
				VolumeMounts: []corev1.VolumeMount{{Name: "lol"}},
			}},
		},
			expected: &corev1.PodSpec{
				Containers: []corev1.Container{{
					Args: []string{
						"foo",
						"--es.server-urls=" + elasticsearchURL,
						"--es.tls=true",
						"--es.tls.ca=" + caPath,
						"--es.tls.cert=" + certPath,
						"--es.tls.key=" + keyPath,
						"--es.timeout=15s",
						"--es.num-shards=0",
						"--es.num-replicas=1",
					},
					VolumeMounts: []corev1.VolumeMount{
						{Name: "lol"},
						{Name: volumeName, ReadOnly: true, MountPath: volumeMountPath},
					},
				}},
				Volumes: []corev1.Volume{{Name: "certs", VolumeSource: corev1.VolumeSource{
					Secret: &corev1.SecretVolumeSource{
						SecretName: "hoo-jaeger-elasticsearch"}}},
				}},
		},
		{pod: &corev1.PodSpec{
			Containers: []corev1.Container{{
				Args: []string{"--es.num-shards=15", "--es.num-replicas=55", "--es.timeout=99s"},
			}},
		},
			expected: &corev1.PodSpec{
				Containers: []corev1.Container{{
					Args: []string{
						"--es.num-shards=15",
						"--es.num-replicas=55",
						"--es.timeout=99s",
						"--es.server-urls=" + elasticsearchURL,
						"--es.tls=true",
						"--es.tls.ca=" + caPath,
						"--es.tls.cert=" + certPath,
						"--es.tls.key=" + keyPath,
					},
					VolumeMounts: []corev1.VolumeMount{
						{Name: volumeName, ReadOnly: true, MountPath: volumeMountPath},
					},
				}},
				Volumes: []corev1.Volume{{Name: "certs", VolumeSource: corev1.VolumeSource{
					Secret: &corev1.SecretVolumeSource{
						SecretName: "hoo-jaeger-elasticsearch"}}},
				}},
		},
		{
			pod: &corev1.PodSpec{Containers: []corev1.Container{{}}},
			es:  v1.ElasticsearchSpec{NodeCount: 15, RedundancyPolicy: esv1.FullRedundancy},
			expected: &corev1.PodSpec{
				Containers: []corev1.Container{{
					Args: []string{
						"--es.server-urls=" + elasticsearchURL,
						"--es.tls=true",
						"--es.tls.ca=" + caPath,
						"--es.tls.cert=" + certPath,
						"--es.tls.key=" + keyPath,
						"--es.timeout=15s",
						"--es.num-shards=12",
						"--es.num-replicas=11",
					},
					VolumeMounts: []corev1.VolumeMount{
						{Name: volumeName, ReadOnly: true, MountPath: volumeMountPath},
					},
				}},
				Volumes: []corev1.Volume{{Name: "certs", VolumeSource: corev1.VolumeSource{
					Secret: &corev1.SecretVolumeSource{
						SecretName: "hoo-jaeger-elasticsearch"}}},
				}},
		},
		{
			pod: &corev1.PodSpec{Containers: []corev1.Container{{Args: []string{"--es-archive.enabled=true"}}}},
			es:  v1.ElasticsearchSpec{NodeCount: 15, RedundancyPolicy: esv1.FullRedundancy},
			expected: &corev1.PodSpec{
				Containers: []corev1.Container{{
					Args: []string{
						"--es-archive.enabled=true",
						"--es.server-urls=" + elasticsearchURL,
						"--es.tls=true",
						"--es.tls.ca=" + caPath,
						"--es.tls.cert=" + certPath,
						"--es.tls.key=" + keyPath,
						"--es.timeout=15s",
						"--es.num-shards=12",
						"--es.num-replicas=11",
						"--es-archive.server-urls=" + elasticsearchURL,
						"--es-archive.tls=true",
						"--es-archive.tls.ca=" + caPath,
						"--es-archive.tls.cert=" + certPath,
						"--es-archive.tls.key=" + keyPath,
						"--es-archive.timeout=15s",
						"--es-archive.num-shards=12",
						"--es-archive.num-replicas=11",
					},
					VolumeMounts: []corev1.VolumeMount{
						{Name: volumeName, ReadOnly: true, MountPath: volumeMountPath},
					},
				}},
				Volumes: []corev1.Volume{{Name: "certs", VolumeSource: corev1.VolumeSource{
					Secret: &corev1.SecretVolumeSource{
						SecretName: "hoo-jaeger-elasticsearch"}}},
				}},
		},
	}

	for _, test := range tests {
		es := &ElasticsearchDeployment{Jaeger: v1.NewJaeger(types.NamespacedName{Name: "hoo"})}
		es.Jaeger.Spec.Storage.Elasticsearch = test.es
		es.InjectStorageConfiguration(test.pod)
		assert.Equal(t, test.expected, test.pod)
	}

}

func TestCalculateReplicaShards(t *testing.T) {
	tests := []struct {
		dataNodes int
		redType   esv1.RedundancyPolicyType
		shards    int
	}{
		{redType: esv1.ZeroRedundancy, dataNodes: 1, shards: 0},
		{redType: esv1.ZeroRedundancy, dataNodes: 1, shards: 0},
		{redType: esv1.SingleRedundancy, dataNodes: 1, shards: 1},
		{redType: esv1.SingleRedundancy, dataNodes: 20, shards: 1},
		{redType: esv1.MultipleRedundancy, dataNodes: 1, shards: 0},
		{redType: esv1.MultipleRedundancy, dataNodes: 20, shards: 9},
		{redType: esv1.FullRedundancy, dataNodes: 1, shards: 0},
		{redType: esv1.FullRedundancy, dataNodes: 20, shards: 19},
	}
	for _, test := range tests {
		assert.Equal(t, test.shards, calculateReplicaShards(test.redType, test.dataNodes))
	}
}
