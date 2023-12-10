package consolelink

import (
	"fmt"
	"testing"

	consolev1 "github.com/openshift/api/console/v1"
	corev1 "github.com/openshift/api/route/v1"
	"github.com/stretchr/testify/assert"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"

	v1 "github.com/jaegertracing/jaeger-operator/apis/v1"
)

func TestConsoleLinkGet(t *testing.T) {
	jaegerName := "TestConsoleLinkJaeger"
	jaegerNamespace := "TestNS"
	routerName := "TestConsoleLinkRoute"

	jaeger := v1.NewJaeger(types.NamespacedName{Name: jaegerName, Namespace: jaegerNamespace})
	route := &corev1.Route{
		TypeMeta: metav1.TypeMeta{
			Kind:       "Route",
			APIVersion: "route.openshift.io/v1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name: routerName,
		},
	}

	link := Get(jaeger, route)
	assert.Equal(t, Name(jaeger), link.Name)
	assert.Contains(t, link.Annotations, RouteAnnotation)
	assert.Equal(t, routerName, link.Annotations[RouteAnnotation])
}

func TestUpdateHref(t *testing.T) {
	jaegerName := "TestConsoleLinkJaeger"
	jaegerNamespace := "TestNS"
	routerName := "TestConsoleLinkRoute"
	jaeger := v1.NewJaeger(types.NamespacedName{Name: jaegerName, Namespace: jaegerNamespace})
	route := corev1.Route{
		TypeMeta: metav1.TypeMeta{
			Kind:       "Route",
			APIVersion: "route.openshift.io/v1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name: routerName,
		},
	}

	link := Get(jaeger, &route)
	assert.Equal(t, "", link.Spec.Href)
	route.Spec.Host = "namespace.somehostname"
	newLinks := UpdateHref([]corev1.Route{route}, []consolev1.ConsoleLink{*link})
	assert.Equal(t, fmt.Sprintf("https://%s", route.Spec.Host), newLinks[0].Spec.Href)
}
