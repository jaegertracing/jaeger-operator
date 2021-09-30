package service

import (
	"fmt"
	"strconv"

	"github.com/spf13/viper"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	v1 "github.com/jaegertracing/jaeger-operator/pkg/apis/jaegertracing/v1"
	"github.com/jaegertracing/jaeger-operator/pkg/util"
)

// NewCollectorServices returns a new Kubernetes service for Jaeger Collector backed by the pods matching the selector
func NewCollectorServices(jaeger *v1.Jaeger, selector map[string]string) []*corev1.Service {
	services := []*corev1.Service{
		headlessCollectorService(jaeger, selector),
		clusteripCollectorService(jaeger, selector),
	}
	if jaeger.Spec.ServiceMonitor.Enabled != nil && *jaeger.Spec.ServiceMonitor.Enabled {
		services = append(services, collectorAdminService(jaeger, selector))
	}
	return services
}

func headlessCollectorService(jaeger *v1.Jaeger, selector map[string]string) *corev1.Service {
	svc := collectorService(jaeger, selector)
	svc.Name = GetNameForHeadlessCollectorService(jaeger)
	svc.Annotations = map[string]string{
		"prometheus.io/scrape":                               "false",
		"service.beta.openshift.io/serving-cert-secret-name": fmt.Sprintf("%s-tls", svc.Name),
	}
	svc.Spec.ClusterIP = "None"
	return svc
}

func clusteripCollectorService(jaeger *v1.Jaeger, selector map[string]string) *corev1.Service {
	svc := collectorService(jaeger, selector)
	svc.Spec.Type = getTypeForCollectorService(jaeger)
	return svc
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
			Labels:    util.Labels(GetNameForCollectorService(jaeger), "service-collector", *jaeger),
			OwnerReferences: []metav1.OwnerReference{{
				APIVersion: jaeger.APIVersion,
				Kind:       jaeger.Kind,
				Name:       jaeger.Name,
				UID:        jaeger.UID,
				Controller: &trueVar,
			}},
		},
		Spec: corev1.ServiceSpec{
			Selector:  selector,
			ClusterIP: "",
			Ports: []corev1.ServicePort{
				{
					Name: "http",
					Port: 9411,
				},
				{
					Name: GetPortNameForGRPC(jaeger),
					Port: 14250,
				},
				{
					Name: "c-tchan-trft",
					Port: 14267,
				},
				{
					Name: "http-c-binary-trft",
					Port: 14268,
				},
			},
		},
	}

}

func collectorAdminService(jaeger *v1.Jaeger, selector map[string]string) *corev1.Service {
	trueVar := true
	return &corev1.Service{
		TypeMeta: metav1.TypeMeta{
			Kind:       "Service",
			APIVersion: "v1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      GetNameForCollectorAdminService(jaeger),
			Namespace: jaeger.Namespace,
			Labels:    util.Labels(GetNameForCollectorAdminService(jaeger), "service-collector", *jaeger),
			OwnerReferences: []metav1.OwnerReference{{
				APIVersion: jaeger.APIVersion,
				Kind:       jaeger.Kind,
				Name:       jaeger.Name,
				UID:        jaeger.UID,
				Controller: &trueVar,
			}},
		},
		Spec: corev1.ServiceSpec{
			Selector:  selector,
			ClusterIP: "",
			Ports: []corev1.ServicePort{
				{
					Name: "admin",
					Port: util.GetPort("--admin.http.host-port=", jaeger.Spec.Collector.Options.ToArgs(), 14269),
				},
			},
		},
	}
}

// GetNameForCollectorService returns the service name for the collector in this Jaeger instance
func GetNameForCollectorService(jaeger *v1.Jaeger) string {
	return util.DNSName(util.Truncate("%s-collector", 63, jaeger.Name))
}

// GetNameForHeadlessCollectorService returns the headless service name for the collector in this Jaeger instance
func GetNameForHeadlessCollectorService(jaeger *v1.Jaeger) string {
	return util.DNSName(util.Truncate("%s-collector-headless", 63, jaeger.Name))
}

// GetNameForCollectorAdminService returns the collector admin service name for this Jaeger instance
func GetNameForCollectorAdminService(jaeger *v1.Jaeger) string {
	return util.DNSName(util.Truncate("%s-collector-admin", 63, jaeger.Name))
}

// GetPortNameForGRPC returns the port name for 'grpc'. It may either be http-grpc or https-grpc, based on whether
// TLS is enabled for the agent-collector gRPC communication
func GetPortNameForGRPC(jaeger *v1.Jaeger) string {
	if viper.GetString("platform") == v1.FlagPlatformOpenShift {
		// we always have TLS certs when running on OpenShift, so, TLS is always enabled
		return "grpc-https"
	}

	// if we don't have a jaeger provided, it's certainly not TLS...
	if nil == jaeger {
		return "grpc-http"
	}

	// perhaps the user has provisioned the certs and configured the CR manually?
	// for that, we check whether the CLI option `collector.grpc.tls.enabled` was set for the collector
	if val, ok := jaeger.Spec.Collector.Options.StringMap()["collector.grpc.tls.enabled"]; ok {
		enabled, err := strconv.ParseBool(val)
		if err != nil {
			return "grpc-http" // not "true", defaults to false
		}

		if enabled {
			return "grpc-https" // explicit true
		}

		return "grpc-http" // explicit false
	}

	// doesn't look like we have TLS enabled
	return "grpc-http"
}

func getTypeForCollectorService(jaeger *v1.Jaeger) corev1.ServiceType {
	if jaeger.Spec.Collector.ServiceType != "" {
		return jaeger.Spec.Collector.ServiceType
	}
	return corev1.ServiceTypeClusterIP
}
