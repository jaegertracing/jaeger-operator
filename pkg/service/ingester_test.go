package service

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/jaegertracing/jaeger-operator/pkg/apis/io/v1alpha1"
)

func TestIngesterServiceNameAndPorts(t *testing.T) {
	name := "TestIngesterServiceNameAndPorts"
	selector := map[string]string{"app": "myapp", "jaeger": name, "jaeger-component": "ingester"}

	jaeger := &v1alpha1.Jaeger{
		ObjectMeta: metav1.ObjectMeta{
			Name: name,
		},
		Spec: v1alpha1.JaegerSpec{
			Strategy: "streaming",
			Ingester: v1alpha1.JaegerIngesterSpec{
				Options: v1alpha1.NewOptions(map[string]interface{}{
					"any": "option",
				}),
			},
		},
	}
	svc := NewIngesterService(jaeger, selector)
	assert.Equal(t, svc.ObjectMeta.Name, fmt.Sprintf("%s-ingester", name))

	ports := map[int32]bool{
		14270: false,
		14271: false,
	}

	for _, port := range svc.Spec.Ports {
		ports[port.Port] = true
	}

	for k, v := range ports {
		assert.Equal(t, v, true, "Expected port %v to be specified, but wasn't", k)
	}

}

func TestIngesterNoServiceWrongStrategy(t *testing.T) {
	name := "TestIngesterNoServiceWrongStrategy"
	selector := map[string]string{"app": "myapp", "jaeger": name, "jaeger-component": "ingester"}

	jaeger := &v1alpha1.Jaeger{
		ObjectMeta: metav1.ObjectMeta{
			Name: name,
		},
		Spec: v1alpha1.JaegerSpec{
			Strategy: "production",
			Ingester: v1alpha1.JaegerIngesterSpec{
				Options: v1alpha1.NewOptions(map[string]interface{}{
					"any": "option",
				}),
			},
		},
	}
	assert.Nil(t, NewIngesterService(jaeger, selector))
}

func TestIngesterNoServiceMissingStrategy(t *testing.T) {
	name := "TestIngesterNoServiceMissingStrategy"
	selector := map[string]string{"app": "myapp", "jaeger": name, "jaeger-component": "ingester"}

	jaeger := &v1alpha1.Jaeger{
		ObjectMeta: metav1.ObjectMeta{
			Name: name,
		},
		Spec: v1alpha1.JaegerSpec{},
	}
	assert.Nil(t, NewIngesterService(jaeger, selector))
}
