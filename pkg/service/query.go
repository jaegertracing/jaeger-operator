package service

import (
	"fmt"

	"k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/jaegertracing/jaeger-operator/pkg/apis/io/v1alpha1"
)

// NewQueryService returns a new Kubernetes service for Jaeger Query backed by the pods matching the selector
func NewQueryService(jaeger *v1alpha1.Jaeger, selector map[string]string) *v1.Service {
	trueVar := true

	labels := map[string]string{}
	for k, v := range jaeger.Spec.Query.Labels {
		labels[k] = v
	}
	for k, v := range selector {
		labels[k] = v
	}

	annotations := map[string]string{}
	for k, v := range jaeger.Spec.Query.Annotations {
		annotations[k] = v
	}

	return &v1.Service{
		TypeMeta: metav1.TypeMeta{
			Kind:       "Service",
			APIVersion: "v1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:        GetNameForQueryService(jaeger),
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
					Name: "query",
					Port: int32(GetPortForQueryService(jaeger)),
				},
			},
		},
	}
}

// GetNameForQueryService returns the query service name for this Jaeger instance
func GetNameForQueryService(jaeger *v1alpha1.Jaeger) string {
	return fmt.Sprintf("%s-query", jaeger.Name)
}

// GetPortForQueryService returns the query service name for this Jaeger instance
func GetPortForQueryService(jaeger *v1alpha1.Jaeger) int {
	return 16686
}
