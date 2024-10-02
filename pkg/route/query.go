package route

import (
	corev1 "github.com/openshift/api/route/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"

	v1 "github.com/jaegertracing/jaeger-operator/apis/v1"
	"github.com/jaegertracing/jaeger-operator/pkg/service"
	"github.com/jaegertracing/jaeger-operator/pkg/util"
)

// QueryRoute builds a route for jaegertracing/jaeger-query
type QueryRoute struct {
	jaeger *v1.Jaeger
}

// NewQueryRoute builds a new QueryRoute struct based on the given spec
func NewQueryRoute(jaeger *v1.Jaeger) *QueryRoute {
	return &QueryRoute{jaeger: jaeger}
}

// Get returns an ingress specification for the current instance
func (r *QueryRoute) Get() *corev1.Route {
	if r.jaeger.Spec.Ingress.Enabled != nil && !*r.jaeger.Spec.Ingress.Enabled {
		return nil
	}

	trueVar := true

	var termination corev1.TLSTerminationType
	if r.jaeger.Spec.Ingress.Security == v1.IngressSecurityOAuthProxy {
		termination = corev1.TLSTerminationReencrypt
	} else {
		termination = corev1.TLSTerminationEdge
	}

	var name string

	host := ""
	if len(r.jaeger.Spec.Ingress.Hosts) > 0 {
		host = r.jaeger.Spec.Ingress.Hosts[0]
	}

	if len(r.jaeger.Namespace) >= 63 {
		// the route is doomed already, nothing we can do...
		name = r.jaeger.Name
		if host == "" {
			r.jaeger.Logger().V(1).Info(
				"the route's hostname will have more than 63 chars and will not be valid",
				"name", name,
			)
		}
	} else {
		// -namespace is added to the host by OpenShift
		name = util.Truncate(r.jaeger.Name, 62-len(r.jaeger.Namespace))
	}
	name = util.DNSName(name)

	return &corev1.Route{
		TypeMeta: metav1.TypeMeta{
			Kind:       "Route",
			APIVersion: "route.openshift.io/v1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:        name,
			Namespace:   r.jaeger.Namespace,
			Labels:      util.Labels(r.jaeger.Name, "query-route", *r.jaeger),
			Annotations: r.jaeger.Spec.Ingress.Annotations,
			OwnerReferences: []metav1.OwnerReference{
				{
					APIVersion: r.jaeger.APIVersion,
					Kind:       r.jaeger.Kind,
					Name:       r.jaeger.Name,
					UID:        r.jaeger.UID,
					Controller: &trueVar,
				},
			},
		},
		Spec: corev1.RouteSpec{
			To: corev1.RouteTargetReference{
				Kind: "Service",
				Name: service.GetNameForQueryService(r.jaeger),
			},
			Port: &corev1.RoutePort{
				TargetPort: intstr.FromString(service.GetPortNameForQueryService(r.jaeger)),
			},
			TLS: &corev1.TLSConfig{
				Termination: termination,
			},
			Host: host,
		},
	}
}
