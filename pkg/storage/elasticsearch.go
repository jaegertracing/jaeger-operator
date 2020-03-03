package storage

import (
	"fmt"
	"strconv"
	"strings"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	v1 "github.com/jaegertracing/jaeger-operator/pkg/apis/jaegertracing/v1"
	esv1 "github.com/jaegertracing/jaeger-operator/pkg/storage/elasticsearch/v1"
	"github.com/jaegertracing/jaeger-operator/pkg/util"
)

const (
	volumeName       = "certs"
	volumeMountPath  = "/certs"
	caPath           = volumeMountPath + "/ca"
	keyPath          = volumeMountPath + "/key"
	certPath         = volumeMountPath + "/cert"
	elasticsearchURL = "https://elasticsearch:9200"
)

// ShouldDeployElasticsearch determines whether a new instance of Elasticsearch should be deployed
func ShouldDeployElasticsearch(s v1.JaegerStorageSpec) bool {
	if !strings.EqualFold(s.Type, "elasticsearch") {
		return false
	}
	_, ok := s.Options.Map()["es.server-urls"]
	return !ok
}

// ElasticsearchDeployment represents an ES deployment for Jaeger
type ElasticsearchDeployment struct {
	Jaeger     *v1.Jaeger
	CertScript string
	Secrets    []corev1.Secret
}

func (ed *ElasticsearchDeployment) injectArguments(container *corev1.Container) {
	container.Args = append(container.Args, "--es.server-urls="+elasticsearchURL)
	if util.FindItem("--es.tls=", container.Args) == "" && util.FindItem("--es.tls.enabled=", container.Args) == "" {
		container.Args = append(container.Args, "--es.tls.enabled=true")
	}
	container.Args = append(container.Args,
		"--es.tls.ca="+caPath,
		"--es.tls.cert="+certPath,
		"--es.tls.key="+keyPath)

	if util.FindItem("--es.timeout", container.Args) == "" {
		container.Args = append(container.Args, "--es.timeout=15s")
	}
	if util.FindItem("--es.num-shards", container.Args) == "" {
		// taken from https://github.com/openshift/cluster-logging-operator/blob/32b69e8bcf61a805e8f3c45c664a3c08d1ee62d5/vendor/github.com/openshift/elasticsearch-operator/pkg/k8shandler/configmaps.go#L38
		// every ES node is a data node
		container.Args = append(container.Args, fmt.Sprintf("--es.num-shards=%d", ed.Jaeger.Spec.Storage.Elasticsearch.NodeCount))
	}
	if util.FindItem("--es.num-replicas", container.Args) == "" {
		container.Args = append(container.Args, fmt.Sprintf("--es.num-replicas=%d",
			calculateReplicaShards(ed.Jaeger.Spec.Storage.Elasticsearch.RedundancyPolicy, int(ed.Jaeger.Spec.Storage.Elasticsearch.NodeCount))))
	}
	if strings.EqualFold(util.FindItem("--es-archive.enabled", container.Args), "--es-archive.enabled=true") {
		container.Args = append(container.Args,
			"--es-archive.server-urls="+elasticsearchURL,
		)
		if util.FindItem("--es-archive.tls=", container.Args) == "" && util.FindItem("--es-archive.tls.enabled=", container.Args) == "" {
			container.Args = append(container.Args, "--es-archive.tls.enabled=true")
		}
		container.Args = append(container.Args,
			"--es-archive.tls.ca="+caPath,
			"--es-archive.tls.cert="+certPath,
			"--es-archive.tls.key="+keyPath,
		)
		if util.FindItem("--es-archive.timeout", container.Args) == "" {
			container.Args = append(container.Args, "--es-archive.timeout=15s")
		}
		if util.FindItem("--es-archive.num-shards", container.Args) == "" {
			// taken from https://github.com/openshift/cluster-logging-operator/blob/32b69e8bcf61a805e8f3c45c664a3c08d1ee62d5/vendor/github.com/openshift/elasticsearch-operator/pkg/k8shandler/configmaps.go#L38
			// every ES node is a data node
			container.Args = append(container.Args, fmt.Sprintf("--es-archive.num-shards=%d", ed.Jaeger.Spec.Storage.Elasticsearch.NodeCount))
		}
		if util.FindItem("--es-archive.num-replicas", container.Args) == "" {
			container.Args = append(container.Args, fmt.Sprintf("--es-archive.num-replicas=%d",
				calculateReplicaShards(ed.Jaeger.Spec.Storage.Elasticsearch.RedundancyPolicy, int(ed.Jaeger.Spec.Storage.Elasticsearch.NodeCount))))
		}
	}
}

// InjectStorageConfiguration changes the given spec to include ES-related command line options
func (ed *ElasticsearchDeployment) InjectStorageConfiguration(p *corev1.PodSpec) {
	p.Volumes = append(p.Volumes, corev1.Volume{
		Name: volumeName,
		VolumeSource: corev1.VolumeSource{
			Secret: &corev1.SecretVolumeSource{
				SecretName: jaegerSecret.instanceName(ed.Jaeger),
			},
		},
	})
	// we assume jaeger containers are first
	if len(p.Containers) > 0 {
		ed.injectArguments(&p.Containers[0])
		p.Containers[0].VolumeMounts = append(p.Containers[0].VolumeMounts, corev1.VolumeMount{
			Name:      volumeName,
			ReadOnly:  true,
			MountPath: volumeMountPath,
		})
	}
}

// InjectSecretsConfiguration changes the given spec to include the options for the index cleaner
func (ed *ElasticsearchDeployment) InjectSecretsConfiguration(p *corev1.PodSpec) {
	p.Volumes = append(p.Volumes, corev1.Volume{
		Name: volumeName,
		VolumeSource: corev1.VolumeSource{
			Secret: &corev1.SecretVolumeSource{
				SecretName: curatorSecret.instanceName(ed.Jaeger),
			},
		},
	})
	// we assume jaeger containers are first
	if len(p.Containers) > 0 {
		// the size of arguments array should be always 2
		p.Containers[0].Args[1] = elasticsearchURL
		p.Containers[0].Env = append(p.Containers[0].Env,
			corev1.EnvVar{Name: "ES_TLS", Value: "true"},
			corev1.EnvVar{Name: "ES_TLS_CA", Value: caPath},
			corev1.EnvVar{Name: "ES_TLS_KEY", Value: keyPath},
			corev1.EnvVar{Name: "ES_TLS_CERT", Value: certPath},
			corev1.EnvVar{Name: "SHARDS", Value: strconv.Itoa(int(ed.Jaeger.Spec.Storage.Elasticsearch.NodeCount))},
			corev1.EnvVar{Name: "REPLICAS", Value: strconv.Itoa(calculateReplicaShards(ed.Jaeger.Spec.Storage.Elasticsearch.RedundancyPolicy, int(ed.Jaeger.Spec.Storage.Elasticsearch.NodeCount)))},
		)
		p.Containers[0].VolumeMounts = append(p.Containers[0].VolumeMounts, corev1.VolumeMount{
			Name:      volumeName,
			ReadOnly:  true,
			MountPath: volumeMountPath,
		})
	}
}

// Elasticsearch returns an ES CR for the deployment
func (ed *ElasticsearchDeployment) Elasticsearch() *esv1.Elasticsearch {
	// this might yield names like:
	// elasticsearch-cdm-osdke2ee7864afba6854e498f316bd37347f666simpleprod-1
	// for the above value to contain at most 63 chars, our uuid has to have at most 42 chars
	uuid := strings.Replace(util.Truncate(util.DNSName(ed.Jaeger.Namespace+ed.Jaeger.Name), 42), "-", "", -1)
	var res corev1.ResourceRequirements
	if ed.Jaeger.Spec.Storage.Elasticsearch.Resources != nil {
		res = *ed.Jaeger.Spec.Storage.Elasticsearch.Resources
	}
	return &esv1.Elasticsearch{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: ed.Jaeger.Namespace,
			Name:      esSecret.name,
			Labels: map[string]string{
				"app":                         "jaeger",
				"app.kubernetes.io/name":      util.Truncate(esSecret.name, 63),
				"app.kubernetes.io/instance":  util.Truncate(ed.Jaeger.Name, 63),
				"app.kubernetes.io/component": "elasticsearch",
				"app.kubernetes.io/part-of":   "jaeger",
				// We cannot use jaeger-operator label because our controllers would try
				// to manipulate with objects created by ES operator.
				//"app.kubernetes.io/managed-by": "jaeger-operator",
			},
			OwnerReferences: []metav1.OwnerReference{util.AsOwner(ed.Jaeger)},
		},
		Spec: esv1.ElasticsearchSpec{
			ManagementState:  esv1.ManagementStateManaged,
			RedundancyPolicy: ed.Jaeger.Spec.Storage.Elasticsearch.RedundancyPolicy,
			Spec: esv1.ElasticsearchNodeSpec{
				Image:     ed.Jaeger.Spec.Storage.Elasticsearch.Image,
				Resources: res,
			},
			Nodes: getNodes(uuid, ed.Jaeger.Spec.Storage.Elasticsearch),
		},
	}
}

func getNodes(uuid string, es v1.ElasticsearchSpec) []esv1.ElasticsearchNode {
	if es.NodeCount <= 3 {
		return []esv1.ElasticsearchNode{
			{
				NodeCount:    es.NodeCount,
				Storage:      es.Storage,
				NodeSelector: es.NodeSelector,
				Roles:        []esv1.ElasticsearchNodeRole{esv1.ElasticsearchRoleMaster, esv1.ElasticsearchRoleClient, esv1.ElasticsearchRoleData},
				GenUUID:      &uuid,
			},
		}
	}
	genuuidmaster := uuid + "master"
	return []esv1.ElasticsearchNode{
		{
			NodeCount:    3,
			Storage:      es.Storage,
			NodeSelector: es.NodeSelector,
			Roles:        []esv1.ElasticsearchNodeRole{esv1.ElasticsearchRoleMaster, esv1.ElasticsearchRoleClient, esv1.ElasticsearchRoleData},
			GenUUID:      &genuuidmaster,
		},
		{
			NodeCount:    es.NodeCount - 3,
			Storage:      es.Storage,
			NodeSelector: es.NodeSelector,
			Roles:        []esv1.ElasticsearchNodeRole{esv1.ElasticsearchRoleClient, esv1.ElasticsearchRoleData},
			GenUUID:      &uuid,
		},
	}
}

// taken from https://github.com/openshift/cluster-logging-operator/blob/1ead6701c7c7af9c0578aa66597261079b2781d5/vendor/github.com/openshift/elasticsearch-operator/pkg/k8shandler/defaults.go#L33
func calculateReplicaShards(policyType esv1.RedundancyPolicyType, dataNodes int) int {
	switch policyType {
	case esv1.FullRedundancy:
		return dataNodes - 1
	case esv1.MultipleRedundancy:
		return (dataNodes - 1) / 2
	case esv1.SingleRedundancy:
		return 1
	case esv1.ZeroRedundancy:
		return 0
	default:
		return 1
	}
}
