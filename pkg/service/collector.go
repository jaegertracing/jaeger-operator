package service

import (
	"fmt"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	v1 "github.com/jaegertracing/jaeger-operator/pkg/apis/jaegertracing/v1"
	"github.com/jaegertracing/jaeger-operator/pkg/util"
)

// NewCollectorServices returns a new Kubernetes service for Jaeger Collector backed by the pods matching the selector
func NewCollectorServices(jaeger *v1.Jaeger, selector map[string]string) []*corev1.Service {
	return []*corev1.Service{
		headlessCollectorService(jaeger, selector),
		clusteripCollectorService(jaeger, selector),
	}
}

func headlessCollectorService(jaeger *v1.Jaeger, selector map[string]string) *corev1.Service {
	svc := collectorService(jaeger, selector)
	svc.Name = GetNameForHeadlessCollectorService(jaeger)
	svc.Annotations = map[string]string{
		"prometheus.io/scrape": "false",
	}
	svc.Spec.ClusterIP = "None"
	return svc
}

func clusteripCollectorService(jaeger *v1.Jaeger, selector map[string]string) *corev1.Service {
	return collectorService(jaeger, selector)
}

func collectorService(jaeger *v1.Jaeger, selector map[string]string) *corev1.Service {
	trueVar := true
	return &corev1.Service{
		TypeMeta: metav1.TypeMeta{
			Kind:       "Service",
			APIVersion: "v1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      GetNameForCollectorService(jaeger),
			Namespace: jaeger.Namespace,
			Labels: map[string]string{
				"app":                          "jaeger",
				"app.kubernetes.io/name":       GetNameForCollectorService(jaeger),
				"app.kubernetes.io/instance":   jaeger.Name,
				"app.kubernetes.io/component":  "service-collector",
				"app.kubernetes.io/part-of":    "jaeger",
				"app.kubernetes.io/managed-by": "jaeger-operator",
			},
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
			ClusterIP: "",
			Ports: []corev1.ServicePort{
				{
					Name: "zipkin",
					Port: 9411,
				},
				{
					Name: "grpc",
					Port: 14250,
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
func GetNameForCollectorService(jaeger *v1.Jaeger) string {
	return util.DNSName(fmt.Sprintf("%s-collector", jaeger.Name))
}

// GetNameForHeadlessCollectorService returns the headless service name for the collector in this Jaeger instance
func GetNameForHeadlessCollectorService(jaeger *v1.Jaeger) string {
	return util.DNSName(fmt.Sprintf("%s-collector-headless", jaeger.Name))
}
