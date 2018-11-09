package account

import (
	"fmt"

	"k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/jaegertracing/jaeger-operator/pkg/apis/io/v1alpha1"
)

// OAuthProxy returns a service account representing a client in the context of the OAuth Proxy
func OAuthProxy(jaeger *v1alpha1.Jaeger) *v1.ServiceAccount {
	trueVar := true
	return &v1.ServiceAccount{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "v1",
			Kind:       "ServiceAccount",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      OAuthProxyAccountNameFor(jaeger),
			Namespace: jaeger.Namespace,
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
func OAuthProxyAccountNameFor(jaeger *v1alpha1.Jaeger) string {
	return fmt.Sprintf("%s-ui-proxy", jaeger.Name)
}

func getOAuthRedirectReference(jaeger *v1alpha1.Jaeger) string {
	return fmt.Sprintf(`{"kind":"OAuthRedirectReference","apiVersion":"v1","reference":{"kind":"Route","name":"%s"}}`, jaeger.Name)
}
