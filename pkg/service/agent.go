package service

import (
	"fmt"

	"k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/jaegertracing/jaeger-operator/pkg/apis/io/v1alpha1"
)

// NewAgentService returns a new Kubernetes service for Jaeger Agent backed by the pods matching the selector
func NewAgentService(jaeger *v1alpha1.Jaeger, selector map[string]string) *v1.Service {
	trueVar := true

	return &v1.Service{
		TypeMeta: metav1.TypeMeta{
			Kind:       "Service",
			APIVersion: "v1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      fmt.Sprintf("%s-agent", jaeger.Name),
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
					Name: "zk-compact-trft",
					Port: 5775,
				},
				{
					Name: "config-rest",
					Port: 5778,
				},
				{
					Name: "jg-compact-trft",
					Port: 6831,
				},
				{
					Name: "jg-binary-trft",
					Port: 6832,
				},
			},
		},
	}
}
