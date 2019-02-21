package inventory

import (
	rbacv1 "k8s.io/api/rbac/v1"
)

// RoleBinding represents the service RoleBinding inventory based on the current and desired states
type RoleBinding struct {
	Create []rbacv1.RoleBinding
	Update []rbacv1.RoleBinding
	Delete []rbacv1.RoleBinding
}

// ForRoleBindings builds a new RoleBinding inventory based on the existing and desired states
func ForRoleBindings(existing []rbacv1.RoleBinding, desired []rbacv1.RoleBinding) RoleBinding {
	update := []rbacv1.RoleBinding{}
	mcreate := roleBindingMap(desired)
	mdelete := roleBindingMap(existing)

	for k, v := range mcreate {
		if t, ok := mdelete[k]; ok {
			tp := t.DeepCopy()

			tp.Subjects = v.Subjects
			tp.ObjectMeta.Labels = v.ObjectMeta.Labels
			tp.ObjectMeta.Annotations = v.ObjectMeta.Annotations
			tp.ObjectMeta.OwnerReferences = v.ObjectMeta.OwnerReferences

			update = append(update, *tp)
			delete(mcreate, k)
			delete(mdelete, k)
		}
	}

	return RoleBinding{
		Create: roleBindingList(mcreate),
		Update: update,
		Delete: roleBindingList(mdelete),
	}
}

func roleBindingMap(deps []rbacv1.RoleBinding) map[string]rbacv1.RoleBinding {
	m := map[string]rbacv1.RoleBinding{}
	for _, d := range deps {
		m[d.Name] = d
	}
	return m
}

func roleBindingList(m map[string]rbacv1.RoleBinding) []rbacv1.RoleBinding {
	l := []rbacv1.RoleBinding{}
	for _, v := range m {
		l = append(l, v)
	}
	return l
}
