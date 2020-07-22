package service

import (
	"testing"

	"github.com/spf13/viper"
	"github.com/stretchr/testify/assert"
	"k8s.io/apimachinery/pkg/types"

	v1 "github.com/jaegertracing/jaeger-operator/pkg/apis/jaegertracing/v1"
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
	}

	svc := svcs[0]
	for _, port := range svc.Spec.Ports {
		ports[port.Port] = true
	}

	for k, v := range ports {
		assert.Equal(t, v, true, "Expected port %v to be specified, but wasn't", k)
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
	assert.Len(t, svcs[1].Spec.ClusterIP, 0)
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
			"http-grpc",
			false, // in openshift?
		},
		{
			"no-tls",
			&v1.Jaeger{},
			"http-grpc",
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
			"http-grpc",
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
			"http-grpc",
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
			"https-grpc",
			false, // in openshift?
		},
		{
			"in-openshift",
			&v1.Jaeger{},
			"https-grpc",
			true, // in openshift?
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			// prepare
			if tt.inOpenShift {
				viper.Set("platform", v1.FlagPlatformOpenShift)
				defer viper.Reset()
			}

			// test
			portName := GetPortNameForGRPC(tt.input)
			assert.Equal(t, tt.expected, portName)
		})
	}
}
