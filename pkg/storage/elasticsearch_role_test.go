package storage

import (
	"testing"

	"github.com/stretchr/testify/assert"
	rbacv1 "k8s.io/api/rbac/v1"

	"github.com/jaegertracing/jaeger-operator/pkg/apis/io/v1alpha1"
)

func TestGetESRoles_NoDeployment(t *testing.T) {
	j := v1alpha1.NewJaeger("foo")
	r := ESRole(j)
	assert.Equal(t, []rbacv1.PolicyRule{
		{
			Verbs:     []string{"get"},
			Resources: []string{"jaeger"},
			APIGroups: []string{"elasticsearch.jaegertracing.io"},
		},
	}, r.Rules)

	rb := ESRoleBinding(j)
	assert.Equal(t, 0, len(rb.Subjects))
}

func TestGetESRoles_ServiceAccount(t *testing.T) {
	j := v1alpha1.NewJaeger("foo")
	j.Namespace = "myproject"

	r := ESRole(j)
	assert.Equal(t, []rbacv1.PolicyRule{
		{
			Verbs:     []string{"get"},
			Resources: []string{"jaeger"},
			APIGroups: []string{"elasticsearch.jaegertracing.io"},
		},
	}, r.Rules)

	rb := ESRoleBinding(j, "bar")
	assert.Equal(t, "foo-elasticsearch", rb.Name)
	assert.Equal(t, []rbacv1.Subject{
		{
			Kind:      rbacv1.ServiceAccountKind,
			Name:      "bar",
			Namespace: "myproject",
		},
	}, rb.Subjects)
}
