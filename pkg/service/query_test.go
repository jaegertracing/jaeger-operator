package service

import (
	"testing"

	"github.com/stretchr/testify/assert"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/intstr"

	v1 "github.com/jaegertracing/jaeger-operator/apis/v1"
)

func TestQueryServiceNameAndPorts(t *testing.T) {
	name := "TestQueryServiceNameAndPorts"
	selector := map[string]string{"app": "myapp", "jaeger": name, "jaeger-component": "query"}

	jaeger := v1.NewJaeger(types.NamespacedName{Name: name})
	svc := NewQueryService(jaeger, selector)

	assert.Equal(t, "testqueryservicenameandports-query", svc.ObjectMeta.Name)
	assert.Len(t, svc.Spec.Ports, 3)
	assert.Equal(t, int32(16686), svc.Spec.Ports[0].Port)
	assert.Equal(t, int32(16685), svc.Spec.Ports[1].Port)
	assert.Equal(t, int32(16687), svc.Spec.Ports[2].Port)
	assert.Equal(t, "http-query", svc.Spec.Ports[0].Name)
	assert.Equal(t, "grpc-query", svc.Spec.Ports[1].Name)
	assert.Equal(t, "admin-http", svc.Spec.Ports[2].Name)
	assert.Equal(t, intstr.FromInt(16686), svc.Spec.Ports[0].TargetPort)
	assert.Equal(t, intstr.FromInt(16685), svc.Spec.Ports[1].TargetPort)
	assert.Equal(t, intstr.FromInt(16687), svc.Spec.Ports[2].TargetPort)
	assert.Empty(t, svc.Spec.ClusterIP)                         // make sure we get a cluster IP
	assert.Equal(t, corev1.ServiceTypeClusterIP, svc.Spec.Type) // make sure we get a ClusterIP service
}

func TestQueryDottedServiceName(t *testing.T) {
	name := "TestQueryDottedServiceName.With.Dots"
	selector := map[string]string{"app": "myapp", "jaeger": name, "jaeger-component": "query"}

	jaeger := v1.NewJaeger(types.NamespacedName{Name: name})
	svc := NewQueryService(jaeger, selector)

	assert.Equal(t, "testquerydottedservicename-with-dots-query", svc.ObjectMeta.Name)
}

func TestQueryServiceNameAndPortsWithOAuthProxy(t *testing.T) {
	name := "TestQueryServiceNameAndPortsWithOAuthProxy"
	selector := map[string]string{"app": "myapp", "jaeger": name, "jaeger-component": "query"}

	jaeger := v1.NewJaeger(types.NamespacedName{Name: name})
	jaeger.Spec.Ingress.Security = v1.IngressSecurityOAuthProxy
	svc := NewQueryService(jaeger, selector)

	assert.Equal(t, "testqueryservicenameandportswithoauthproxy-query", svc.ObjectMeta.Name)
	assert.Len(t, svc.Spec.Ports, 3)
	assert.Equal(t, int32(443), svc.Spec.Ports[0].Port)
	assert.Equal(t, int32(16685), svc.Spec.Ports[1].Port)
	assert.Equal(t, "https-query", svc.Spec.Ports[0].Name)
	assert.Equal(t, intstr.FromInt(8443), svc.Spec.Ports[0].TargetPort)
}

func TestQueryServiceNodePortWithIngress(t *testing.T) {
	name := "TestQueryServiceNodePortWithIngress"
	selector := map[string]string{"app": "myapp", "jaeger": name, "jaeger-component": "query"}

	jaeger := v1.NewJaeger(types.NamespacedName{Name: name})
	jaeger.Spec.Query.ServiceType = corev1.ServiceTypeNodePort
	svc := NewQueryService(jaeger, selector)

	assert.Equal(t, "testqueryservicenodeportwithingress-query", svc.ObjectMeta.Name)
	assert.Len(t, svc.Spec.Ports, 3)
	assert.Equal(t, int32(16686), svc.Spec.Ports[0].Port)
	assert.Equal(t, int32(16685), svc.Spec.Ports[1].Port)
	assert.Equal(t, int32(16687), svc.Spec.Ports[2].Port)
	assert.Equal(t, "http-query", svc.Spec.Ports[0].Name)
	assert.Equal(t, "grpc-query", svc.Spec.Ports[1].Name)
	assert.Equal(t, "admin-http", svc.Spec.Ports[2].Name)
	assert.Equal(t, int32(0), svc.Spec.Ports[0].NodePort)
	assert.Equal(t, int32(0), svc.Spec.Ports[1].NodePort)
	assert.Equal(t, intstr.FromInt(16686), svc.Spec.Ports[0].TargetPort)
	assert.Equal(t, intstr.FromInt(16685), svc.Spec.Ports[1].TargetPort)
	assert.Equal(t, intstr.FromInt(16687), svc.Spec.Ports[2].TargetPort)
	assert.Equal(t, corev1.ServiceTypeNodePort, svc.Spec.Type) // make sure we get a NodePort service
}

func TestQueryServiceLoadBalancerWithIngress(t *testing.T) {
	name := "TestQueryServiceNodePortWithIngress"
	selector := map[string]string{"app": "myapp", "jaeger": name, "jaeger-component": "query"}

	jaeger := v1.NewJaeger(types.NamespacedName{Name: name})
	jaeger.Spec.Query.ServiceType = corev1.ServiceTypeLoadBalancer
	svc := NewQueryService(jaeger, selector)

	assert.Equal(t, "testqueryservicenodeportwithingress-query", svc.ObjectMeta.Name)
	assert.Len(t, svc.Spec.Ports, 3)
	assert.Equal(t, int32(16686), svc.Spec.Ports[0].Port)
	assert.Equal(t, int32(16685), svc.Spec.Ports[1].Port)
	assert.Equal(t, int32(16687), svc.Spec.Ports[2].Port)
	assert.Equal(t, "http-query", svc.Spec.Ports[0].Name)
	assert.Equal(t, "grpc-query", svc.Spec.Ports[1].Name)
	assert.Equal(t, "admin-http", svc.Spec.Ports[2].Name)
	assert.Equal(t, intstr.FromInt(16686), svc.Spec.Ports[0].TargetPort)
	assert.Equal(t, intstr.FromInt(16685), svc.Spec.Ports[1].TargetPort)
	assert.Equal(t, intstr.FromInt(16687), svc.Spec.Ports[2].TargetPort)
	assert.Equal(t, corev1.ServiceTypeLoadBalancer, svc.Spec.Type) // make sure we get a LoadBalancer service
}

func TestQueryServiceSpecifiedNodePortWithIngress(t *testing.T) {
	name := "TestQueryServiceSpecifiedNodePortWithIngress"
	selector := map[string]string{"app": "myapp", "jaeger": name, "jaeger-component": "query"}

	jaeger := v1.NewJaeger(types.NamespacedName{Name: name})
	jaeger.Spec.Query.ServiceType = corev1.ServiceTypeNodePort
	jaeger.Spec.Query.NodePort = 32767
	svc := NewQueryService(jaeger, selector)

	assert.Equal(t, "testqueryservicespecifiednodeportwithingress-query", svc.ObjectMeta.Name)
	assert.Len(t, svc.Spec.Ports, 3)
	assert.Equal(t, int32(16686), svc.Spec.Ports[0].Port)
	assert.Equal(t, int32(16685), svc.Spec.Ports[1].Port)
	assert.Equal(t, int32(16687), svc.Spec.Ports[2].Port)
	assert.Equal(t, "http-query", svc.Spec.Ports[0].Name)
	assert.Equal(t, "grpc-query", svc.Spec.Ports[1].Name)
	assert.Equal(t, "admin-http", svc.Spec.Ports[2].Name)
	assert.Equal(t, int32(32767), svc.Spec.Ports[0].NodePort) // make sure we get the same NodePort as set above
	assert.Equal(t, intstr.FromInt(16686), svc.Spec.Ports[0].TargetPort)
	assert.Equal(t, intstr.FromInt(16685), svc.Spec.Ports[1].TargetPort)
	assert.Equal(t, intstr.FromInt(16687), svc.Spec.Ports[2].TargetPort)
	assert.Equal(t, corev1.ServiceTypeNodePort, svc.Spec.Type)
}

func TestQueryServiceSpecAnnotations(t *testing.T) {
	name := "TestQueryServiceSpecAnnotations"
	selector := map[string]string{"app": "myapp", "jaeger": name, "jaeger-component": "query"}

	jaeger := v1.NewJaeger(types.NamespacedName{Name: name})
	jaeger.Spec.Query.Annotations = map[string]string{"component": "jaeger"}
	svc := NewQueryService(jaeger, selector)

	assert.Equal(t, "testqueryservicespecannotations-query", svc.ObjectMeta.Name)
	assert.Len(t, svc.Spec.Ports, 3)
	assert.Equal(t, int32(16686), svc.Spec.Ports[0].Port)
	assert.Equal(t, int32(16685), svc.Spec.Ports[1].Port)
	assert.Equal(t, int32(16687), svc.Spec.Ports[2].Port)
	assert.Equal(t, "http-query", svc.Spec.Ports[0].Name)
	assert.Equal(t, "grpc-query", svc.Spec.Ports[1].Name)
	assert.Equal(t, "admin-http", svc.Spec.Ports[2].Name)
	assert.Equal(t, intstr.FromInt(16686), svc.Spec.Ports[0].TargetPort)
	assert.Equal(t, intstr.FromInt(16685), svc.Spec.Ports[1].TargetPort)
	assert.Equal(t, intstr.FromInt(16687), svc.Spec.Ports[2].TargetPort)
	assert.Equal(t, map[string]string{"component": "jaeger"}, svc.Annotations)
}
