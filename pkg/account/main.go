package account

import (
	"k8s.io/api/core/v1"

	"github.com/jaegertracing/jaeger-operator/pkg/apis/io/v1alpha1"
)

// Get returns all the service accounts to be created for this Jaeger instance
func Get(jaeger *v1alpha1.Jaeger) []*v1.ServiceAccount {
	accounts := []*v1.ServiceAccount{}
	if jaeger.Spec.Ingress.Security == v1alpha1.IngressSecurityOAuthProxy {
		accounts = append(accounts, OAuthProxy(jaeger))
	}
	return accounts
}
