package service

import (
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"

	v1 "github.com/jaegertracing/jaeger-operator/apis/v1"
	"github.com/jaegertracing/jaeger-operator/pkg/util"
)

// NewQueryService returns a new Kubernetes service for Jaeger Query backed by the pods matching the selector
func NewQueryService(jaeger *v1.Jaeger, selector map[string]string) *corev1.Service {
	trueVar := true

	annotations := map[string]string{}
	if jaeger.Spec.Query.Annotations != nil {
		annotations = jaeger.Spec.Query.Annotations
	}
	if jaeger.Spec.Ingress.Security == v1.IngressSecurityOAuthProxy {
		annotations["service.alpha.openshift.io/serving-cert-secret-name"] = GetTLSSecretNameForQueryService(jaeger)
	}

	ports := []corev1.ServicePort{
		{
			Name:       GetPortNameForQueryService(jaeger),
			Port:       int32(GetPortForQueryService(jaeger)),
			TargetPort: intstr.FromInt(getTargetPortForQueryService(jaeger)),
		},
		{
			Name:       "grpc-query",
			Port:       int32(16685),
			TargetPort: intstr.FromInt(16685),
		},
	}
	if jaeger.Spec.Query.ServiceType == corev1.ServiceTypeNodePort {
		ports[0].NodePort = GetNodePortForQueryService(jaeger)
		ports[1].NodePort = GetGRPCNodePortForQueryService(jaeger)

	}

	return &corev1.Service{
		TypeMeta: metav1.TypeMeta{
			Kind:       "Service",
			APIVersion: "v1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:        GetNameForQueryService(jaeger),
			Namespace:   jaeger.Namespace,
			Labels:      util.Labels(GetNameForQueryService(jaeger), "service-query", *jaeger),
			Annotations: annotations,
			OwnerReferences: []metav1.OwnerReference{
				{
					APIVersion: jaeger.APIVersion,
					Kind:       jaeger.Kind,
					Name:       jaeger.Name,
					UID:        jaeger.UID,
					Controller: &trueVar,
				},
			},
		},
		Spec: corev1.ServiceSpec{
			Selector: selector,
			Type:     getTypeForQueryService(jaeger),
			Ports:    ports,
		},
	}
}

// GetNameForQueryService returns the query service name for this Jaeger instance
func GetNameForQueryService(jaeger *v1.Jaeger) string {
	return util.DNSName(util.Truncate("%s-query", 63, jaeger.Name))
}

// GetTLSSecretNameForQueryService returns the auto-generated TLS secret name for the Query Service for the given Jaeger instance
func GetTLSSecretNameForQueryService(jaeger *v1.Jaeger) string {
	return util.DNSName(util.Truncate("%s-ui-oauth-proxy-tls", 63, jaeger.Name))
}

// GetPortForQueryService returns the query service port number for this Jaeger instance
func GetPortForQueryService(jaeger *v1.Jaeger) int {
	if jaeger.Spec.Ingress.Security == v1.IngressSecurityOAuthProxy {
		return 443
	}
	return 16686
}

// GetPortNameForQueryService returns the query service port name for this Jaeger instance
func GetPortNameForQueryService(jaeger *v1.Jaeger) string {
	if jaeger.Spec.Ingress.Security == v1.IngressSecurityOAuthProxy {
		return "https-query"
	}
	return "http-query"
}

func getTargetPortForQueryService(jaeger *v1.Jaeger) int {
	if jaeger.Spec.Ingress.Security == v1.IngressSecurityOAuthProxy {
		return 8443
	}
	return 16686
}

func getTypeForQueryService(jaeger *v1.Jaeger) corev1.ServiceType {
	if jaeger.Spec.Query.ServiceType != "" {
		return jaeger.Spec.Query.ServiceType
	}
	return corev1.ServiceTypeClusterIP
}

// GetNodePortForQueryService returns the query service NodePort for this Jaeger instance
func GetNodePortForQueryService(jaeger *v1.Jaeger) int32 {
	return jaeger.Spec.Query.NodePort
}

// GetGRPCNodePortForQueryService returns the query service grpc NodePort for this Jaeger instance
func GetGRPCNodePortForQueryService(jaeger *v1.Jaeger) int32 {
	return jaeger.Spec.Query.GRPCNodePort
}
