package inventory

import (
	"testing"

	"github.com/stretchr/testify/assert"
	rbacv1 "k8s.io/api/rbac/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestRoleBindingInventory(t *testing.T) {
	toCreate := rbacv1.RoleBinding{
		ObjectMeta: metav1.ObjectMeta{
			Name: "to-create",
		},
	}
	toUpdate := rbacv1.RoleBinding{
		ObjectMeta: metav1.ObjectMeta{
			Name: "to-update",
		},
	}
	toDelete := rbacv1.RoleBinding{
		ObjectMeta: metav1.ObjectMeta{
			Name: "to-delete",
		},
	}

	existing := []rbacv1.RoleBinding{toUpdate, toDelete}
	desired := []rbacv1.RoleBinding{toUpdate, toCreate}

	inv := ForRoleBindings(existing, desired)
	assert.Len(t, inv.Create, 1)
	assert.Equal(t, "to-create", inv.Create[0].Name)

	assert.Len(t, inv.Update, 1)
	assert.Equal(t, "to-update", inv.Update[0].Name)

	assert.Len(t, inv.Delete, 1)
	assert.Equal(t, "to-delete", inv.Delete[0].Name)
}
