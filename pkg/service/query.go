package service

import (
	"fmt"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"

	v1 "github.com/jaegertracing/jaeger-operator/pkg/apis/jaegertracing/v1"
	"github.com/jaegertracing/jaeger-operator/pkg/util"
)

// NewQueryService returns a new Kubernetes service for Jaeger Query backed by the pods matching the selector
func NewQueryService(jaeger *v1.Jaeger, selector map[string]string) *corev1.Service {
	trueVar := true

	annotations := map[string]string{}
	if jaeger.Spec.Ingress.Security == v1.IngressSecurityOAuthProxy {
		annotations["service.alpha.openshift.io/serving-cert-secret-name"] = GetTLSSecretNameForQueryService(jaeger)
	}

	return &corev1.Service{
		TypeMeta: metav1.TypeMeta{
			Kind:       "Service",
			APIVersion: "v1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      GetNameForQueryService(jaeger),
			Namespace: jaeger.Namespace,
			Labels: map[string]string{
				"app":                          "jaeger",
				"app.kubernetes.io/name":       GetNameForQueryService(jaeger),
				"app.kubernetes.io/instance":   jaeger.Name,
				"app.kubernetes.io/component":  "service-query",
				"app.kubernetes.io/part-of":    "jaeger",
				"app.kubernetes.io/managed-by": "jaeger-operator",
			},
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
		Spec: corev1.ServiceSpec{
			Selector:  selector,
			ClusterIP: "",
			Ports: []corev1.ServicePort{
				{
					Name:       "query",
					Port:       int32(GetPortForQueryService(jaeger)),
					TargetPort: intstr.FromInt(getTargetPortForQueryService(jaeger)),
				},
			},
		},
	}
}

// GetNameForQueryService returns the query service name for this Jaeger instance
func GetNameForQueryService(jaeger *v1.Jaeger) string {
	return util.DNSName(fmt.Sprintf("%s-query", jaeger.Name))
}

// GetTLSSecretNameForQueryService returns the auto-generated TLS secret name for the Query Service for the given Jaeger instance
func GetTLSSecretNameForQueryService(jaeger *v1.Jaeger) string {
	return fmt.Sprintf("%s-ui-oauth-proxy-tls", jaeger.Name)
}

// GetPortForQueryService returns the query service name for this Jaeger instance
func GetPortForQueryService(jaeger *v1.Jaeger) int {
	if jaeger.Spec.Ingress.Security == v1.IngressSecurityOAuthProxy {
		return 443
	}
	return 16686
}

func getTargetPortForQueryService(jaeger *v1.Jaeger) int {
	if jaeger.Spec.Ingress.Security == v1.IngressSecurityOAuthProxy {
		return 8443
	}
	return 16686
}
