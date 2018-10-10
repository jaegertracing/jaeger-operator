package ingress

import (
	"fmt"

	"k8s.io/api/extensions/v1beta1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"

	"github.com/jaegertracing/jaeger-operator/pkg/apis/io/v1alpha1"
	"github.com/jaegertracing/jaeger-operator/pkg/service"
)

// NewQueryIngress returns a new ingress object for the Query service
func NewQueryIngress(jaeger *v1alpha1.Jaeger) *v1beta1.Ingress {
	trueVar := true

	// this is pretty much the only object where we don't directly gain anything
	// from copying the map, instead of reusing the source map,
	// but for consistency's sake, let's do the same here...
	labels := map[string]string{}
	for k, v := range jaeger.Spec.Query.Labels {
		labels[k] = v
	}

	annotations := map[string]string{}
	for k, v := range jaeger.Spec.Query.Annotations {
		annotations[k] = v
	}

	return &v1beta1.Ingress{
		TypeMeta: metav1.TypeMeta{
			Kind:       "Ingress",
			APIVersion: "extensions/v1beta1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:        fmt.Sprintf("%s-query", jaeger.Name),
			Namespace:   jaeger.Namespace,
			Labels:      labels,
			Annotations: annotations,
			OwnerReferences: []metav1.OwnerReference{
				metav1.OwnerReference{
					APIVersion: jaeger.APIVersion,
					Kind:       jaeger.Kind,
					Name:       jaeger.Name,
					UID:        jaeger.UID,
					Controller: &trueVar,
				},
			},
		},
		Spec: v1beta1.IngressSpec{
			Backend: &v1beta1.IngressBackend{
				ServiceName: service.GetNameForQueryService(jaeger),
				ServicePort: intstr.FromInt(service.GetPortForQueryService(jaeger)),
			},
		},
	}
}
