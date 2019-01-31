package storage

import (
	"fmt"

	rbacv1 "k8s.io/api/rbac/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"

	"github.com/jaegertracing/jaeger-operator/pkg/apis/io/v1alpha1"
)

func GetESRoles(jaeger *v1alpha1.Jaeger, sas ...string) []runtime.Object {
	roleName := fmt.Sprintf("%s-elasticsearch", jaeger.Name)
	r := &rbacv1.Role{
		ObjectMeta: metav1.ObjectMeta{
			Annotations:     map[string]string{rbacv1.AutoUpdateAnnotationKey: "true"},
			Name:            roleName,
			Namespace:       jaeger.Namespace,
			OwnerReferences: []metav1.OwnerReference{asOwner(jaeger)},
		},
		Rules: []rbacv1.PolicyRule{
			{
				// These values are virtual and defined in SearchGuard sg_config.yml under subjectAccessReviews
				// The SG invokes this API to allow the request
				// TOKEN=$(oc serviceaccounts get-token jaeger-simple-prod)
				// curl -k -v -XPOST  -H "Content-Type: application/json" -H "Authorization: Bearer $TOKEN" https://127.0.0.1:8443/apis/authorization.k8s.io/v1/selfsubjectaccessreviews -d '{"kind":"SelfSubjectAccessReview","apiVersion":"authorization.k8s.io/v1","spec":{"resourceAttributes":{"group":"jaeger.openshift.io","verb":"get","resource":"jaeger"}}}'
				APIGroups: []string{"elasticsearch.jaegertracing.io"},
				Resources: []string{"jaeger"},
				Verbs:     []string{"get"},
			},
		},
	}
	rb := &rbacv1.RoleBinding{
		ObjectMeta: metav1.ObjectMeta{
			Name:            roleName,
			Namespace:       jaeger.Namespace,
			OwnerReferences: []metav1.OwnerReference{asOwner(jaeger)},
		},
		RoleRef: rbacv1.RoleRef{
			Kind: "Role",
			Name: roleName,
		},
	}
	for _, sa := range sas {
		sb := rbacv1.Subject{
			Kind:      rbacv1.ServiceAccountKind,
			Namespace: jaeger.Namespace,
			Name:      sa,
		}
		rb.Subjects = append(rb.Subjects, sb)
	}
	return []runtime.Object{r, rb}
}
