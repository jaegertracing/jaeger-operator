package inventory

import (
	rbacv1 "k8s.io/api/rbac/v1"
)

// Role represents the service Role inventory based on the current and desired states
type Role struct {
	Create []rbacv1.Role
	Update []rbacv1.Role
	Delete []rbacv1.Role
}

// ForRoles builds a new role inventory based on the existing and desired states
func ForRoles(existing []rbacv1.Role, desired []rbacv1.Role) Role {
	update := []rbacv1.Role{}
	mcreate := roleMap(desired)
	mdelete := roleMap(existing)

	for k, v := range mcreate {
		if t, ok := mdelete[k]; ok {
			tp := t.DeepCopy()

			tp.Rules = v.Rules
			tp.ObjectMeta.Labels = v.ObjectMeta.Labels
			tp.ObjectMeta.Annotations = v.ObjectMeta.Annotations
			tp.ObjectMeta.OwnerReferences = v.ObjectMeta.OwnerReferences

			update = append(update, *tp)
			delete(mcreate, k)
			delete(mdelete, k)
		}
	}

	return Role{
		Create: roleList(mcreate),
		Update: update,
		Delete: roleList(mdelete),
	}
}

func roleMap(deps []rbacv1.Role) map[string]rbacv1.Role {
	m := map[string]rbacv1.Role{}
	for _, d := range deps {
		m[d.Name] = d
	}
	return m
}

func roleList(m map[string]rbacv1.Role) []rbacv1.Role {
	l := []rbacv1.Role{}
	for _, v := range m {
		l = append(l, v)
	}
	return l
}
