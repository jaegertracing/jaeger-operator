package storage

import (
	"strings"

	"k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/jaegertracing/jaeger-operator/pkg/apis/io/v1alpha1"
	esv1alpha1 "github.com/jaegertracing/jaeger-operator/pkg/storage/elasticsearch/v1alpha1"
)

const (
	// #nosec   G101: Potential hardcoded credentials (Confidence: LOW, Severity: HIGH)
	k8sTokenFile     = "/var/run/secrets/kubernetes.io/serviceaccount/token"
	volumeName       = "certs"
	volumeMountPath  = "/certs"
	caPath           = volumeMountPath + "/ca"
	keyPath          = volumeMountPath + "/key"
	certPath         = volumeMountPath + "/cert"
	elasticsearchUrl = "https://elasticsearch:9200"
)

func ShouldDeployElasticsearch(s v1alpha1.JaegerStorageSpec) bool {
	if !strings.EqualFold(s.Type, "elasticsearch") {
		return false
	}
	_, ok := s.Options.Map()["es.server-urls"]
	return !ok
}

type ElasticsearchDeployment struct {
	Jaeger *v1alpha1.Jaeger
}

func (ed *ElasticsearchDeployment) InjectStorageConfiguration(p *v1.PodSpec) {
	p.Volumes = append(p.Volumes, v1.Volume{
		Name: volumeName,
		VolumeSource: v1.VolumeSource{
			Secret: &v1.SecretVolumeSource{
				SecretName: secretName(ed.Jaeger.Name, jaegerSecret.name),
			},
		},
	})
	// we assume jaeger containers are first
	if len(p.Containers) > 0 {
		// TODO add to archive storage if it is enabled?
		p.Containers[0].Args = append(p.Containers[0].Args,
			"--es.server-urls="+elasticsearchUrl,
			"--es.token-file="+k8sTokenFile,
			"--es.tls.ca="+caPath)
		p.Containers[0].VolumeMounts = append(p.Containers[0].VolumeMounts, v1.VolumeMount{
			Name:      volumeName,
			ReadOnly:  true,
			MountPath: volumeMountPath,
		})
	}
}

func (ed *ElasticsearchDeployment) InjectIndexCleanerConfiguration(p *v1.PodSpec) {
	p.Volumes = append(p.Volumes, v1.Volume{
		Name: volumeName,
		VolumeSource: v1.VolumeSource{
			Secret: &v1.SecretVolumeSource{
				SecretName: secretName(ed.Jaeger.Name, curatorSecret.name),
			},
		},
	})
	// we assume jaeger containers are first
	if len(p.Containers) > 0 {
		// the size of arguments array should be always 2
		p.Containers[0].Args[1] = elasticsearchUrl
		p.Containers[0].Env = append(p.Containers[0].Env,
			v1.EnvVar{Name: "ES_TLS", Value: "true"},
			v1.EnvVar{Name: "ES_TLS_CA", Value: caPath},
			v1.EnvVar{Name: "ES_TLS_KEY", Value: keyPath},
			v1.EnvVar{Name: "ES_TLS_CERT", Value: certPath},
		)
		p.Containers[0].VolumeMounts = append(p.Containers[0].VolumeMounts, v1.VolumeMount{
			Name:      volumeName,
			ReadOnly:  true,
			MountPath: volumeMountPath,
		})
	}
}

func (ed *ElasticsearchDeployment) createCr() *esv1alpha1.Elasticsearch {
	return &esv1alpha1.Elasticsearch{
		ObjectMeta: metav1.ObjectMeta{
			Namespace:       ed.Jaeger.Namespace,
			Name:            esSecret.name,
			OwnerReferences: []metav1.OwnerReference{asOwner(ed.Jaeger)},
		},
		Spec: esv1alpha1.ElasticsearchSpec{
			ManagementState:  esv1alpha1.ManagementStateManaged,
			RedundancyPolicy: ed.Jaeger.Spec.Storage.Elasticsearch.RedundancyPolicy,
			Spec: esv1alpha1.ElasticsearchNodeSpec{
				Resources: ed.Jaeger.Spec.Storage.Elasticsearch.Resources,
			},
			Nodes: []esv1alpha1.ElasticsearchNode{
				{
					NodeCount:    ed.Jaeger.Spec.Storage.Elasticsearch.NodeCount,
					Storage:      ed.Jaeger.Spec.Storage.Elasticsearch.Storage,
					NodeSelector: ed.Jaeger.Spec.Storage.Elasticsearch.NodeSelector,
					Roles:        []esv1alpha1.ElasticsearchNodeRole{esv1alpha1.ElasticsearchRoleClient, esv1alpha1.ElasticsearchRoleData, esv1alpha1.ElasticsearchRoleMaster},
				},
			},
		},
	}
}
