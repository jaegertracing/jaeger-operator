package route

import (
	"github.com/openshift/api/route/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/jaegertracing/jaeger-operator/pkg/apis/io/v1alpha1"
	"github.com/jaegertracing/jaeger-operator/pkg/service"
)

// QueryRoute builds a route for jaegertracing/jaeger-query
type QueryRoute struct {
	jaeger *v1alpha1.Jaeger
}

// NewQueryRoute builds a new QueryRoute struct based on the given spec
func NewQueryRoute(jaeger *v1alpha1.Jaeger) *QueryRoute {
	return &QueryRoute{jaeger: jaeger}
}

// Get returns an ingress specification for the current instance
func (r *QueryRoute) Get() *v1.Route {
	if r.jaeger.Spec.Ingress.Enabled != nil && *r.jaeger.Spec.Ingress.Enabled == false {
		return nil
	}

	trueVar := true

	return &v1.Route{
		TypeMeta: metav1.TypeMeta{
			Kind:       "Route",
			APIVersion: "route.openshift.io/v1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      r.jaeger.Name,
			Namespace: r.jaeger.Namespace,
			OwnerReferences: []metav1.OwnerReference{
				metav1.OwnerReference{
					APIVersion: r.jaeger.APIVersion,
					Kind:       r.jaeger.Kind,
					Name:       r.jaeger.Name,
					UID:        r.jaeger.UID,
					Controller: &trueVar,
				},
			},
		},
		Spec: v1.RouteSpec{
			To: v1.RouteTargetReference{
				Kind: "Service",
				Name: service.GetNameForQueryService(r.jaeger),
			},
			TLS: &v1.TLSConfig{
				Termination: v1.TLSTerminationReencrypt,
			},
		},
	}
}
