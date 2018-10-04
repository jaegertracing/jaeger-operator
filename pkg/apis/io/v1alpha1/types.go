package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
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
	AllInOne  JaegerAllInOneSpec  `json:"all-in-one"`
	Query     JaegerQuerySpec     `json:"query"`
	Collector JaegerCollectorSpec `json:"collector"`
	Agent     JaegerAgentSpec     `json:"agent"`
	Storage   JaegerStorageSpec   `json:"storage"`
	// TODO: should we add Labels and Annotations from this level as well?
	// Here, it would be ambiguous, as it inherits from the main Jaeger object: is it then
	// related to the operator, or to the objects the operator creates? for now, let's avoid
	// this ambiguity
}

// JaegerStatus defines what is to be returned from a status query
type JaegerStatus struct {
	// Fill me
}

// JaegerQuerySpec defines the options to be used when deploying the query
type JaegerQuerySpec struct {
	Ingress     JaegerIngressSpec `json:"ingress"`
	Size        int               `json:"size"`
	Image       string            `json:"image"`
	Options     Options           `json:"options"`
	Labels      map[string]string `json:"labels"`
	Annotations map[string]string `json:"annotations"`
}

// JaegerIngressSpec defines the options to be used when deploying the query ingress
type JaegerIngressSpec struct {
	Enabled     *bool             `json:"enabled"`
	Labels      map[string]string `json:"labels"`
	Annotations map[string]string `json:"annotations"`
}

// JaegerAllInOneSpec defines the options to be used when deploying the query
type JaegerAllInOneSpec struct {
	Ingress     JaegerIngressSpec `json:"ingress"`
	Image       string            `json:"image"`
	Options     Options           `json:"options"`
	Labels      map[string]string `json:"labels"`
	Annotations map[string]string `json:"annotations"`
}

// JaegerCollectorSpec defines the options to be used when deploying the collector
type JaegerCollectorSpec struct {
	Size        int               `json:"size"`
	Image       string            `json:"image"`
	Options     Options           `json:"options"`
	Labels      map[string]string `json:"labels"`
	Annotations map[string]string `json:"annotations"`
}

// JaegerAgentSpec defines the options to be used when deploying the agent
type JaegerAgentSpec struct {
	Strategy    string            `json:"strategy"` // can be either 'DaemonSet' or 'Sidecar' (default)
	Image       string            `json:"image"`
	Labels      map[string]string `json:"labels"`
	Annotations map[string]string `json:"annotations"`
}

// JaegerStorageSpec defines the common storage options to be used for the query and collector
type JaegerStorageSpec struct {
	Type    string  `json:"type"` // can be `memory` (default), `cassandra`, `elasticsearch`, `kafka` or `managed`
	Options Options `json:"options"`
}
