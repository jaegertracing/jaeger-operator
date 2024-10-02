package route

import (
	"testing"

	corev1 "github.com/openshift/api/route/v1"
	"github.com/stretchr/testify/assert"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/intstr"

	v1 "github.com/jaegertracing/jaeger-operator/apis/v1"
)

func TestQueryRoute(t *testing.T) {
	name := "TestQueryRoute"
	jaeger := v1.NewJaeger(types.NamespacedName{Name: name})
	route := NewQueryRoute(jaeger)

	dep := route.Get()

	assert.Contains(t, dep.Spec.To.Name, "testqueryroute-query")
}

func TestQueryRouteDisabled(t *testing.T) {
	enabled := false
	name := "TestQueryRouteDisabled"
	jaeger := v1.NewJaeger(types.NamespacedName{Name: name})
	jaeger.Spec.Ingress.Enabled = &enabled
	route := NewQueryRoute(jaeger)

	dep := route.Get()

	assert.Nil(t, dep)
}

func TestQueryRouteEnabled(t *testing.T) {
	enabled := true
	name := "TestQueryRouteEnabled"
	jaeger := v1.NewJaeger(types.NamespacedName{Name: name})
	jaeger.Spec.Ingress.Enabled = &enabled
	route := NewQueryRoute(jaeger)

	dep := route.Get()

	assert.NotNil(t, dep)
}

func TestQueryRouteWithOAuthProxy(t *testing.T) {
	jaeger := v1.NewJaeger(types.NamespacedName{Name: "TestQueryRouteWithOAuthProxy"})
	jaeger.Spec.Ingress.Security = v1.IngressSecurityOAuthProxy
	jaeger.Spec.Ingress.Annotations = map[string]string{"timeout": "10s"}
	route := NewQueryRoute(jaeger)

	r := route.Get()
	assert.Equal(t, corev1.TLSTerminationReencrypt, r.Spec.TLS.Termination)
	assert.Equal(t, intstr.FromString("https-query"), r.Spec.Port.TargetPort)
	assert.Equal(t, map[string]string{"timeout": "10s"}, r.Annotations)
}

func TestQueryRouteWithoutOAuthProxy(t *testing.T) {
	jaeger := v1.NewJaeger(types.NamespacedName{Name: "TestQueryRouteWithOAuthProxy"})
	jaeger.Spec.Ingress.Security = v1.IngressSecurityNone
	route := NewQueryRoute(jaeger)

	r := route.Get()
	assert.Equal(t, corev1.TLSTerminationEdge, r.Spec.TLS.Termination)
	assert.Equal(t, intstr.FromString("http-query"), r.Spec.Port.TargetPort)
}
