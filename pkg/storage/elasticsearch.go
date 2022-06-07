package storage

import (
	"fmt"
	"strconv"
	"strings"

	esv1 "github.com/openshift/elasticsearch-operator/apis/logging/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	v1 "github.com/jaegertracing/jaeger-operator/apis/v1"
	"github.com/jaegertracing/jaeger-operator/pkg/util"
)

const (
	volumeName               = "certs"
	volumeMountPath          = "/certs"
	caPathESCerManagement    = volumeMountPath + "/ca-bundle.crt"
	keyPathESCertManagement  = volumeMountPath + "/tls.key"
	certPathESCertManagement = volumeMountPath + "/tls.crt"
	caPath                   = volumeMountPath + "/ca"
	keyPath                  = volumeMountPath + "/key"
	certPath                 = volumeMountPath + "/cert"
)

func (ed *ElasticsearchDeployment) getCertPath() string {
	if ed.Jaeger.Spec.Storage.Elasticsearch.UseCertManagement != nil && *ed.Jaeger.Spec.Storage.Elasticsearch.UseCertManagement {
		return certPathESCertManagement
	}
	return certPath
}

func (ed *ElasticsearchDeployment) getCertKeyPath() string {
	if ed.Jaeger.Spec.Storage.Elasticsearch.UseCertManagement != nil && *ed.Jaeger.Spec.Storage.Elasticsearch.UseCertManagement {
		return keyPathESCertManagement
	}
	return keyPath
}

func (ed *ElasticsearchDeployment) getCertCaPath() string {
	if ed.Jaeger.Spec.Storage.Elasticsearch.UseCertManagement != nil && *ed.Jaeger.Spec.Storage.Elasticsearch.UseCertManagement {
		return caPathESCerManagement
	}
	return caPath
}

// ElasticsearchDeployment represents an ES deployment for Jaeger
type ElasticsearchDeployment struct {
	Jaeger     *v1.Jaeger
	CertScript string
	Secrets    []corev1.Secret
}

func (ed *ElasticsearchDeployment) injectArguments(container *corev1.Container) {
	container.Args = append(container.Args, fmt.Sprintf("--es.server-urls=https://%s:9200", ed.Jaeger.Spec.Storage.Elasticsearch.Name))
	if util.FindItem("--es.tls=", container.Args) == "" && util.FindItem("--es.tls.enabled=", container.Args) == "" {
		container.Args = append(container.Args, "--es.tls.enabled=true")
	}
	container.Args = append(container.Args,
		"--es.tls.ca="+ed.getCertCaPath(),
		"--es.tls.cert="+ed.getCertPath(),
		"--es.tls.key="+ed.getCertKeyPath())

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
		container.Args = append(container.Args, fmt.Sprintf("--es-archive.server-urls=https://%s:9200", ed.Jaeger.Spec.Storage.Elasticsearch.Name))
		if util.FindItem("--es-archive.tls=", container.Args) == "" && util.FindItem("--es-archive.tls.enabled=", container.Args) == "" {
			container.Args = append(container.Args, "--es-archive.tls.enabled=true")
		}
		container.Args = append(container.Args,
			"--es-archive.tls.ca="+ed.getCertCaPath(),
			"--es-archive.tls.cert="+ed.getCertPath(),
			"--es-archive.tls.key="+ed.getCertKeyPath(),
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
				SecretName: jaegerESSecretName(*ed.Jaeger),
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
		p.Containers[0].Args[1] = fmt.Sprintf("https://%s:9200", ed.Jaeger.Spec.Storage.Elasticsearch.Name)
		p.Containers[0].Env = append(p.Containers[0].Env,
			corev1.EnvVar{Name: "ES_TLS_ENABLED", Value: "true"},
			corev1.EnvVar{Name: "ES_TLS_CA", Value: ed.getCertCaPath()},
			corev1.EnvVar{Name: "ES_TLS_KEY", Value: ed.getCertKeyPath()},
			corev1.EnvVar{Name: "ES_TLS_CERT", Value: ed.getCertPath()},
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
	if ed.Jaeger.Spec.Storage.Elasticsearch.DoNotProvision {
		// Do not provision ES
		// The ES instance will be reused from already provisioned one
		return nil
	}

	// this might yield names like:
	// elasticsearch-cdm-osdke2ee7864afba6854e498f316bd37347f666simpleprod-1
	// for the above value to contain at most 63 chars, our uuid has to have at most 42 chars
	uuid := strings.Replace(util.Truncate(util.DNSName(ed.Jaeger.Namespace+ed.Jaeger.Name), 42), "-", "", -1)
	var res corev1.ResourceRequirements
	if ed.Jaeger.Spec.Storage.Elasticsearch.Resources != nil {
		res = *ed.Jaeger.Spec.Storage.Elasticsearch.Resources
	}

	annotations := map[string]string{}
	if ed.Jaeger.Spec.Storage.Elasticsearch.UseCertManagement != nil && *ed.Jaeger.Spec.Storage.Elasticsearch.UseCertManagement == true {
		annotations["logging.openshift.io/elasticsearch-cert-management"] = "true"
		// The value has to match searchguard configuration
		// https://github.com/openshift/origin-aggregated-logging/blob/50126fb8e0c602e9c623d6a8599857aaf98f80f8/elasticsearch/sgconfig/roles_mapping.yml#L34
		annotations[fmt.Sprintf("logging.openshift.io/elasticsearch-cert.%s", jaegerESSecretName(*ed.Jaeger))] = "user.jaeger"
	}
	return &esv1.Elasticsearch{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: ed.Jaeger.Namespace,
			Name:      ed.Jaeger.Spec.Storage.Elasticsearch.Name,
			Labels: map[string]string{
				"app":                         "jaeger",
				"app.kubernetes.io/name":      util.Truncate(ed.Jaeger.Spec.Storage.Elasticsearch.Name, 63),
				"app.kubernetes.io/instance":  util.Truncate(ed.Jaeger.Name, 63),
				"app.kubernetes.io/component": "elasticsearch",
				"app.kubernetes.io/part-of":   "jaeger",
				// We cannot use jaeger-operator label because our controllers would try
				// to manipulate with objects created by ES operator.
				//"app.kubernetes.io/managed-by": "jaeger-operator",
			},
			Annotations:     annotations,
			OwnerReferences: []metav1.OwnerReference{util.AsOwner(ed.Jaeger)},
		},
		Spec: esv1.ElasticsearchSpec{
			ManagementState:  esv1.ManagementStateManaged,
			RedundancyPolicy: ed.Jaeger.Spec.Storage.Elasticsearch.RedundancyPolicy,
			Spec: esv1.ElasticsearchNodeSpec{
				Image:       ed.Jaeger.Spec.Storage.Elasticsearch.Image,
				Resources:   res,
				Tolerations: ed.Jaeger.Spec.Storage.Elasticsearch.Tolerations,
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

func jaegerESSecretName(jaeger v1.Jaeger) string {
	prefix := ""
	// ES cert management creates cert named jaeger-<elasticsearch-name>
	// Cert management in Jaeger creates cert named <jaeger-name>-jaeger-elasticsearch
	if jaeger.Spec.Storage.Elasticsearch.UseCertManagement == nil || !*jaeger.Spec.Storage.Elasticsearch.UseCertManagement {
		prefix = jaeger.Name + "-"
	}
	return fmt.Sprintf("%sjaeger-%s", prefix, jaeger.Spec.Storage.Elasticsearch.Name)
}
