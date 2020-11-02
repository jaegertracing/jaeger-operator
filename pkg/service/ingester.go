package service

import (
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	v1 "github.com/jaegertracing/jaeger-operator/pkg/apis/jaegertracing/v1"
	"github.com/jaegertracing/jaeger-operator/pkg/util"
)

// NewIngesterAdminService returns a new Kubernetes service for Jaeger ingester backed by the pods matching the selector
func NewIngesterAdminService(jaeger *v1.Jaeger, selector map[string]string) *corev1.Service {
	trueVar := true
	return &corev1.Service{
		TypeMeta: metav1.TypeMeta{
			Kind:       "Service",
			APIVersion: "v1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      getNameForIngesterService(jaeger),
			Namespace: jaeger.Namespace,
			Labels:    util.Labels(getNameForIngesterService(jaeger), "ingester", *jaeger),
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
					Name: "admin",
					Port: util.GetPort("--admin.http.host-port", jaeger.Spec.Ingester.Options.ToArgs(), 14270),
				},
			},
		},
	}
}

func getNameForIngesterService(jaeger *v1.Jaeger) string {
	return util.DNSName(util.Truncate("%s-ingester", 63, jaeger.Name))
}
