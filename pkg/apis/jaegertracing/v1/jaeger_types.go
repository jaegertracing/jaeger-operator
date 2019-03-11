package v1

import (
	"k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/jaegertracing/jaeger-operator/pkg/storage/elasticsearch/v1alpha1"
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
	Strategy  string              `json:"strategy"`
	AllInOne  JaegerAllInOneSpec  `json:"allInOne"`
	Query     JaegerQuerySpec     `json:"query"`
	Collector JaegerCollectorSpec `json:"collector"`
	Ingester  JaegerIngesterSpec  `json:"ingester"`
	Agent     JaegerAgentSpec     `json:"agent"`
	UI        JaegerUISpec        `json:"ui"`
	Sampling  JaegerSamplingSpec  `json:"sampling"`
	Storage   JaegerStorageSpec   `json:"storage"`
	Ingress   JaegerIngressSpec   `json:"ingress"`
	JaegerCommonSpec
}

// JaegerStatus defines the observed state of Jaeger
// +k8s:openapi-gen=true
type JaegerStatus struct {
	// CollectorSpansReceived represents sum of the metric jaeger_collector_spans_received_total across all collectors
	CollectorSpansReceived int `json:"collectorSpansReceived"`

	// CollectorSpansDropped represents sum of the metric jaeger_collector_spans_dropped_total across all collectors
	CollectorSpansDropped int `json:"collectorSpansDropped"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// Jaeger is the Schema for the jaegers API
// +k8s:openapi-gen=true
type Jaeger struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   JaegerSpec   `json:"spec,omitempty"`
	Status JaegerStatus `json:"status,omitempty"`
}

// JaegerCommonSpec defines the common elements used in multiple other spec structs
// +k8s:openapi-gen=true
type JaegerCommonSpec struct {
	Volumes      []v1.Volume             `json:"volumes"`
	VolumeMounts []v1.VolumeMount        `json:"volumeMounts"`
	Annotations  map[string]string       `json:"annotations,omitempty"`
	Resources    v1.ResourceRequirements `json:"resources,omitempty"`
}

// JaegerQuerySpec defines the options to be used when deploying the query
// +k8s:openapi-gen=true
type JaegerQuerySpec struct {
	// Size represents the number of replicas to create for this service. DEPRECATED, use `Replicas` instead.
	Size int `json:"size"`

	// Replicas represents the number of replicas to create for this service.
	Replicas *int32 `json:"replicas"`

	Image   string  `json:"image"`
	Options Options `json:"options"`
	JaegerCommonSpec
}

// JaegerUISpec defines the options to be used to configure the UI
// +k8s:openapi-gen=true
type JaegerUISpec struct {
	Options FreeForm `json:"options"`
}

// JaegerSamplingSpec defines the options to be used to configure the UI
// +k8s:openapi-gen=true
type JaegerSamplingSpec struct {
	Options FreeForm `json:"options"`
}

// JaegerIngressSpec defines the options to be used when deploying the query ingress
// +k8s:openapi-gen=true
type JaegerIngressSpec struct {
	Enabled  *bool               `json:"enabled"`
	Security IngressSecurityType `json:"security"`
	JaegerCommonSpec
}

// JaegerAllInOneSpec defines the options to be used when deploying the query
// +k8s:openapi-gen=true
type JaegerAllInOneSpec struct {
	Image   string  `json:"image"`
	Options Options `json:"options"`
	JaegerCommonSpec
}

// JaegerCollectorSpec defines the options to be used when deploying the collector
// +k8s:openapi-gen=true
type JaegerCollectorSpec struct {
	// Size represents the number of replicas to create for this service. DEPRECATED, use `Replicas` instead.
	Size int `json:"size"`

	// Replicas represents the number of replicas to create for this service.
	Replicas *int32 `json:"replicas"`

	Image   string  `json:"image"`
	Options Options `json:"options"`
	JaegerCommonSpec
}

// JaegerIngesterSpec defines the options to be used when deploying the ingester
// +k8s:openapi-gen=true
type JaegerIngesterSpec struct {
	// Size represents the number of replicas to create for this service. DEPRECATED, use `Replicas` instead.
	Size int `json:"size"`

	// Replicas represents the number of replicas to create for this service.
	Replicas *int32 `json:"replicas"`

	Image   string  `json:"image"`
	Options Options `json:"options"`
	JaegerCommonSpec
}

// JaegerAgentSpec defines the options to be used when deploying the agent
// +k8s:openapi-gen=true
type JaegerAgentSpec struct {
	Strategy string  `json:"strategy"` // can be either 'DaemonSet' or 'Sidecar' (default)
	Image    string  `json:"image"`
	Options  Options `json:"options"`
	JaegerCommonSpec
}

// JaegerStorageSpec defines the common storage options to be used for the query and collector
// +k8s:openapi-gen=true
type JaegerStorageSpec struct {
	Type                  string                          `json:"type"` // can be `memory` (default), `cassandra`, `elasticsearch`, `kafka` or `managed`
	SecretName            string                          `json:"secretName"`
	Options               Options                         `json:"options"`
	CassandraCreateSchema JaegerCassandraCreateSchemaSpec `json:"cassandraCreateSchema"`
	SparkDependencies     JaegerDependenciesSpec          `json:"dependencies"`
	EsIndexCleaner        JaegerEsIndexCleanerSpec        `json:"esIndexCleaner"`
	Rollover              JaegerEsRolloverSpec            `json:"esRollover"`
	Elasticsearch         ElasticsearchSpec               `json:"elasticsearch"`
}

// ElasticsearchSpec represents the ES configuration options that we pass down to the Elasticsearch operator
// +k8s:openapi-gen=true
type ElasticsearchSpec struct {
	Image            string                            `json:"image"`
	Resources        v1.ResourceRequirements           `json:"resources"`
	NodeCount        int32                             `json:"nodeCount"`
	NodeSelector     map[string]string                 `json:"nodeSelector,omitempty"`
	Storage          v1alpha1.ElasticsearchStorageSpec `json:"storage"`
	RedundancyPolicy v1alpha1.RedundancyPolicyType     `json:"redundancyPolicy"`
}

// JaegerCassandraCreateSchemaSpec holds the options related to the create-schema batch job
// +k8s:openapi-gen=true
type JaegerCassandraCreateSchemaSpec struct {
	Enabled    *bool  `json:"enabled"`
	Image      string `json:"image"`
	Datacenter string `json:"datacenter"`
	Mode       string `json:"mode"`
}

// JaegerDependenciesSpec defined options for running spark-dependencies.
// +k8s:openapi-gen=true
type JaegerDependenciesSpec struct {
	Enabled                     *bool  `json:"enabled"`
	SparkMaster                 string `json:"sparkMaster"`
	Schedule                    string `json:"schedule"`
	Image                       string `json:"image"`
	JavaOpts                    string `json:"javaOpts"`
	CassandraClientAuthEnabled  bool   `json:"cassandraClientAuthEnabled"`
	ElasticsearchClientNodeOnly bool   `json:"elasticsearchClientNodeOnly"`
	ElasticsearchNodesWanOnly   bool   `json:"elasticsearchNodesWanOnly"`
}

// JaegerEsIndexCleanerSpec holds the options related to es-index-cleaner
// +k8s:openapi-gen=true
type JaegerEsIndexCleanerSpec struct {
	Enabled      *bool  `json:"enabled"`
	NumberOfDays int    `json:"numberOfDays"`
	Schedule     string `json:"schedule"`
	Image        string `json:"image"`
}

// JaegerEsRolloverSpec holds the options related to es-rollover
type JaegerEsRolloverSpec struct {
	Image      string `json:"image"`
	Schedule   string `json:"schedule"`
	Conditions string `json:"conditions"`
	// we parse it with time.ParseDuration
	ReadTTL string `json:"readTTL"`
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
