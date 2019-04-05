package v1

import (
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// TODO remove this file, it's temporary copied from es-operator due to old SDK dependency
//   https://github.com/jaegertracing/jaeger-operator/issues/206

const (
	ServiceAccountName string = "elasticsearch"
	ConfigMapName      string = "elasticsearch"
	SecretName         string = "elasticsearch"
)

// ElasticsearchList struct represents list of Elasticsearch objects
type ElasticsearchList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata"`
	Items           []Elasticsearch `json:"items"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// Elasticsearch struct represents Elasticsearch cluster CRD
type Elasticsearch struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata"`
	Spec              ElasticsearchSpec   `json:"spec"`
	Status            ElasticsearchStatus `json:"status,omitempty"`
}

// RedundancyPolicyType controls number of elasticsearch replica shards
type RedundancyPolicyType string

const (
	// FullRedundancy - each index is fully replicated on every Data node in the cluster
	FullRedundancy RedundancyPolicyType = "FullRedundancy"
	// MultipleRedundancy - each index is spread over half of the Data nodes
	MultipleRedundancy RedundancyPolicyType = "MultipleRedundancy"
	// SingleRedundancy - one replica shard
	SingleRedundancy RedundancyPolicyType = "SingleRedundancy"
	// ZeroRedundancy - no replica shards
	ZeroRedundancy RedundancyPolicyType = "ZeroRedundancy"
)

// ElasticsearchSpec struct represents the Spec of Elasticsearch cluster CRD
type ElasticsearchSpec struct {
	// managementState indicates whether and how the operator should manage the component
	ManagementState  ManagementState       `json:"managementState"`
	RedundancyPolicy RedundancyPolicyType  `json:"redundancyPolicy"`
	Nodes            []ElasticsearchNode   `json:"nodes"`
	Spec             ElasticsearchNodeSpec `json:"nodeSpec"`
}

// ElasticsearchNode struct represents individual node in Elasticsearch cluster
type ElasticsearchNode struct {
	Roles        []ElasticsearchNodeRole  `json:"roles"`
	NodeCount    int32                    `json:"nodeCount"`
	Resources    v1.ResourceRequirements  `json:"resources"`
	NodeSelector map[string]string        `json:"nodeSelector,omitempty"`
	Storage      ElasticsearchStorageSpec `json:"storage"`
	GenUUID      *string                  `json:"genUUID,omitempty"`
	// GenUUID will be populated by the operator if not provided
}

type ElasticsearchStorageSpec struct {
	StorageClassName *string            `json:"storageClassName,omitempty"`
	Size             *resource.Quantity `json:"size,omitempty"`
}

// ElasticsearchNodeStatus represents the status of individual Elasticsearch node
type ElasticsearchNodeStatus struct {
	DeploymentName  string                         `json:"deploymentName,omitempty"`
	ReplicaSetName  string                         `json:"replicaSetName,omitempty"`
	StatefulSetName string                         `json:"statefulSetName,omitempty"`
	PodName         string                         `json:"podName,omitempty"`
	Status          string                         `json:"status,omitempty"`
	UpgradeStatus   ElasticsearchNodeUpgradeStatus `json:"upgradeStatus,omitempty"`
	Roles           []ElasticsearchNodeRole        `json:"roles,omitempty"`
	Conditions      []ClusterCondition             `json:"conditions,omitempty"`
}

type ElasticsearchNodeUpgradeStatus struct {
	ScheduledForUpgrade  v1.ConditionStatus        `json:"scheduledUpgrade,omitempty"`
	ScheduledForRedeploy v1.ConditionStatus        `json:"scheduledRedeploy,omitempty"`
	UnderUpgrade         v1.ConditionStatus        `json:"underUpgrade,omitempty"`
	UpgradePhase         ElasticsearchUpgradePhase `json:"upgradePhase,omitempty"`
}

type ElasticsearchUpgradePhase string

const (
	NodeRestarting    ElasticsearchUpgradePhase = "nodeRestarting"
	RecoveringData    ElasticsearchUpgradePhase = "recoveringData"
	ControllerUpdated ElasticsearchUpgradePhase = "controllerUpdated"
)

// ElasticsearchNodeSpec represents configuration of an individual Elasticsearch node
type ElasticsearchNodeSpec struct {
	Image        string                  `json:"image,omitempty"`
	Resources    v1.ResourceRequirements `json:"resources"`
	NodeSelector map[string]string       `json:"nodeSelector,omitempty"`
}

type ElasticsearchRequiredAction string

const (
	ElasticsearchActionRollingRestartNeeded ElasticsearchRequiredAction = "RollingRestartNeeded"
	ElasticsearchActionFullRestartNeeded    ElasticsearchRequiredAction = "FullRestartNeeded"
	ElasticsearchActionInterventionNeeded   ElasticsearchRequiredAction = "InterventionNeeded"
	ElasticsearchActionNewClusterNeeded     ElasticsearchRequiredAction = "NewClusterNeeded"
	ElasticsearchActionNone                 ElasticsearchRequiredAction = "ClusterOK"
	ElasticsearchActionScaleDownNeeded      ElasticsearchRequiredAction = "ScaleDownNeeded"
)

type ElasticsearchNodeRole string

const (
	ElasticsearchRoleClient ElasticsearchNodeRole = "client"
	ElasticsearchRoleData   ElasticsearchNodeRole = "data"
	ElasticsearchRoleMaster ElasticsearchNodeRole = "master"
)

type ShardAllocationState string

const (
	ShardAllocationAll     ShardAllocationState = "all"
	ShardAllocationNone    ShardAllocationState = "none"
	ShardAllocationUnknown ShardAllocationState = "shard allocation unknown"
)

// ElasticsearchStatus represents the status of Elasticsearch cluster
type ElasticsearchStatus struct {
	Nodes                  []ElasticsearchNodeStatus             `json:"nodes"`
	ClusterHealth          string                                `json:"clusterHealth"`
	ShardAllocationEnabled ShardAllocationState                  `json:"shardAllocationEnabled"`
	Pods                   map[ElasticsearchNodeRole]PodStateMap `json:"pods"`
	Conditions             []ClusterCondition                    `json:"conditions"`
}

type PodStateMap map[PodStateType][]string

type PodStateType string

const (
	PodStateTypeReady    PodStateType = "ready"
	PodStateTypeNotReady PodStateType = "notReady"
	PodStateTypeFailed   PodStateType = "failed"
)

type ManagementState string

const (
	// Managed means that the operator is actively managing its resources and trying to keep the component active.
	// It will only upgrade the component if it is safe to do so
	ManagementStateManaged ManagementState = "Managed"
	// Unmanaged means that the operator will not take any action related to the component
	ManagementStateUnmanaged ManagementState = "Unmanaged"
)

// ClusterCondition contains details for the current condition of this elasticsearch cluster.
type ClusterCondition struct {
	// Type is the type of the condition.
	Type ClusterConditionType `json:"type"`
	// Status is the status of the condition.
	Status v1.ConditionStatus `json:"status"`
	// Last time the condition transitioned from one status to another.
	LastTransitionTime metav1.Time `json:"lastTransitionTime"`
	// Unique, one-word, CamelCase reason for the condition's last transition.
	Reason string `json:"reason,omitempty" protobuf:"bytes,5,opt,name=reason"`
	// Human-readable message indicating details about last transition.
	Message string `json:"message,omitempty" protobuf:"bytes,6,opt,name=message"`
}

// ClusterConditionType is a valid value for ClusterCondition.Type
type ClusterConditionType string

// These are valid conditions for elasticsearch node
const (
	UpdatingSettings ClusterConditionType = "UpdatingSettings"
	ScalingUp        ClusterConditionType = "ScalingUp"
	ScalingDown      ClusterConditionType = "ScalingDown"
	Restarting       ClusterConditionType = "Restarting"

	InvalidMasters    ClusterConditionType = "InvalidMasters"
	InvalidData       ClusterConditionType = "InvalidData"
	InvalidRedundancy ClusterConditionType = "InvalidRedundancy"
	InvalidUUID       ClusterConditionType = "InvalidUUID"

	ESContainerWaiting       ClusterConditionType = "ElasticsearchContainerWaiting"
	ESContainerTerminated    ClusterConditionType = "ElasticsearchContainerTerminated"
	ProxyContainerWaiting    ClusterConditionType = "ProxyContainerWaiting"
	ProxyContainerTerminated ClusterConditionType = "ProxyContainerTerminated"
	Unschedulable            ClusterConditionType = "Unschedulable"
	NodeStorage              ClusterConditionType = "NodeStorage"
)

type ClusterEvent string

const (
	ScaledDown            ClusterEvent = "ScaledDown"
	ScaledUp              ClusterEvent = "ScaledUp"
	UpdateClusterSettings ClusterEvent = "UpdateClusterSettings"
	NoEvent               ClusterEvent = "NoEvent"
)

func init() {
	SchemeBuilder.Register(&Elasticsearch{}, &ElasticsearchList{})
}
