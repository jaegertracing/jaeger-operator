package v1

import (
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	esv1 "github.com/jaegertracing/jaeger-operator/pkg/storage/elasticsearch/v1"
)

// IngressSecurityType represents the possible values for the security type
// +k8s:openapi-gen=true
type IngressSecurityType string

const (
	// FlagPlatformKubernetes represents the value for the 'platform' flag for Kubernetes
	// +k8s:openapi-gen=true
	FlagPlatformKubernetes = "kubernetes"

	// FlagPlatformOpenShift represents the value for the 'platform' flag for OpenShift
	// +k8s:openapi-gen=true
	FlagPlatformOpenShift = "openshift"

	// FlagPlatformAutoDetect represents the "auto-detect" value for the platform flag
	// +k8s:openapi-gen=true
	FlagPlatformAutoDetect = "auto-detect"

	// FlagProvisionElasticsearchAuto represents the 'auto' value for the 'es-provision' flag
	// +k8s:openapi-gen=true
	FlagProvisionElasticsearchAuto = "auto"

	// FlagProvisionElasticsearchTrue represents the value 'true' for the 'es-provision' flag
	// +k8s:openapi-gen=true
	FlagProvisionElasticsearchTrue = "true"

	// FlagProvisionElasticsearchFalse represents the value 'false' for the 'es-provision' flag
	// +k8s:openapi-gen=true
	FlagProvisionElasticsearchFalse = "false"

	// IngressSecurityNone disables any form of security for ingress objects (default)
	// +k8s:openapi-gen=true
	IngressSecurityNone IngressSecurityType = ""

	// IngressSecurityNoneExplicit used when the user specifically set it to 'none'
	// +k8s:openapi-gen=true
	IngressSecurityNoneExplicit IngressSecurityType = "none"

	// IngressSecurityOAuthProxy represents an OAuth Proxy as security type
	// +k8s:openapi-gen=true
	IngressSecurityOAuthProxy IngressSecurityType = "oauth-proxy"
)

// JaegerSpec defines the desired state of Jaeger
// +k8s:openapi-gen=true
type JaegerSpec struct {
	// +optional
	Strategy string `json:"strategy,omitempty"`

	// +optional
	AllInOne JaegerAllInOneSpec `json:"allInOne,omitempty"`

	// +optional
	Query JaegerQuerySpec `json:"query,omitempty"`

	// +optional
	Collector JaegerCollectorSpec `json:"collector,omitempty"`

	// +optional
	Ingester JaegerIngesterSpec `json:"ingester,omitempty"`

	// +optional
	Agent JaegerAgentSpec `json:"agent,omitempty"`

	// +optional
	UI JaegerUISpec `json:"ui,omitempty"`

	// +optional
	Sampling JaegerSamplingSpec `json:"sampling,omitempty"`

	// +optional
	Storage JaegerStorageSpec `json:"storage,omitempty"`

	// +optional
	Ingress JaegerIngressSpec `json:"ingress,omitempty"`

	// +optional
	JaegerCommonSpec `json:",inline,omitempty"`
}

// JaegerStatus defines the observed state of Jaeger
// +k8s:openapi-gen=true
type JaegerStatus struct {
	// Fill me
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// Jaeger is the Schema for the jaegers API
// +k8s:openapi-gen=true
type Jaeger struct {
	metav1.TypeMeta `json:",inline"`

	// +optional
	metav1.ObjectMeta `json:"metadata,omitempty"`

	// +optional
	Spec JaegerSpec `json:"spec,omitempty"`

	// +optional
	Status JaegerStatus `json:"status,omitempty"`
}

// JaegerCommonSpec defines the common elements used in multiple other spec structs
// +k8s:openapi-gen=true
type JaegerCommonSpec struct {
	// +optional
	Volumes []v1.Volume `json:"volumes,omitempty"`

	// +optional
	VolumeMounts []v1.VolumeMount `json:"volumeMounts,omitempty"`

	// +optional
	Annotations map[string]string `json:"annotations,omitempty"`

	// +optional
	Labels map[string]string `json:"labels,omitempty"`

	// +optional
	Resources v1.ResourceRequirements `json:"resources,omitempty"`

	// +optional
	Affinity *v1.Affinity `json:"affinity,omitempty"`

	// +optional
	Tolerations []v1.Toleration `json:"tolerations,omitempty"`

	// +optional
	SecurityContext *v1.PodSecurityContext `json:"securityContext,omitempty"`

	// +optional
	ServiceAccount string `json:"serviceAccount,omitempty"`
}

// JaegerQuerySpec defines the options to be used when deploying the query
// +k8s:openapi-gen=true
type JaegerQuerySpec struct {
	// Size represents the number of replicas to create for this service. DEPRECATED, use `Replicas` instead.
	// +optional
	Size int `json:"size,omitempty"`

	// Replicas represents the number of replicas to create for this service.
	// +optional
	Replicas *int32 `json:"replicas,omitempty"`

	// +optional
	Image string `json:"image,omitempty"`

	// +optional
	Options Options `json:"options,omitempty"`

	// +optional
	JaegerCommonSpec `json:",inline,omitempty"`
}

// JaegerUISpec defines the options to be used to configure the UI
// +k8s:openapi-gen=true
type JaegerUISpec struct {
	// +optional
	Options FreeForm `json:"options,omitempty"`
}

// JaegerSamplingSpec defines the options to be used to configure the UI
// +k8s:openapi-gen=true
type JaegerSamplingSpec struct {
	// +optional
	Options FreeForm `json:"options,omitempty"`
}

// JaegerIngressSpec defines the options to be used when deploying the query ingress
// +k8s:openapi-gen=true
type JaegerIngressSpec struct {
	// +optional
	Enabled *bool `json:"enabled,omitempty"`

	// +optional
	Security IngressSecurityType `json:"security,omitempty"`

	// +optional
	OpenShift JaegerIngressOpenShiftSpec `json:"openshift,omitempty"`

	// +optional
	JaegerCommonSpec `json:",inline,omitempty"`
}

// JaegerIngressOpenShiftSpec defines the OpenShift-specific options in the context of ingress connections,
// such as options for the OAuth Proxy
// +k8s:openapi-gen=true
type JaegerIngressOpenShiftSpec struct {
	// +optional
	SAR string `json:"sar,omitempty"`

	// +optional
	DelegateURLs string `json:"delegate-urls,omitempty"`
}

// JaegerAllInOneSpec defines the options to be used when deploying the query
// +k8s:openapi-gen=true
type JaegerAllInOneSpec struct {
	// +optional
	Image string `json:"image,omitempty"`

	// +optional
	Options Options `json:"options,omitempty"`

	// +optional
	JaegerCommonSpec `json:",inline,omitempty"`
}

// JaegerCollectorSpec defines the options to be used when deploying the collector
// +k8s:openapi-gen=true
type JaegerCollectorSpec struct {
	// Size represents the number of replicas to create for this service. DEPRECATED, use `Replicas` instead.
	// +optional
	Size int `json:"size,omitempty"`

	// Replicas represents the number of replicas to create for this service.
	// +optional
	Replicas *int32 `json:"replicas,omitempty"`

	// +optional
	Image string `json:"image,omitempty"`

	// +optional
	Options Options `json:"options,omitempty"`

	// +optional
	JaegerCommonSpec `json:",inline,omitempty"`
}

// JaegerIngesterSpec defines the options to be used when deploying the ingester
// +k8s:openapi-gen=true
type JaegerIngesterSpec struct {
	// Size represents the number of replicas to create for this service. DEPRECATED, use `Replicas` instead.
	// +optional
	Size int `json:"size,omitempty"`

	// Replicas represents the number of replicas to create for this service.
	// +optional
	Replicas *int32 `json:"replicas,omitempty"`

	// +optional
	Image string `json:"image,omitempty"`

	// +optional
	Options Options `json:"options,omitempty"`

	// +optional
	JaegerCommonSpec `json:",inline,omitempty"`
}

// JaegerAgentSpec defines the options to be used when deploying the agent
// +k8s:openapi-gen=true
type JaegerAgentSpec struct {
	// Strategy can be either 'DaemonSet' or 'Sidecar' (default)
	// +optional
	Strategy string `json:"strategy,omitempty"`

	// +optional
	Image string `json:"image,omitempty"`

	// +optional
	Options Options `json:"options,omitempty"`

	// +optional
	JaegerCommonSpec `json:",inline,omitempty"`
}

// JaegerStorageSpec defines the common storage options to be used for the query and collector
// +k8s:openapi-gen=true
type JaegerStorageSpec struct {
	// Type can be `memory` (default), `cassandra`, `elasticsearch`, `kafka` or `managed`
	// +optional
	Type string `json:"type,omitempty"`

	// +optional
	SecretName string `json:"secretName,omitempty"`

	// +optional
	Options Options `json:"options,omitempty"`

	// +optional
	CassandraCreateSchema JaegerCassandraCreateSchemaSpec `json:"cassandraCreateSchema,omitempty"`

	// +optional
	Dependencies JaegerDependenciesSpec `json:"dependencies,omitempty"`

	// +optional
	EsIndexCleaner JaegerEsIndexCleanerSpec `json:"esIndexCleaner,omitempty"`

	// +optional
	EsRollover JaegerEsRolloverSpec `json:"esRollover,omitempty"`

	// +optional
	Elasticsearch ElasticsearchSpec `json:"elasticsearch,omitempty"`
}

// ElasticsearchSpec represents the ES configuration options that we pass down to the Elasticsearch operator
// +k8s:openapi-gen=true
type ElasticsearchSpec struct {
	// +optional
	Image string `json:"image,omitempty"`

	// +optional
	Resources v1.ResourceRequirements `json:"resources,omitempty"`

	// +optional
	NodeCount int32 `json:"nodeCount,omitempty"`

	// +optional
	NodeSelector map[string]string `json:"nodeSelector,omitempty"`

	// +optional
	Storage esv1.ElasticsearchStorageSpec `json:"storage,omitempty"`

	// +optional
	RedundancyPolicy esv1.RedundancyPolicyType `json:"redundancyPolicy,omitempty"`
}

// JaegerCassandraCreateSchemaSpec holds the options related to the create-schema batch job
// +k8s:openapi-gen=true
type JaegerCassandraCreateSchemaSpec struct {
	// +optional
	Enabled *bool `json:"enabled,omitempty"`

	// +optional
	Image string `json:"image,omitempty"`

	// +optional
	Datacenter string `json:"datacenter,omitempty"`

	// +optional
	Mode string `json:"mode,omitempty"`

	// +optional
	TTLSecondsAfterFinished *int32 `json:"ttlSecondsAfterFinished,omitempty"`
}

// JaegerDependenciesSpec defined options for running spark-dependencies.
// +k8s:openapi-gen=true
type JaegerDependenciesSpec struct {
	// +optional
	Enabled *bool `json:"enabled,omitempty"`

	// +optional
	SparkMaster string `json:"sparkMaster,omitempty"`

	// +optional
	Schedule string `json:"schedule,omitempty"`

	// +optional
	Image string `json:"image,omitempty"`

	// +optional
	JavaOpts string `json:"javaOpts,omitempty"`

	// +optional
	CassandraClientAuthEnabled bool `json:"cassandraClientAuthEnabled,omitempty"`

	// +optional
	ElasticsearchClientNodeOnly bool `json:"elasticsearchClientNodeOnly,omitempty"`

	// +optional
	ElasticsearchNodesWanOnly bool `json:"elasticsearchNodesWanOnly,omitempty"`

	// +optional
	TTLSecondsAfterFinished *int32 `json:"ttlSecondsAfterFinished,omitempty"`
}

// JaegerEsIndexCleanerSpec holds the options related to es-index-cleaner
// +k8s:openapi-gen=true
type JaegerEsIndexCleanerSpec struct {
	// +optional
	Enabled *bool `json:"enabled,omitempty"`

	// +optional
	NumberOfDays *int `json:"numberOfDays,omitempty"`

	// +optional
	Schedule string `json:"schedule,omitempty"`

	// +optional
	Image string `json:"image,omitempty"`

	// +optional
	TTLSecondsAfterFinished *int32 `json:"ttlSecondsAfterFinished,omitempty"`
}

// JaegerEsRolloverSpec holds the options related to es-rollover
type JaegerEsRolloverSpec struct {
	// +optional
	Image string `json:"image,omitempty"`

	// +optional
	Schedule string `json:"schedule,omitempty"`

	// +optional
	Conditions string `json:"conditions,omitempty"`

	// +optional
	TTLSecondsAfterFinished *int32 `json:"ttlSecondsAfterFinished,omitempty"`

	// we parse it with time.ParseDuration
	// +optional
	ReadTTL string `json:"readTTL,omitempty"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// JaegerList contains a list of Jaeger
type JaegerList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []Jaeger `json:"items"`
}

func init() {
	SchemeBuilder.Register(&Jaeger{}, &JaegerList{})
}
