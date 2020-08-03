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

//Name derived a console link resource name from jaeger instance
func Name(jaeger *v1.Jaeger) string {
	return "jaeger-" + jaeger.Namespace + "-" + jaeger.Name
}

// Get returns a ConsoleLink specification for the current instance
func Get(jaeger *v1.Jaeger, route *routev1.Route) *consolev1.ConsoleLink {
	// If ingress is not enable there is no reason for create a console link
	if jaeger.Spec.Ingress.Enabled != nil && *jaeger.Spec.Ingress.Enabled == false {
		return nil
	}

	return &consolev1.ConsoleLink{
		ObjectMeta: metav1.ObjectMeta{
			Name:      Name(jaeger),
			Namespace: jaeger.Namespace, // Prevent warning at creation time.
			Labels: map[string]string{
				"app.kubernetes.io/instance": jaeger.Name,
				// Allow distinction between same jaeger instances in different namespaces
				"app.kubernetes.io/namespace":  jaeger.Namespace,
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
func UpdateHref(routes []routev1.Route, links []consolev1.ConsoleLink) []consolev1.ConsoleLink {
	var updated []consolev1.ConsoleLink
	mapRoutes := make(map[string]string)
	for _, route := range routes {
		mapRoutes[route.Name] = route.Spec.Host
	}
	for _, cl := range links {
		routeName := cl.Annotations[RouteAnnotation]
		// Only append it if we can found the route
		if host, ok := mapRoutes[routeName]; ok {
			cl.Spec.Href = fmt.Sprintf("https://%s", host)
			updated = append(updated, cl)
		}
		//TODO: log if not found the route associated with the link
	}
	return updated
}
