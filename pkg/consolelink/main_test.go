package consolelink

import (
	"testing"

	corev1 "github.com/openshift/api/route/v1"
	"github.com/stretchr/testify/assert"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"

	v1 "github.com/jaegertracing/jaeger-operator/pkg/apis/jaegertracing/v1"
)

func TestConsoleLinkGet(t *testing.T) {
	jaegerName := "TestConsoleLinkJaeger"
	jaegerNamespace := "TestNS"
	routerNamer := "TestConsoleLinkRoute"

	jaeger := v1.NewJaeger(types.NamespacedName{Name: jaegerName, Namespace: jaegerNamespace})
	route := &corev1.Route{
		TypeMeta: metav1.TypeMeta{
			Kind:       "Route",
			APIVersion: "route.openshift.io/v1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name: routerNamer,
		},
	}

	link := Get(jaeger, route)
	assert.Equal(t, jaeger.Namespace+".jaeger-consolelink-"+jaeger.Name, link.Name)
	assert.Contains(t, link.Annotations, RouteAnnotation)
	assert.Equal(t, routerNamer, link.Annotations[RouteAnnotation])
}

func TestUpdateHref(t *testing.T) {
	jaegerName := "TestConsoleLinkJaeger"
	jaegerNamespace := "TestNS"
	routerNamer := "TestConsoleLinkRoute"

	jaeger := v1.NewJaeger(types.NamespacedName{Name: jaegerName, Namespace: jaegerNamespace})
	route := &corev1.Route{
		TypeMeta: metav1.TypeMeta{
			Kind:       "Route",
			APIVersion: "route.openshift.io/v1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name: routerNamer,
		},
	}

	link := Get(jaeger, route)
	assert.Equal(t, link.Spec.Href, "")

	route.Spec.Host = "namespace.somehostname"
	newLink := UpdateHref(*link, *route)
	assert.Equal(t, "https://"+route.Spec.Host, newLink.Spec.Href)

}
