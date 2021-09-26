package service

import (
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	v1 "github.com/jaegertracing/jaeger-operator/pkg/apis/jaegertracing/v1"
	"github.com/jaegertracing/jaeger-operator/pkg/util"
)

// NewAgentService returns a new Kubernetes service for Jaeger Agent backed by the pods matching the selector
func NewAgentService(jaeger *v1.Jaeger, selector map[string]string) *corev1.Service {
	trueVar := true
	name := util.DNSName(util.Truncate("%s-agent", 63, jaeger.Name))

	return &corev1.Service{
		TypeMeta: metav1.TypeMeta{
			Kind:       "Service",
			APIVersion: "v1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: jaeger.Namespace,
			Labels:    util.Labels(name, "service-agent", *jaeger),
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
		Spec: corev1.ServiceSpec{
			Selector:  selector,
			ClusterIP: "None",
			Ports: []corev1.ServicePort{
				{
					Name:     "zk-compact-trft",
					Port:     5775,
					Protocol: corev1.ProtocolUDP,
				},
				{
					Name: "config-rest",
					Port: 5778,
				},
				{
					Name:     "jg-compact-trft",
					Port:     6831,
					Protocol: corev1.ProtocolUDP,
				},
				{
					Name:     "jg-binary-trft",
					Port:     6832,
					Protocol: corev1.ProtocolUDP,
				},
			},
		},
	}
}
