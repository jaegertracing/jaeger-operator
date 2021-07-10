// NOTE: Boilerplate only.  Ignore this file.

// Package v1beta2 contains API Schema definitions for the kafka v1beta2 API group
// +k8s:deepcopy-gen=package,register
// +groupName=kafka.strimzi.io
package v1beta2

import (
	"k8s.io/apimachinery/pkg/runtime/schema"
	"sigs.k8s.io/controller-runtime/pkg/scheme"
)

var (
	// SchemeGroupVersion is group version used to register these objects
	SchemeGroupVersion = schema.GroupVersion{Group: "kafka.strimzi.io", Version: "v1beta2"}

	// SchemeBuilder is used to add go types to the GroupVersionKind scheme
	SchemeBuilder = &scheme.Builder{GroupVersion: SchemeGroupVersion}
)
