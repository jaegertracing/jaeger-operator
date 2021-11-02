package v1beta2

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	v1 "github.com/jaegertracing/jaeger-operator/pkg/apis/jaegertracing/v1"
)

// KafkaSpec defines the desired state of Kafka
// +k8s:openapi-gen=true
type KafkaSpec struct {
	v1.FreeForm `json:",inline"`
}

// KafkaStatus defines the observed state of Kafka
// +k8s:openapi-gen=true
type KafkaStatus struct {
	// +listType=set
	Conditions []KafkaStatusCondition `json:"conditions,omitempty"`
}

// KafkaStatusCondition holds the different conditions affecting the Kafka instance
// +k8s:openapi-gen=true
type KafkaStatusCondition struct {
	Type               string `json:"type,omitempty"`
	Status             string `json:"status,omitempty"`
	LastTransitionTime string `json:"lastTransitionTime,omitempty"`
	Reason             string `json:"reason,omitempty"`
	Message            string `json:"message,omitempty"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// Kafka is the Schema for the kafkas API
// +k8s:openapi-gen=true
// +kubebuilder:subresource:status
// +kubebuilder:resource:path=kafkas,scope=Namespaced
type Kafka struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   KafkaSpec   `json:"spec,omitempty"`
	Status KafkaStatus `json:"status,omitempty"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// KafkaList contains a list of Kafka
type KafkaList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []Kafka `json:"items"`
}

func init() {
	SchemeBuilder.Register(&Kafka{}, &KafkaList{})
}
