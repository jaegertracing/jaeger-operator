package account

import (
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	v1 "github.com/jaegertracing/jaeger-operator/pkg/apis/jaegertracing/v1"
	"github.com/jaegertracing/jaeger-operator/pkg/util"
)

// Get returns all the service accounts to be created for this Jaeger instance
func Get(jaeger *v1.Jaeger) []*corev1.ServiceAccount {
	accounts := []*corev1.ServiceAccount{}
	if jaeger.Spec.Ingress.Security == v1.IngressSecurityOAuthProxy {
		sa := util.Merge([]v1.JaegerCommonSpec{jaeger.Spec.Query.JaegerCommonSpec, jaeger.Spec.JaegerCommonSpec}).ServiceAccount
		if len(sa) == 0 {
			// if there's a service account specified for the query component, that's the one we use
			// otherwise, we use a custom SA for the OAuth Proxy
			accounts = append(accounts, OAuthProxy(jaeger))
		}
	}
	return append(accounts, getMain(jaeger))
}

func getMain(jaeger *v1.Jaeger) *corev1.ServiceAccount {
	trueVar := true
	return &corev1.ServiceAccount{
		ObjectMeta: metav1.ObjectMeta{
			Name:      JaegerServiceAccountFor(jaeger, ""),
			Namespace: jaeger.Namespace,
			Labels: map[string]string{
				"app":                          "jaeger",
				"app.kubernetes.io/name":       JaegerServiceAccountFor(jaeger, ""),
				"app.kubernetes.io/instance":   jaeger.Name,
				"app.kubernetes.io/component":  "service-account",
				"app.kubernetes.io/part-of":    "jaeger",
				"app.kubernetes.io/managed-by": "jaeger-operator",
			},
			OwnerReferences: []metav1.OwnerReference{
				{
					APIVersion: jaeger.APIVersion,
					Kind:       jaeger.Kind,
					Name:       jaeger.Name,
					UID:        jaeger.UID,
					Controller: &trueVar,
				},
			},
		},
	}
}

// JaegerServiceAccountFor prints service name for Jaeger instance
func JaegerServiceAccountFor(jaeger *v1.Jaeger, component Component) string {
	sa := ""
	switch component {
	case CollectorComponent:
		sa = util.Merge([]v1.JaegerCommonSpec{jaeger.Spec.Collector.JaegerCommonSpec, jaeger.Spec.JaegerCommonSpec}).ServiceAccount
	case QueryComponent:
		sa = util.Merge([]v1.JaegerCommonSpec{jaeger.Spec.Query.JaegerCommonSpec, jaeger.Spec.JaegerCommonSpec}).ServiceAccount
	case IngesterComponent:
		sa = util.Merge([]v1.JaegerCommonSpec{jaeger.Spec.Ingester.JaegerCommonSpec, jaeger.Spec.JaegerCommonSpec}).ServiceAccount
	case AllInOneComponent:
		sa = util.Merge([]v1.JaegerCommonSpec{jaeger.Spec.AllInOne.JaegerCommonSpec, jaeger.Spec.JaegerCommonSpec}).ServiceAccount
	case AgentComponent:
		sa = util.Merge([]v1.JaegerCommonSpec{jaeger.Spec.Agent.JaegerCommonSpec, jaeger.Spec.JaegerCommonSpec}).ServiceAccount
	}

	if sa == "" {
		return jaeger.Name
	}
	return sa
}
