package service

import (
	"fmt"

	"k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/jaegertracing/jaeger-operator/pkg/apis/io/v1alpha1"
)

// NewCollectorService returns a new Kubernetes service for Jaeger Collector backed by the pods matching the selector
func NewCollectorService(jaeger *v1alpha1.Jaeger, selector map[string]string) *v1.Service {
	trueVar := true

	labels := map[string]string{}
	for k, v := range jaeger.Spec.Collector.Labels {
		labels[k] = v
	}
	for k, v := range selector {
		labels[k] = v
	}

	annotations := map[string]string{}
	for k, v := range jaeger.Spec.Collector.Annotations {
		annotations[k] = v
	}

	return &v1.Service{
		TypeMeta: metav1.TypeMeta{
			Kind:       "Service",
			APIVersion: "v1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:        GetNameForCollectorService(jaeger),
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
		Spec: v1.ServiceSpec{
			Selector:  selector,
			ClusterIP: "None",
			Ports: []v1.ServicePort{
				{
					Name: "zipkin",
					Port: 9411,
				},
				{
					Name: "c-tchan-trft",
					Port: 14267,
				},
				{
					Name: "c-binary-trft",
					Port: 14268,
				},
			},
		},
	}
}

// GetNameForCollectorService returns the service name for the collector in this Jaeger instance
func GetNameForCollectorService(jaeger *v1alpha1.Jaeger) string {
	return fmt.Sprintf("%s-collector", jaeger.Name)
}
