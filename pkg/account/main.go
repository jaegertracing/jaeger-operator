package account

import (
	"fmt"

	"k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/jaegertracing/jaeger-operator/pkg/apis/io/v1alpha1"
)

// Get returns all the service accounts to be created for this Jaeger instance
func Get(jaeger *v1alpha1.Jaeger) []*v1.ServiceAccount {
	accounts := []*v1.ServiceAccount{}
	if jaeger.Spec.Ingress.Security == v1alpha1.IngressSecurityOAuthProxy {
		accounts = append(accounts, OAuthProxy(jaeger))
	}
	return append(accounts, getMain(jaeger))
}

func getMain(jaeger *v1alpha1.Jaeger) *v1.ServiceAccount {
	trueVar := true
	return &v1.ServiceAccount{
		ObjectMeta: metav1.ObjectMeta{
			Name:      JaegerServiceAccountFor(jaeger),
			Namespace: jaeger.Namespace,
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

func JaegerServiceAccountFor(jaeger *v1alpha1.Jaeger) string {
	return fmt.Sprintf("%s", jaeger.Name)
}
