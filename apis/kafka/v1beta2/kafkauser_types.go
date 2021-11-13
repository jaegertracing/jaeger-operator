package v1beta2

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	v1 "github.com/jaegertracing/jaeger-operator/apis/v1"
)

// KafkaUserSpec defines the desired state of KafkaUser
type KafkaUserSpec struct {
	v1.FreeForm `json:",inline"`
}

// KafkaUserStatus defines the observed state of KafkaUser
type KafkaUserStatus struct {
	// +listType=set
	Conditions []KafkaStatusCondition `json:"conditions,omitempty"`
}

// KafkaUser is the Schema for the kafkausers API
// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:resource:path=kafkausers,scope=Namespaced
type KafkaUser struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   KafkaUserSpec   `json:"spec,omitempty"`
	Status KafkaUserStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// KafkaUserList contains a list of KafkaUser
type KafkaUserList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []KafkaUser `json:"items"`
}

func init() {
	SchemeBuilder.Register(&KafkaUser{}, &KafkaUserList{})
}
