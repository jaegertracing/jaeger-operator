package v1

import (
	esv1 "github.com/openshift/elasticsearch-operator/apis/logging/v1"
	appsv1 "k8s.io/api/apps/v1"
	v1 "k8s.io/api/core/v1"
	networkingv1 "k8s.io/api/networking/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// IngressSecurityType represents the possible values for the security type
type IngressSecurityType string

// JaegerPhase represents the current phase of Jaeger instances
type JaegerPhase string

// JaegerStorageType represents the Jaeger storage type
type JaegerStorageType string

const (
	// FlagCronJobsVersion represents the version of the Kubernetes CronJob API
	FlagCronJobsVersion = "cronjobs-version"

	// FlagCronJobsVersionBatchV1 represents the batch/v1 version of the Kubernetes CronJob API, available as of 1.21
	FlagCronJobsVersionBatchV1 = "batch/v1"

	// FlagCronJobsVersionBatchV1Beta1 represents the batch/v1beta1 version of the Kubernetes CronJob API, no longer available as of 1.25
	FlagCronJobsVersionBatchV1Beta1 = "batch/v1beta1"

	// FlagAutoscalingVersion represents the version of the Kubernetes Autoscaling API
	FlagAutoscalingVersion = "autoscaling-version"

	// FlagAutoscalingVersionV2 represents the v2 version of the Kubernetes Autoscaling API, available as of 1.23
	FlagAutoscalingVersionV2 = "autoscaling/v2"

	// FlagAutoscalingVersionV2Beta2 represents the v2beta2 version of the Kubernetes Autoscaling API, no longer available as of 1.26
	FlagAutoscalingVersionV2Beta2 = "autoscaling/v2beta2"

	// FlagPlatformKubernetes represents the value for the 'platform' flag for Kubernetes
	FlagPlatformKubernetes = "kubernetes"

	// FlagPlatformOpenShift represents the value for the 'platform' flag for OpenShift
	FlagPlatformOpenShift = "openshift"

	// FlagPlatformAutoDetect represents the "auto-detect" value for the platform flag
	FlagPlatformAutoDetect = "auto-detect"

	// FlagProvisionElasticsearchAuto represents the 'auto' value for the 'es-provision' flag
	FlagProvisionElasticsearchAuto = "auto"

	// FlagProvisionElasticsearchYes represents the value 'yes' for the 'es-provision' flag
	FlagProvisionElasticsearchYes = "yes"

	// FlagProvisionElasticsearchNo represents the value 'no' for the 'es-provision' flag
	FlagProvisionElasticsearchNo = "no"

	// FlagProvisionKafkaAuto represents the 'auto' value for the 'kafka-provision' flag
	FlagProvisionKafkaAuto = "auto"

	// FlagProvisionKafkaYes represents the value 'yes' for the 'kafka-provision' flag
	FlagProvisionKafkaYes = "yes"

	// FlagProvisionKafkaNo represents the value 'no' for the 'kafka-provision' flag
	FlagProvisionKafkaNo = "no"

	// IngressSecurityNone disables any form of security for ingress objects (default)
	IngressSecurityNone IngressSecurityType = ""

	// IngressSecurityNoneExplicit used when the user specifically set it to 'none'
	IngressSecurityNoneExplicit IngressSecurityType = "none"

	// IngressSecurityOAuthProxy represents an OAuth Proxy as security type
	IngressSecurityOAuthProxy IngressSecurityType = "oauth-proxy"

	// AnnotationProvisionedKafkaKey is a label to be added to Kafkas that have been provisioned by Jaeger
	AnnotationProvisionedKafkaKey string = "jaegertracing.io/kafka-provisioned"

	// AnnotationProvisionedKafkaValue is a label to be added to Kafkas that have been provisioned by Jaeger
	AnnotationProvisionedKafkaValue string = "true"

	// JaegerPhaseFailed indicates that the Jaeger instance failed to be provisioned
	JaegerPhaseFailed JaegerPhase = "Failed"

	// JaegerPhaseRunning indicates that the Jaeger instance is ready and running
	JaegerPhaseRunning JaegerPhase = "Running"

	// JaegerMemoryStorage indicates that the Jaeger storage type is memory. This is the default storage type.
	JaegerMemoryStorage JaegerStorageType = "memory"

	// JaegerCassandraStorage indicates that the Jaeger storage type is cassandra
	JaegerCassandraStorage JaegerStorageType = "cassandra"

	// JaegerESStorage indicates that the Jaeger storage type is elasticsearch
	JaegerESStorage JaegerStorageType = "elasticsearch"

	// JaegerKafkaStorage indicates that the Jaeger storage type is kafka
	JaegerKafkaStorage JaegerStorageType = "kafka"

	// JaegerBadgerStorage indicates that the Jaeger storage type is badger
	JaegerBadgerStorage JaegerStorageType = "badger"

	// JaegerGRPCPluginStorage indicates that the Jaeger storage type is grpc-plugin
	JaegerGRPCPluginStorage JaegerStorageType = "grpc-plugin"
)

// ValidStorageTypes returns the list of valid storage types
func ValidStorageTypes() []JaegerStorageType {
	return []JaegerStorageType{
		JaegerMemoryStorage,
		JaegerCassandraStorage,
		JaegerESStorage,
		JaegerKafkaStorage,
		JaegerBadgerStorage,
		JaegerGRPCPluginStorage,
	}
}

// OptionsPrefix returns the options prefix associated with the storage type
func (storageType JaegerStorageType) OptionsPrefix() string {
	if storageType == JaegerESStorage {
		return "es"
	}
	if storageType == JaegerGRPCPluginStorage {
		return "grpc-storage-plugin"
	}
	return string(storageType)
}

// JaegerSpec defines the desired state of Jaeger
type JaegerSpec struct {
	// +optional
	// +operator-sdk:csv:customresourcedefinitions:type=spec,displayName="Strategy"
	Strategy DeploymentStrategy `json:"strategy,omitempty"`

	// +optional
	AllInOne JaegerAllInOneSpec `json:"allInOne,omitempty"`

	// +optional
	Query JaegerQuerySpec `json:"query,omitempty"`

	// +optional
	Collector JaegerCollectorSpec `json:"collector,omitempty"`

	// +optional
	Ingester JaegerIngesterSpec `json:"ingester,omitempty"`

	// +optional
	// +nullable
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
type JaegerStatus struct {
	// +operator-sdk:csv:customresourcedefinitions:type=status
	// +operator-sdk:csv:customresourcedefinitions:displayName="Version"
	Version string `json:"version"`
	// +operator-sdk:csv:customresourcedefinitions:type=status
	// +operator-sdk:csv:customresourcedefinitions:displayName="Phase"
	Phase JaegerPhase `json:"phase"`
}

// Jaeger is the Schema for the jaegers API
// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +operator-sdk:gen-csv:customresourcedefinitions.displayName="Jaeger"
// +operator-sdk:csv:customresourcedefinitions:resources={{CronJob,v1beta1},{Pod,v1},{Deployment,apps/v1}, {Ingress,networking/v1},{DaemonSets,apps/v1},{StatefulSets,apps/v1},{ConfigMaps,v1},{Service,v1}}
// +kubebuilder:subresource:status
// +kubebuilder:printcolumn:name="Status",type="string",JSONPath=".status.phase",description="Jaeger instance's status"
// +kubebuilder:printcolumn:name="Version",type="string",JSONPath=".status.version",description="Jaeger Version"
// +kubebuilder:printcolumn:name="Strategy",type="string",JSONPath=".spec.strategy",description="Jaeger deployment strategy"
// +kubebuilder:printcolumn:name="Storage",type="string",JSONPath=".spec.storage.type",description="Jaeger storage type"
// +kubebuilder:printcolumn:name="Age",type="date",JSONPath=".metadata.creationTimestamp"
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
type JaegerCommonSpec struct {
	// +optional
	// +listType=atomic
	Volumes []v1.Volume `json:"volumes,omitempty"`

	// +optional
	// +listType=atomic
	VolumeMounts []v1.VolumeMount `json:"volumeMounts,omitempty"`

	// +nullable
	// +optional
	Annotations map[string]string `json:"annotations,omitempty"`

	// +optional
	Labels map[string]string `json:"labels,omitempty"`

	// +nullable
	// +optional
	Resources v1.ResourceRequirements `json:"resources,omitempty"`

	// +optional
	Affinity *v1.Affinity `json:"affinity,omitempty"`

	// +optional
	// +listType=atomic
	Tolerations []v1.Toleration `json:"tolerations,omitempty"`

	// +optional
	SecurityContext *v1.PodSecurityContext `json:"securityContext,omitempty"`

	// +optional
	ContainerSecurityContext *v1.SecurityContext `json:"containerSecurityContext,omitempty"`

	// +optional
	ServiceAccount string `json:"serviceAccount,omitempty"`

	// +optional
	LivenessProbe *v1.Probe `json:"livenessProbe,omitempty"`

	// +optional
	// +listType=atomic
	ImagePullSecrets []v1.LocalObjectReference `json:"imagePullSecrets,omitempty"`

	// +optional
	ImagePullPolicy v1.PullPolicy `json:"imagePullPolicy,omitempty"`
}

// JaegerQuerySpec defines the options to be used when deploying the query
type JaegerQuerySpec struct {
	// Replicas represents the number of replicas to create for this service.
	// +optional
	Replicas *int32 `json:"replicas,omitempty"`

	// +optional
	Image string `json:"image,omitempty"`

	// +optional
	// +kubebuilder:pruning:PreserveUnknownFields
	Options Options `json:"options,omitempty"`

	// +optional
	MetricsStorage JaegerMetricsStorageSpec `json:"metricsStorage,omitempty"`

	// +optional
	JaegerCommonSpec `json:",inline,omitempty"`

	// +optional
	// ServiceType represents the type of Service to create.
	// Valid values include: ClusterIP, NodePort, LoadBalancer, and ExternalName.
	// The default, if omitted, is ClusterIP.
	// See https://kubernetes.io/docs/concepts/services-networking/service/#publishing-services-service-types
	ServiceType v1.ServiceType `json:"serviceType,omitempty"`

	// +optional
	// NodePort represents the port at which the NodePort service to allocate
	NodePort int32 `json:"nodePort,omitempty"`

	// +optional
	// NodePort represents the port at which the NodePort service to allocate
	GRPCNodePort int32 `json:"grpcNodePort,omitempty"`

	// +optional
	// TracingEnabled if set to false adds the JAEGER_DISABLED environment flag and removes the injected
	// agent container from the query component to disable tracing requests to the query service.
	// The default, if omitted, is true
	TracingEnabled *bool `json:"tracingEnabled,omitempty"`

	// +optional
	PriorityClassName string `json:"priorityClassName,omitempty"`

	// +optional
	// +operator-sdk:csv:customresourcedefinitions:type=spec,displayName="Strategy"
	Strategy *appsv1.DeploymentStrategy `json:"strategy,omitempty"`

	// +optional
	// +nullable
	NodeSelector map[string]string `json:"nodeSelector,omitempty"`
}

// JaegerUISpec defines the options to be used to configure the UI
type JaegerUISpec struct {
	// +optional
	// +kubebuilder:pruning:PreserveUnknownFields
	Options FreeForm `json:"options,omitempty"`
}

// JaegerSamplingSpec defines the options to be used to configure the UI
type JaegerSamplingSpec struct {
	// +optional
	// +kubebuilder:pruning:PreserveUnknownFields
	Options FreeForm `json:"options,omitempty"`
}

// JaegerIngressSpec defines the options to be used when deploying the query ingress
type JaegerIngressSpec struct {
	// +optional
	Enabled *bool `json:"enabled,omitempty"`

	// +optional
	Security IngressSecurityType `json:"security,omitempty"`

	// +optional
	Openshift JaegerIngressOpenShiftSpec `json:"openshift,omitempty"`

	// +optional
	// +listType=atomic
	Hosts []string `json:"hosts,omitempty"`

	// +optional
	PathType networkingv1.PathType `json:"pathType,omitempty"`

	// +optional
	// +listType=atomic
	TLS []JaegerIngressTLSSpec `json:"tls,omitempty"`

	// Deprecated in favor of the TLS property
	// +optional
	SecretName string `json:"secretName,omitempty"`

	// +optional
	JaegerCommonSpec `json:",inline,omitempty"`

	// +optional
	// +kubebuilder:pruning:PreserveUnknownFields
	Options Options `json:"options,omitempty"`

	// +optional
	IngressClassName *string `json:"ingressClassName,omitempty"`
}

// JaegerIngressTLSSpec defines the TLS configuration to be used when deploying the query ingress
type JaegerIngressTLSSpec struct {
	// +optional
	// +listType=atomic
	Hosts []string `json:"hosts,omitempty"`

	// +optional
	SecretName string `json:"secretName,omitempty"`
}

// JaegerIngressOpenShiftSpec defines the OpenShift-specific options in the context of ingress connections,
// such as options for the OAuth Proxy
type JaegerIngressOpenShiftSpec struct {
	// +optional
	SAR *string `json:"sar,omitempty"`

	// +optional
	DelegateUrls string `json:"delegateUrls,omitempty"`

	// +optional
	HtpasswdFile string `json:"htpasswdFile,omitempty"`

	// SkipLogout tells the operator to not automatically add a "Log Out" menu option to the custom Jaeger configuration
	// +optional
	SkipLogout *bool `json:"skipLogout,omitempty"`
}

// JaegerAllInOneSpec defines the options to be used when deploying the query
type JaegerAllInOneSpec struct {
	// +optional
	Image string `json:"image,omitempty"`

	// +optional
	// +kubebuilder:pruning:PreserveUnknownFields
	Options Options `json:"options,omitempty"`

	// +optional
	// +kubebuilder:pruning:PreserveUnknownFields
	Config FreeForm `json:"config,omitempty"`

	// +optional
	MetricsStorage JaegerMetricsStorageSpec `json:"metricsStorage,omitempty"`

	// +optional
	JaegerCommonSpec `json:",inline,omitempty"`

	// +optional
	// TracingEnabled if set to false adds the JAEGER_DISABLED environment flag and removes the injected
	// agent container from the query component to disable tracing requests to the query service.
	// The default, if omitted, is true
	TracingEnabled *bool `json:"tracingEnabled,omitempty"`

	// +optional
	// +operator-sdk:csv:customresourcedefinitions:type=spec,displayName="Strategy"
	Strategy *appsv1.DeploymentStrategy `json:"strategy,omitempty"`

	// +optional
	PriorityClassName string `json:"priorityClassName,omitempty"`
}

// AutoScaleSpec defines the common elements used for create HPAs
type AutoScaleSpec struct {
	// Autoscale turns on/off the autoscale feature. By default, it's enabled if the Replicas field is not set.
	// +optional
	Autoscale *bool `json:"autoscale,omitempty"`
	// MinReplicas sets a lower bound to the autoscaling feature.
	// +optional
	MinReplicas *int32 `json:"minReplicas,omitempty"`

	// MaxReplicas sets an upper bound to the autoscaling feature. When autoscaling is enabled and no value is provided, a default value is used.
	// +optional
	MaxReplicas *int32 `json:"maxReplicas,omitempty"`
}

// JaegerCollectorSpec defines the options to be used when deploying the collector
type JaegerCollectorSpec struct {
	// +optional
	AutoScaleSpec `json:",inline,omitempty"`

	// Replicas represents the number of replicas to create for this service.
	// +optional
	Replicas *int32 `json:"replicas,omitempty"`

	// +optional
	Image string `json:"image,omitempty"`

	// +optional
	// +kubebuilder:pruning:PreserveUnknownFields
	Options Options `json:"options,omitempty"`

	// +optional
	JaegerCommonSpec `json:",inline,omitempty"`

	// +optional
	// +kubebuilder:pruning:PreserveUnknownFields
	Config FreeForm `json:"config,omitempty"`

	// +optional
	// ServiceType represents the type of Service to create.
	// Valid values include: ClusterIP, NodePort, LoadBalancer, and ExternalName.
	// The default, if omitted, is ClusterIP.
	// See https://kubernetes.io/docs/concepts/services-networking/service/#publishing-services-service-types
	ServiceType v1.ServiceType `json:"serviceType,omitempty"`

	// +optional
	PriorityClassName string `json:"priorityClassName,omitempty"`

	// +optional
	// +operator-sdk:csv:customresourcedefinitions:type=spec,displayName="Strategy"
	Strategy *appsv1.DeploymentStrategy `json:"strategy,omitempty"`

	// +optional
	KafkaSecretName string `json:"kafkaSecretName"`

	// +optional
	// +nullable
	NodeSelector map[string]string `json:"nodeSelector,omitempty"`

	// +optional
	Lifecycle *v1.Lifecycle `json:"lifecycle,omitempty"`

	// +optional
	TerminationGracePeriodSeconds *int64 `json:"terminationGracePeriodSeconds,omitempty"`
}

// JaegerIngesterSpec defines the options to be used when deploying the ingester
type JaegerIngesterSpec struct {
	// +optional
	AutoScaleSpec `json:",inline,omitempty"`

	// Replicas represents the number of replicas to create for this service.
	// +optional
	Replicas *int32 `json:"replicas,omitempty"`

	// +optional
	Image string `json:"image,omitempty"`

	// +optional
	// +kubebuilder:pruning:PreserveUnknownFields
	Options Options `json:"options,omitempty"`

	// +optional
	JaegerCommonSpec `json:",inline,omitempty"`

	// +optional
	// +kubebuilder:pruning:PreserveUnknownFields
	Config FreeForm `json:"config,omitempty"`

	// +optional
	Strategy *appsv1.DeploymentStrategy `json:"strategy,omitempty"`

	// +optional
	KafkaSecretName string `json:"kafkaSecretName"`

	// +optional
	// +nullable
	NodeSelector map[string]string `json:"nodeSelector,omitempty"`
}

// JaegerAgentSpec defines the options to be used when deploying the agent
type JaegerAgentSpec struct {
	// Strategy can be either 'DaemonSet' or 'Sidecar' (default)
	// +optional
	Strategy string `json:"strategy,omitempty"`

	// +optional
	Image string `json:"image,omitempty"`

	// +optional
	// +kubebuilder:pruning:PreserveUnknownFields
	Options Options `json:"options,omitempty"`

	// +optional
	JaegerCommonSpec `json:",inline,omitempty"`

	// +optional
	// +kubebuilder:pruning:PreserveUnknownFields
	Config FreeForm `json:"config,omitempty"`

	// +optional
	SidecarSecurityContext *v1.SecurityContext `json:"sidecarSecurityContext,omitempty"`

	// +optional
	HostNetwork *bool `json:"hostNetwork,omitempty"`

	// +optional
	DNSPolicy v1.DNSPolicy `json:"dnsPolicy,omitempty"`

	// +optional
	PriorityClassName string `json:"priorityClassName,omitempty"`
}

// JaegerStorageSpec defines the common storage options to be used for the query and collector
type JaegerStorageSpec struct {
	// +optional
	Type JaegerStorageType `json:"type,omitempty"`

	// +optional
	SecretName string `json:"secretName,omitempty"`

	// +optional
	// +kubebuilder:pruning:PreserveUnknownFields
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

	// +optional
	GRPCPlugin GRPCPluginSpec `json:"grpcPlugin,omitempty"`
}

// JaegerMetricsStorageSpec defines the Metrics storage options to be used for the query and collector.
type JaegerMetricsStorageSpec struct {
	// +optional
	Type JaegerStorageType `json:"type,omitempty"`
}

// ElasticsearchSpec represents the ES configuration options that we pass down to the OpenShift Elasticsearch operator.
type ElasticsearchSpec struct {
	// Name of the OpenShift Elasticsearch instance. Defaults to elasticsearch.
	// +optional
	Name string `json:"name,omitempty"`

	// Whether Elasticsearch should be provisioned or not.
	// +optional
	DoNotProvision bool `json:"doNotProvision,omitempty"`

	// Whether Elasticsearch cert management feature should be used.
	// This is a preferred setting for new Jaeger deployments on OCP versions newer than 4.6.
	// The cert management feature was added to Red Hat Openshift logging 5.2 in OCP 4.7.
	// +optional
	UseCertManagement *bool `json:"useCertManagement,omitempty"`

	// +optional
	Image string `json:"image,omitempty"`

	// +optional
	Resources *v1.ResourceRequirements `json:"resources,omitempty"`

	// +optional
	NodeCount int32 `json:"nodeCount,omitempty"`

	// +optional
	NodeSelector map[string]string `json:"nodeSelector,omitempty"`

	// +optional
	Storage esv1.ElasticsearchStorageSpec `json:"storage,omitempty"`

	// +optional
	RedundancyPolicy esv1.RedundancyPolicyType `json:"redundancyPolicy,omitempty"`

	// +optional
	// +listType=atomic
	Tolerations []v1.Toleration `json:"tolerations,omitempty"`

	// +optional
	ProxyResources *v1.ResourceRequirements `json:"proxyResources,omitempty"`
}

// JaegerCassandraCreateSchemaSpec holds the options related to the create-schema batch job
type JaegerCassandraCreateSchemaSpec struct {
	// +optional
	Enabled *bool `json:"enabled,omitempty"`

	// Image specifies the container image to use to create the cassandra schema.
	// The Image is used by a Kubernetes Job, defaults to the image provided through the cli flag "jaeger-cassandra-schema-image" (default: jaegertracing/jaeger-cassandra-schema).
	// See here for the jaeger-provided image: https://github.com/jaegertracing/jaeger/tree/main/plugin/storage/cassandra
	// +optional
	Image string `json:"image,omitempty"`

	// Datacenter is a collection of racks in the cassandra topology.
	// defaults to "test"
	// +optional
	Datacenter string `json:"datacenter,omitempty"`

	// Mode controls the replication factor of your cassandra schema.
	// Set it to "prod" (which is the default) to use the NetworkTopologyStrategy with a replication factor of 2, effectively meaning
	// that at least 3 nodes are required in the cassandra cluster.
	// When set to "test" the schema uses the SimpleStrategy with a replication factor of 1. You never want to do this in a production setup.
	// +optional
	Mode string `json:"mode,omitempty"`

	// TraceTTL sets the TTL for your trace data
	// +optional
	TraceTTL string `json:"traceTTL,omitempty"`

	// Timeout controls the Job deadline, it defaults to 1 day.
	// specify it with a value which can be parsed by time.ParseDuration, e.g. 24h or 120m.
	// If the job does not succeed within that duration it transitions into a permanent error state.
	// See https://github.com/jaegertracing/jaeger-kubernetes/issues/32 and
	// https://github.com/jaegertracing/jaeger-kubernetes/pull/125
	// +optional
	Timeout string `json:"timeout,omitempty"`

	// +optional
	Affinity *v1.Affinity `json:"affinity,omitempty"`

	// +optional
	TTLSecondsAfterFinished *int32 `json:"ttlSecondsAfterFinished,omitempty"`
}

// GRPCPluginSpec represents the grpc-plugin configuration options.
type GRPCPluginSpec struct {
	// This image is used as an init-container to copy plugin binary into /plugin directory.
	// +optional
	Image string `json:"image,omitempty"`
}

// JaegerDependenciesSpec defined options for running spark-dependencies.
type JaegerDependenciesSpec struct {
	// +optional
	Enabled *bool `json:"enabled,omitempty"`

	// +optional
	SparkMaster string `json:"sparkMaster,omitempty"`

	// +optional
	Schedule string `json:"schedule,omitempty"`

	// +optional
	SuccessfulJobsHistoryLimit *int32 `json:"successfulJobsHistoryLimit,omitempty"`

	// +optional
	Image string `json:"image,omitempty"`

	// +optional
	JavaOpts string `json:"javaOpts,omitempty"`

	// +optional
	CassandraClientAuthEnabled bool `json:"cassandraClientAuthEnabled,omitempty"`

	// +optional
	ElasticsearchClientNodeOnly *bool `json:"elasticsearchClientNodeOnly,omitempty"`

	// +optional
	ElasticsearchNodesWanOnly *bool `json:"elasticsearchNodesWanOnly,omitempty"`

	// +optional
	ElasticsearchTimeRange string `json:"elasticsearchTimeRange,omitempty"`

	// +optional
	TTLSecondsAfterFinished *int32 `json:"ttlSecondsAfterFinished,omitempty"`

	// BackoffLimit sets the Kubernetes back-off limit
	// +optional
	BackoffLimit *int32 `json:"backoffLimit,omitempty"`

	// +optional
	JaegerCommonSpec `json:",inline,omitempty"`
}

// JaegerEsIndexCleanerSpec holds the options related to es-index-cleaner
type JaegerEsIndexCleanerSpec struct {
	// +optional
	Enabled *bool `json:"enabled,omitempty"`

	// +optional
	NumberOfDays *int `json:"numberOfDays,omitempty"`

	// +optional
	Schedule string `json:"schedule,omitempty"`

	// +optional
	SuccessfulJobsHistoryLimit *int32 `json:"successfulJobsHistoryLimit,omitempty"`

	// +optional
	Image string `json:"image,omitempty"`

	// +optional
	TTLSecondsAfterFinished *int32 `json:"ttlSecondsAfterFinished,omitempty"`

	// BackoffLimit sets the Kubernetes back-off limit
	// +optional
	BackoffLimit *int32 `json:"backoffLimit,omitempty"`

	// +optional
	JaegerCommonSpec `json:",inline,omitempty"`

	// +optional
	PriorityClassName string `json:"priorityClassName,omitempty"`
}

// JaegerEsRolloverSpec holds the options related to es-rollover
type JaegerEsRolloverSpec struct {
	// +optional
	Image string `json:"image,omitempty"`

	// +optional
	Schedule string `json:"schedule,omitempty"`

	// +optional
	SuccessfulJobsHistoryLimit *int32 `json:"successfulJobsHistoryLimit,omitempty"`

	// +optional
	Conditions string `json:"conditions,omitempty"`

	// +optional
	TTLSecondsAfterFinished *int32 `json:"ttlSecondsAfterFinished,omitempty"`

	// BackoffLimit sets the Kubernetes back-off limit
	// +optional
	BackoffLimit *int32 `json:"backoffLimit,omitempty"`

	// we parse it with time.ParseDuration
	// +optional
	ReadTTL string `json:"readTTL,omitempty"`

	// +optional
	JaegerCommonSpec `json:",inline,omitempty"`
}

//+kubebuilder:object:root=true

// JaegerList contains a list of Jaeger
type JaegerList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []Jaeger `json:"items"`
}

func init() {
	SchemeBuilder.Register(&Jaeger{}, &JaegerList{})
}
