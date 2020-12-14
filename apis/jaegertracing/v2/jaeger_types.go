/*


Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package v2

import (
	"k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// IngressSecurityType represents the possible values for the security type
type IngressSecurityType string

// JaegerPhase represents the current phase of Jaeger instances
type JaegerPhase string

// JaegerStorageType represents the Jaeger storage type
type JaegerStorageType string

// JaegerSpec defines the desired state of Jaeger
type JaegerSpec struct {
	// +optional
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
	Ingress JaegerIngressSpec `json:"ingress,omitempty"`

	// +optional
	JaegerCommonSpec `json:",inline,omitempty"`
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
	ServiceAccount string `json:"serviceAccount,omitempty"`
}

// JaegerQuerySpec defines the options to be used when deploying the query
type JaegerQuerySpec struct {
	// Replicas represents the number of replicas to create for this service.
	// +optional
	Replicas *int32 `json:"replicas,omitempty"`

	// +optional
	Image string `json:"image,omitempty"`

	// +optional
	JaegerCommonSpec `json:",inline,omitempty"`

	// +optional
	// ServiceType represents the type of Service to create.
	// Valid values include: ClusterIP, NodePort, LoadBalancer, and ExternalName.
	// The default, if omitted, is ClusterIP.
	// See https://kubernetes.io/docs/concepts/services-networking/service/#publishing-services-service-types
	ServiceType v1.ServiceType `json:"serviceType,omitempty"`

	// +optional
	// TracingEnabled if set to false adds the JAEGER_DISABLED environment flag and removes the injected
	// agent container from the query component to disable tracing requests to the query service.
	// The default, if ommited, is true
	TracingEnabled *bool `json:"tracingEnabled,omitempty"`
}

// JaegerIngressSpec defines the options to be used when deploying the query ingress
type JaegerIngressSpec struct {
	// +optional
	Enabled *bool `json:"enabled,omitempty"`

	// +optional
	Security IngressSecurityType `json:"security,omitempty"`

	// +optional
	// +listType=atomic
	Hosts []string `json:"hosts,omitempty"`

	// +optional
	// +listType=atomic
	TLS []JaegerIngressTLSSpec `json:"tls,omitempty"`

	// Deprecated in favor of the TLS property
	// +optional
	SecretName string `json:"secretName,omitempty"`

	// +optional
	JaegerCommonSpec `json:",inline,omitempty"`
}

// JaegerIngressTLSSpec defines the TLS configuration to be used when deploying the query ingress
type JaegerIngressTLSSpec struct {
	// +optional
	// +listType=atomic
	Hosts []string `json:"hosts,omitempty"`

	// +optional
	SecretName string `json:"secretName,omitempty"`
}

// JaegerAllInOneSpec defines the options to be used when deploying the query
type JaegerAllInOneSpec struct {
	// +optional
	Image string `json:"image,omitempty"`

	// +optional
	JaegerCommonSpec `json:",inline,omitempty"`

	// +optional
	// TracingEnabled if set to false adds the JAEGER_DISABLED environment flag and removes the injected
	// agent container from the query component to disable tracing requests to the query service.
	// The default, if ommited, is true
	TracingEnabled *bool `json:"tracingEnabled,omitempty"`
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
	JaegerCommonSpec `json:",inline,omitempty"`

	// +optional
	Config string `json:"config,omitempty"`

	// +optional
	// ServiceType represents the type of Service to create.
	// Valid values include: ClusterIP, NodePort, LoadBalancer, and ExternalName.
	// The default, if omitted, is ClusterIP.
	// See https://kubernetes.io/docs/concepts/services-networking/service/#publishing-services-service-types
	ServiceType v1.ServiceType `json:"serviceType,omitempty"`
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
	JaegerCommonSpec `json:",inline,omitempty"`
}

// JaegerAgentSpec defines the options to be used when deploying the agent
type JaegerAgentSpec struct {
	// Strategy can be either 'DaemonSet' or 'Sidecar' (default)
	// +optional
	Strategy string `json:"strategy,omitempty"`

	// +optional
	Image string `json:"image,omitempty"`

	// +optional
	// +listType=atomic
	ImagePullSecrets []v1.LocalObjectReference `json:"imagePullSecrets,omitempty"`

	// +optional
	JaegerCommonSpec `json:",inline,omitempty"`

	// +optional
	SidecarSecurityContext *v1.SecurityContext `json:"sidecarSecurityContext,omitempty"`

	// +optional
	HostNetwork *bool `json:"hostNetwork,omitempty"`
}

// JaegerStatus defines the observed state of Jaeger
type JaegerStatus struct {
	Version string      `json:"version"`
	Phase   JaegerPhase `json:"phase"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:printcolumn:name="Status",type="string",JSONPath=".status.phase",description="Jaeger instance's status"
// +kubebuilder:printcolumn:name="Version",type="string",JSONPath=".status.version",description="Jaeger Version"
// +kubebuilder:printcolumn:name="Strategy",type="string",JSONPath=".spec.strategy",description="Jaeger deployment strategy"
// +kubebuilder:printcolumn:name="Storage",type="string",JSONPath=".spec.storage.type",description="Jaeger storage type"
// +kubebuilder:printcolumn:name="Age",type="date",JSONPath=".metadata.creationTimestamp"
// Jaeger is the Schema for the jaegers API
type Jaeger struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   JaegerSpec   `json:"spec,omitempty"`
	Status JaegerStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// JaegerList contains a list of Jaeger
type JaegerList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []Jaeger `json:"items"`
}

func init() {
	SchemeBuilder.Register(&Jaeger{}, &JaegerList{})
}
