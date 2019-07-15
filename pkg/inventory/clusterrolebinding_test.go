package inventory

import (
	"testing"

	"github.com/stretchr/testify/assert"
	rbac "k8s.io/api/rbac/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestClusterRoleBindingInventory(t *testing.T) {
	toCreate := rbac.ClusterRoleBinding{
		ObjectMeta: metav1.ObjectMeta{
			Name: "to-create",
		},
	}
	toUpdate := rbac.ClusterRoleBinding{
		ObjectMeta: metav1.ObjectMeta{
			Name: "to-update",
		},
		Subjects: []rbac.Subject{{
			Name: "serviceaccount1",
		}},
	}
	updated := rbac.ClusterRoleBinding{
		ObjectMeta: metav1.ObjectMeta{
			Name:        "to-update",
			Annotations: map[string]string{"gopher": "jaeger"},
			Labels:      map[string]string{"gopher": "jaeger"},
		},
		Subjects: []rbac.Subject{{
			Name: "serviceaccount2",
		}},
	}
	toDelete := rbac.ClusterRoleBinding{
		ObjectMeta: metav1.ObjectMeta{
			Name: "to-delete",
		},
	}

	existing := []rbac.ClusterRoleBinding{toUpdate, toDelete}
	desired := []rbac.ClusterRoleBinding{updated, toCreate}

	inv := ForClusterRoleBindings(existing, desired)
	assert.Len(t, inv.Create, 1)
	assert.Equal(t, "to-create", inv.Create[0].Name)

	assert.Len(t, inv.Update, 1)
	assert.Equal(t, "to-update", inv.Update[0].Name)
	assert.Equal(t, "serviceaccount2", inv.Update[0].Subjects[0].Name)

	assert.Len(t, inv.Delete, 1)
	assert.Equal(t, "to-delete", inv.Delete[0].Name)
}
