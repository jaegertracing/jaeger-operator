package account

import (
	"fmt"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	v1 "github.com/jaegertracing/jaeger-operator/pkg/apis/jaegertracing/v1"
	"github.com/jaegertracing/jaeger-operator/pkg/util"
)

// OAuthProxy returns a service account representing a client in the context of the OAuth Proxy
func OAuthProxy(jaeger *v1.Jaeger) *corev1.ServiceAccount {
	trueVar := true
	return &corev1.ServiceAccount{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "v1",
			Kind:       "ServiceAccount",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      OAuthProxyAccountNameFor(jaeger),
			Namespace: jaeger.Namespace,
			Labels: map[string]string{
				"app":                          "jaeger",
				"app.kubernetes.io/name":       OAuthProxyAccountNameFor(jaeger),
				"app.kubernetes.io/instance":   jaeger.Name,
				"app.kubernetes.io/component":  "service-account-oauth-proxy",
				"app.kubernetes.io/part-of":    "jaeger",
				"app.kubernetes.io/managed-by": "jaeger-operator",
			},
			Annotations: map[string]string{
				"serviceaccounts.openshift.io/oauth-redirectreference.primary": getOAuthRedirectReference(jaeger),
			},
			OwnerReferences: []metav1.OwnerReference{
				metav1.OwnerReference{
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

// OAuthProxyAccountNameFor returns the service account name for this Jaeger instance in the context of the OAuth Proxy
func OAuthProxyAccountNameFor(jaeger *v1.Jaeger) string {
	sa := util.Merge([]v1.JaegerCommonSpec{jaeger.Spec.Query.JaegerCommonSpec, jaeger.Spec.JaegerCommonSpec}).ServiceAccount
	if len(sa) > 0 {
		// if we have a custom service account for the query object, that's the service name we return
		return sa
	}

	return fmt.Sprintf("%s-ui-proxy", jaeger.Name)
}

func getOAuthRedirectReference(jaeger *v1.Jaeger) string {
	return fmt.Sprintf(`{"kind":"OAuthRedirectReference","apiVersion":"v1","reference":{"kind":"Route","name":"%s"}}`, jaeger.Name)
}
