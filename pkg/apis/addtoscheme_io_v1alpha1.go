package apis

import (
	"github.com/jaegertracing/jaeger-operator/pkg/apis/io/v1alpha1"
	esv1alpha1 "github.com/jaegertracing/jaeger-operator/pkg/storage/elasticsearch/v1alpha1"
)

func init() {
	// Register the types with the Scheme so the components can map objects to GroupVersionKinds and back
	AddToSchemes = append(AddToSchemes, v1alpha1.SchemeBuilder.AddToScheme, esv1alpha1.SchemeBuilder.AddToScheme)
}
