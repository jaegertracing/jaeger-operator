package storage

import (
	"strings"

	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"

	"github.com/jaegertracing/jaeger-operator/pkg/apis/io/v1alpha1"
	esv1alpha1 "github.com/jaegertracing/jaeger-operator/pkg/storage/elasticsearch/v1alpha1"
)

const (
	// #nosec   G101: Potential hardcoded credentials (Confidence: LOW, Severity: HIGH)
	k8sTokenFile    = "/var/run/secrets/kubernetes.io/serviceaccount/token"
	volumeName      = "certs"
	volumeMountPath = "/sec"
	caCert          = volumeMountPath + "/ca"
)

func ShouldDeployElasticsearch(s v1alpha1.JaegerStorageSpec) bool {
	if strings.ToLower(s.Type) != "elasticsearch" {
		return false
	}
	_, ok := s.Options.Map()["es.server-urls"]
	if ok {
		return false
	}
	return true
}

func CreateElasticsearchObjects(j *v1alpha1.Jaeger, collector, query *v1.PodSpec) ([]runtime.Object, error) {
	err := createESCerts(certScript)
	if err != nil {
		logrus.Error("Failed to create Elasticsearch certificates: ", err)
		return nil, errors.Wrap(err, "failed to create Elasticsearch certificates")
	}
	os := []runtime.Object{}
	esSecret := createESSecrets(j)
	for _, s := range esSecret {
		os = append(os, s)
	}
	os = append(os, getESRoles(j, collector.ServiceAccountName, query.ServiceAccountName)...)
	os = append(os, createCr(j))
	inject(collector)
	inject(query)
	return os, nil
}

// TODO inject curator certs to es-index-cleaner
func inject(p *v1.PodSpec) {
	p.Volumes = append(p.Volumes, v1.Volume{
		Name: volumeName,
		VolumeSource: v1.VolumeSource{
			Secret: &v1.SecretVolumeSource{
				SecretName: "jaeger-elasticsearch",
			},
		},
	})
	// we assume jaeger containers are first
	if len(p.Containers) > 0 {
		p.Containers[0].Args = append(p.Containers[0].Args,
			"--es.server-urls=https://elasticsearch:9200",
			"--es-archive.server-urls=https://elasticsearch:9200",
			"--es.token-file="+k8sTokenFile,
			"--es-archive.token-file="+k8sTokenFile,
			"--es.tls.ca="+caCert,
			"--es-archive.tls.ca="+caCert)
		p.Containers[0].VolumeMounts = append(p.Containers[0].VolumeMounts, v1.VolumeMount{
			Name:      volumeName,
			ReadOnly:  true,
			MountPath: volumeMountPath,
		})
	}
}

func createCr(j *v1alpha1.Jaeger) *esv1alpha1.Elasticsearch {
	return &esv1alpha1.Elasticsearch{
		ObjectMeta: metav1.ObjectMeta{
			Namespace:       j.Namespace,
			Name:            "elasticsearch",
			OwnerReferences: []metav1.OwnerReference{asOwner(j)},
		},
		Spec: esv1alpha1.ElasticsearchSpec{
			Spec: esv1alpha1.ElasticsearchNodeSpec{
				// TODO remove after https://github.com/openshift/origin-aggregated-logging/pull/1500 is merged
				Image:     "pavolloffay/ecl-es:latest",
				Resources: v1.ResourceRequirements{},
			},
			ManagementState:  esv1alpha1.ManagementStateManaged,
			RedundancyPolicy: esv1alpha1.SingleRedundancy,
			Nodes: []esv1alpha1.ElasticsearchNode{
				{
					NodeCount: 1,
					Roles:     []esv1alpha1.ElasticsearchNodeRole{esv1alpha1.ElasticsearchRoleClient, esv1alpha1.ElasticsearchRoleData, esv1alpha1.ElasticsearchRoleMaster}},
			},
		},
	}
}
