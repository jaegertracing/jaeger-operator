package inventory

import (
	"testing"

	"github.com/stretchr/testify/assert"
	rbacv1 "k8s.io/api/rbac/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestRoleInventory(t *testing.T) {
	toCreate := rbacv1.Role{
		ObjectMeta: metav1.ObjectMeta{
			Name: "to-create",
		},
	}
	toUpdate := rbacv1.Role{
		ObjectMeta: metav1.ObjectMeta{
			Name: "to-update",
		},
		Rules: []rbacv1.PolicyRule{{
			Verbs: []string{"get"},
		}},
	}
	updated := rbacv1.Role{
		ObjectMeta: metav1.ObjectMeta{
			Name: "to-update",
		},
		Rules: []rbacv1.PolicyRule{{
			Verbs: []string{"delete"},
		}},
	}
	toDelete := rbacv1.Role{
		ObjectMeta: metav1.ObjectMeta{
			Name: "to-delete",
		},
	}

	existing := []rbacv1.Role{toUpdate, toDelete}
	desired := []rbacv1.Role{updated, toCreate}

	inv := ForRoles(existing, desired)
	assert.Len(t, inv.Create, 1)
	assert.Equal(t, "to-create", inv.Create[0].Name)

	assert.Len(t, inv.Update, 1)
	assert.Len(t, inv.Update[0].Rules, 1)
	assert.Len(t, inv.Update[0].Rules[0].Verbs, 1)
	assert.Equal(t, "to-update", inv.Update[0].Name)
	assert.Equal(t, "delete", inv.Update[0].Rules[0].Verbs[0])

	assert.Len(t, inv.Delete, 1)
	assert.Equal(t, "to-delete", inv.Delete[0].Name)
}
