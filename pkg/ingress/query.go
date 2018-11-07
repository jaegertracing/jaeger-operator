package ingress

import (
	"fmt"

	"k8s.io/api/extensions/v1beta1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"

	"github.com/jaegertracing/jaeger-operator/pkg/apis/io/v1alpha1"
	"github.com/jaegertracing/jaeger-operator/pkg/service"
)

// QueryIngress builds pods for jaegertracing/jaeger-query
type QueryIngress struct {
	jaeger *v1alpha1.Jaeger
}

// NewQueryIngress builds a new QueryIngress struct based on the given spec
func NewQueryIngress(jaeger *v1alpha1.Jaeger) *QueryIngress {
	return &QueryIngress{jaeger: jaeger}
}

// Get returns an ingress specification for the current instance
func (i *QueryIngress) Get() *v1beta1.Ingress {
	if i.jaeger.Spec.Ingress.Enabled != nil && *i.jaeger.Spec.Ingress.Enabled == false {
		return nil
	}

	trueVar := true

	return &v1beta1.Ingress{
		TypeMeta: metav1.TypeMeta{
			Kind:       "Ingress",
			APIVersion: "extensions/v1beta1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      fmt.Sprintf("%s-query", i.jaeger.Name),
			Namespace: i.jaeger.Namespace,
			OwnerReferences: []metav1.OwnerReference{
				metav1.OwnerReference{
					APIVersion: i.jaeger.APIVersion,
					Kind:       i.jaeger.Kind,
					Name:       i.jaeger.Name,
					UID:        i.jaeger.UID,
					Controller: &trueVar,
				},
			},
		},
		Spec: v1beta1.IngressSpec{
			Backend: &v1beta1.IngressBackend{
				ServiceName: service.GetNameForQueryService(i.jaeger),
				ServicePort: intstr.FromInt(service.GetPortForQueryService(i.jaeger)),
			},
		},
	}
}
