package service

import (
	"testing"

	"github.com/spf13/viper"
	"github.com/stretchr/testify/assert"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"

	v1 "github.com/jaegertracing/jaeger-operator/apis/v1"
	"github.com/jaegertracing/jaeger-operator/pkg/autodetect"
)

func TestCollectorServiceNameAndPorts(t *testing.T) {
	name := "TestCollectorServiceNameAndPorts"
	selector := map[string]string{"app": "myapp", "jaeger": name, "jaeger-component": "collector"}

	jaeger := v1.NewJaeger(types.NamespacedName{Name: name})
	svcs := NewCollectorServices(jaeger, selector)

	assert.Equal(t, "testcollectorservicenameandports-collector-headless", svcs[0].Name)
	assert.Equal(t, "testcollectorservicenameandports-collector", svcs[1].Name)

	ports := map[int32]bool{
		9411:  false,
		14250: false,
		14267: false,
		14268: false,
		14269: false,
	}

	svc := svcs[0]
	for _, port := range svc.Spec.Ports {
		ports[port.Port] = true
	}

	for k, v := range ports {
		assert.True(t, v, "Expected port %v to be specified, but wasn't", k)
	}

	// we ensure the ports are the same for both services
	assert.Equal(t, svcs[0].Spec.Ports, svcs[1].Spec.Ports)
}

func TestCollectorServiceWithClusterIPEmptyAndNone(t *testing.T) {
	name := "TestCollectorServiceWithClusterIP"
	selector := map[string]string{"app": "myapp", "jaeger": name, "jaeger-component": "collector"}

	jaeger := v1.NewJaeger(types.NamespacedName{Name: name})
	svcs := NewCollectorServices(jaeger, selector)

	// we want two services, one headless (load balanced by the client, possibly via DNS)
	// and one with a cluster IP (load balanced by kube-proxy)
	assert.Len(t, svcs, 2)
	assert.NotEqual(t, svcs[0].Name, svcs[1].Name) // they can't have the same name
	assert.Equal(t, "None", svcs[0].Spec.ClusterIP)
	assert.Empty(t, svcs[1].Spec.ClusterIP)
}

func TestCollectorGRPCPortName(t *testing.T) {
	for _, tt := range []struct {
		name        string
		input       *v1.Jaeger
		expected    string
		inOpenShift bool
	}{
		{
			"nil",
			nil,
			"grpc-jaeger",
			false, // in openshift?
		},
		{
			"no-tls",
			&v1.Jaeger{},
			"grpc-jaeger",
			false, // in openshift?
		},
		{
			"with-tls-disabled",
			&v1.Jaeger{
				Spec: v1.JaegerSpec{
					Collector: v1.JaegerCollectorSpec{
						Options: v1.NewOptions(map[string]interface{}{"collector.grpc.tls.enabled": "false"}),
					},
				},
			},
			"grpc-jaeger",
			false, // in openshift?
		},
		{
			"with-tls-invalid",
			&v1.Jaeger{
				Spec: v1.JaegerSpec{
					Collector: v1.JaegerCollectorSpec{
						Options: v1.NewOptions(map[string]interface{}{"collector.grpc.tls.enabled": "abc"}),
					},
				},
			},
			"grpc-jaeger",
			false, // in openshift?
		},
		{
			"with-tls-enabled",
			&v1.Jaeger{
				Spec: v1.JaegerSpec{
					Collector: v1.JaegerCollectorSpec{
						Options: v1.NewOptions(map[string]interface{}{"collector.grpc.tls.enabled": "true"}),
					},
				},
			},
			"tls-grpc-jaeger",
			false, // in openshift?
		},
		{
			"in-openshift",
			&v1.Jaeger{},
			"tls-grpc-jaeger",
			true, // in openshift?
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			// prepare
			if tt.inOpenShift {
				viper.Set("platform", autodetect.OpenShiftPlatform.String())
				defer viper.Reset()
			}

			// test
			portName := GetPortNameForGRPC(tt.input)
			assert.Equal(t, tt.expected, portName)
		})
	}
}

func TestCollectorServiceLoadBalancer(t *testing.T) {
	name := "TestCollectorServiceLoadBalancer"
	selector := map[string]string{"app": "myapp", "jaeger": name, "jaeger-component": "collector"}

	jaeger := v1.NewJaeger(types.NamespacedName{Name: name})
	jaeger.Spec.Collector.ServiceType = corev1.ServiceTypeLoadBalancer
	svc := NewCollectorServices(jaeger, selector)

	// Only the non-headless service will receive the type
	assert.Equal(t, corev1.ServiceTypeLoadBalancer, svc[1].Spec.Type)
}

func TestCollectorServiceAnnotations(t *testing.T) {
	name := "TestCollectorServiceLoadBalancer"
	selector := map[string]string{"app": "myapp", "jaeger": name, "jaeger-component": "collector"}

	jaeger := v1.NewJaeger(types.NamespacedName{Name: name})
	jaeger.Spec.Collector.Annotations = map[string]string{"component": "collector"}
	svc := NewCollectorServices(jaeger, selector)

	assert.Equal(t, map[string]string{"component": "collector"}, svc[1].Annotations)
}
