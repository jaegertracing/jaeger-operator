package v1alpha1

import (
	"k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	esv1 "github.com/jaegertracing/jaeger-operator/pkg/storage/elasticsearch/v1"
)

// IngressSecurityType represents the possible values for the security type
type IngressSecurityType string

const (
	// FlagPlatformKubernetes represents the value for the 'platform' flag for Kubernetes
	FlagPlatformKubernetes = "kubernetes"

	// FlagPlatformOpenShift represents the value for the 'platform' flag for OpenShift
	FlagPlatformOpenShift = "openshift"

	// FlagPlatformAutoDetect represents the "auto-detect" value for the platform flag
	FlagPlatformAutoDetect = "auto-detect"

	// FlagProvisionElasticsearchAuto represents the 'auto' value for the 'es-provision' flag
	FlagProvisionElasticsearchAuto = "auto"

	// FlagProvisionElasticsearchTrue represents the value 'true' for the 'es-provision' flag
	FlagProvisionElasticsearchTrue = "true"

	// FlagProvisionElasticsearchFalse represents the value 'false' for the 'es-provision' flag
	FlagProvisionElasticsearchFalse = "false"

	// IngressSecurityNone disables any form of security for ingress objects (default)
	IngressSecurityNone IngressSecurityType = ""

	// IngressSecurityNoneExplicit used when the user specifically set it to 'none'
	IngressSecurityNoneExplicit IngressSecurityType = "none"

	// IngressSecurityOAuthProxy represents an OAuth Proxy as security type
	IngressSecurityOAuthProxy IngressSecurityType = "oauth-proxy"
)

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// JaegerList is a list of Jaeger structs
type JaegerList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata"`
	Items           []Jaeger `json:"items"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// Jaeger defines the main structure for the custom-resource
type Jaeger struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata"`
	Spec              JaegerSpec   `json:"spec"`
	Status            JaegerStatus `json:"status,omitempty"`
}

// JaegerSpec defines the structure of the Jaeger JSON object from the CR
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

// JaegerCommonSpec defines the common elements used in multiple other spec structs
type JaegerCommonSpec struct {
	Volumes      []v1.Volume             `json:"volumes"`
	VolumeMounts []v1.VolumeMount        `json:"volumeMounts"`
	Annotations  map[string]string       `json:"annotations,omitempty"`
	Resources    v1.ResourceRequirements `json:"resources,omitempty"`
}

// JaegerStatus defines what is to be returned from a status query
type JaegerStatus struct {
	// CollectorSpansReceived represents sum of the metric jaeger_collector_spans_received_total across all collectors
	CollectorSpansReceived int `json:"collectorSpansReceived"`

	// CollectorSpansDropped represents sum of the metric jaeger_collector_spans_dropped_total across all collectors
	CollectorSpansDropped int `json:"collectorSpansDropped"`
}

// JaegerQuerySpec defines the options to be used when deploying the query
type JaegerQuerySpec struct {
	Size    int     `json:"size"`
	Image   string  `json:"image"`
	Options Options `json:"options"`
	JaegerCommonSpec
}

// JaegerUISpec defines the options to be used to configure the UI
type JaegerUISpec struct {
	Options FreeForm `json:"options"`
}

// JaegerSamplingSpec defines the options to be used to configure the UI
type JaegerSamplingSpec struct {
	Options FreeForm `json:"options"`
}

// JaegerIngressSpec defines the options to be used when deploying the query ingress
type JaegerIngressSpec struct {
	Enabled  *bool               `json:"enabled"`
	Security IngressSecurityType `json:"security"`
	JaegerCommonSpec
}

// JaegerAllInOneSpec defines the options to be used when deploying the query
type JaegerAllInOneSpec struct {
	Image   string  `json:"image"`
	Options Options `json:"options"`
	JaegerCommonSpec
}

// JaegerCollectorSpec defines the options to be used when deploying the collector
type JaegerCollectorSpec struct {
	Size    int     `json:"size"`
	Image   string  `json:"image"`
	Options Options `json:"options"`
	JaegerCommonSpec
}

// JaegerIngesterSpec defines the options to be used when deploying the ingester
type JaegerIngesterSpec struct {
	Size    int     `json:"size"`
	Image   string  `json:"image"`
	Options Options `json:"options"`
	JaegerCommonSpec
}

// JaegerAgentSpec defines the options to be used when deploying the agent
type JaegerAgentSpec struct {
	Strategy string  `json:"strategy"` // can be either 'DaemonSet' or 'Sidecar' (default)
	Image    string  `json:"image"`
	Options  Options `json:"options"`
	JaegerCommonSpec
}

// JaegerStorageSpec defines the common storage options to be used for the query and collector
type JaegerStorageSpec struct {
	Type                  string                          `json:"type"` // can be `memory` (default), `cassandra`, `elasticsearch`, `kafka` or `managed`
	SecretName            string                          `json:"secretName"`
	Options               Options                         `json:"options"`
	CassandraCreateSchema JaegerCassandraCreateSchemaSpec `json:"cassandraCreateSchema"`
	SparkDependencies     JaegerDependenciesSpec          `json:"dependencies"`
	EsIndexCleaner        JaegerEsIndexCleanerSpec        `json:"esIndexCleaner"`
	Elasticsearch         ElasticsearchSpec               `json:"elasticsearch"`
}

// ElasticsearchSpec represents the ES configuration options that we pass down to the Elasticsearch operator
type ElasticsearchSpec struct {
	Resources        v1.ResourceRequirements       `json:"resources"`
	NodeCount        int32                         `json:"nodeCount"`
	NodeSelector     map[string]string             `json:"nodeSelector,omitempty"`
	Storage          esv1.ElasticsearchStorageSpec `json:"storage"`
	RedundancyPolicy esv1.RedundancyPolicyType     `json:"redundancyPolicy"`
}

// JaegerCassandraCreateSchemaSpec holds the options related to the create-schema batch job
type JaegerCassandraCreateSchemaSpec struct {
	Enabled    *bool  `json:"enabled"`
	Image      string `json:"image"`
	Datacenter string `json:"datacenter"`
	Mode       string `json:"mode"`
}

// JaegerDependenciesSpec defined options for running spark-dependencies.
type JaegerDependenciesSpec struct {
	Enabled                     *bool  `json:"enabled"`
	SparkMaster                 string `json:"sparkMaster"`
	Schedule                    string `json:"schedule"`
	Image                       string `json:"image"`
	JavaOpts                    string `json:"javaOpts"`
	CassandraUseSsl             bool   `json:"cassandraUseSsl"`
	CassandraLocalDc            string `json:"cassandraLocalDc"`
	CassandraClientAuthEnabled  bool   `json:"cassandraClientAuthEnabled"`
	ElasticsearchClientNodeOnly bool   `json:"elasticsearchClientNodeOnly"`
	ElasticsearchNodesWanOnly   bool   `json:"elasticsearchNodesWanOnly"`
}

// JaegerEsIndexCleanerSpec holds the options related to es-index-cleaner
type JaegerEsIndexCleanerSpec struct {
	Enabled      *bool  `json:"enabled"`
	NumberOfDays int    `json:"numberOfDays"`
	Schedule     string `json:"schedule"`
	Image        string `json:"image"`
}

func init() {
	SchemeBuilder.Register(&Jaeger{}, &JaegerList{})
}
