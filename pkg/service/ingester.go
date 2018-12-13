package service

import (
	"fmt"

	"k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/jaegertracing/jaeger-operator/pkg/apis/io/v1alpha1"
)

// NewIngesterService returns a new Kubernetes service for Jaeger Ingester backed by the pods matching the selector
func NewIngesterService(jaeger *v1alpha1.Jaeger, selector map[string]string) *v1.Service {
	// Only create a service if ingester options have been defined
	if len(jaeger.Spec.Ingester.Options.Map()) == 0 {
		return nil
	}

	trueVar := true

	return &v1.Service{
		TypeMeta: metav1.TypeMeta{
			Kind:       "Service",
			APIVersion: "v1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      GetNameForIngesterService(jaeger),
			Namespace: jaeger.Namespace,
			Labels:    selector,
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
					Name: "c-tchan-trft",
					Port: 14267,
				},
			},
		},
	}
}

// GetNameForIngesterService returns the service name for the ingester in this Jaeger instance
func GetNameForIngesterService(jaeger *v1alpha1.Jaeger) string {
	return fmt.Sprintf("%s-ingester", jaeger.Name)
}
