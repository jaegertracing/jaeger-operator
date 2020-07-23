package consolelink

import (
	"fmt"

	consolev1 "github.com/openshift/api/console/v1"
	routev1 "github.com/openshift/api/route/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	v1 "github.com/jaegertracing/jaeger-operator/pkg/apis/jaegertracing/v1"
)

// RouteAnnotation used to annotate the link with the route name
var RouteAnnotation = "consolelink.jaegertracing.io/route"

// Get returns an ConsoleLink specification for the current instance
func Get(jaeger *v1.Jaeger, route *routev1.Route) *consolev1.ConsoleLink {
	return &consolev1.ConsoleLink{
		ObjectMeta: metav1.ObjectMeta{
			Name:      jaeger.Namespace + ".jaeger-consolelink-" + jaeger.Name,
			Namespace: jaeger.Namespace,
			Labels: map[string]string{
				"app.kubernetes.io/instance":   jaeger.Name,
				"app.kubernetes.io/managed-by": "jaeger-operator",
			},
			Annotations: map[string]string{
				RouteAnnotation: route.Name,
			},
		},
		Spec: consolev1.ConsoleLinkSpec{
			Location: consolev1.NamespaceDashboard,
			Link: consolev1.Link{
				Text: "Jaeger [" + jaeger.Name + "]",
			},
			NamespaceDashboard: &consolev1.NamespaceDashboardSpec{
				Namespaces: []string{
					jaeger.Namespace,
				},
			},
		},
	}

}

// UpdateHref returns an ConsoleLink with the href value derived from the route
func UpdateHref(link consolev1.ConsoleLink, route routev1.Route) consolev1.ConsoleLink {
	link.Spec.Href = fmt.Sprintf("https://%s", route.Spec.Host)
	return link
}
