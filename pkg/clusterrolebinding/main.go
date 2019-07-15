package clusterrolebinding

import (
	"fmt"

	"github.com/spf13/viper"
	rbac "k8s.io/api/rbac/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/jaegertracing/jaeger-operator/pkg/account"
	v1 "github.com/jaegertracing/jaeger-operator/pkg/apis/jaegertracing/v1"
	"github.com/jaegertracing/jaeger-operator/pkg/util"
)

// Get returns all the service accounts to be created for this Jaeger instance
func Get(jaeger *v1.Jaeger) []rbac.ClusterRoleBinding {
	if jaeger.Spec.Ingress.Security == v1.IngressSecurityOAuthProxy && len(jaeger.Spec.Ingress.OpenShift.DelegateURLs) > 0 {
		if viper.GetBool("auth-delegator-available") {
			return []rbac.ClusterRoleBinding{oauthProxyAuthDelegator(jaeger)}
		}

		jaeger.Logger().Warn("the requested instance specifies the delegate-urls option for the OAuth Proxy, but this operator cannot assign the proper cluster role to it (system:auth-delegator). Create a cluster role binding between the operator's service account and the cluster role 'system:auth-delegator' in order to allow instances to use 'delegate-urls'")

	}
	return []rbac.ClusterRoleBinding{}
}

func oauthProxyAuthDelegator(jaeger *v1.Jaeger) rbac.ClusterRoleBinding {
	name := util.DNSName(fmt.Sprintf("%s-%s-oauth-proxy-auth-delegator", jaeger.Namespace, jaeger.Name))
	trueVar := true

	return rbac.ClusterRoleBinding{
		ObjectMeta: metav1.ObjectMeta{
			Name: name,
			Labels: map[string]string{
				"app":                          "jaeger",
				"app.kubernetes.io/name":       name,
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
		Subjects: []rbac.Subject{{
			Kind:      "ServiceAccount",
			Name:      account.OAuthProxyAccountNameFor(jaeger),
			Namespace: jaeger.Namespace,
		}},
		RoleRef: rbac.RoleRef{Kind: "ClusterRole", Name: "system:auth-delegator"},
	}
}
