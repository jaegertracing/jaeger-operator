package inventory

import (
	"fmt"

	rbac "k8s.io/api/rbac/v1"

	"github.com/jaegertracing/jaeger-operator/pkg/util"
)

// ClusterRoleBinding represents the inventory of cluster roles based on the current and desired states
type ClusterRoleBinding struct {
	Create []rbac.ClusterRoleBinding
	Update []rbac.ClusterRoleBinding
	Delete []rbac.ClusterRoleBinding
}

// ForClusterRoleBindings builds an inventory of cluster roles based on the existing and desired states
func ForClusterRoleBindings(existing []rbac.ClusterRoleBinding, desired []rbac.ClusterRoleBinding) ClusterRoleBinding {
	update := []rbac.ClusterRoleBinding{}
	mcreate := clusterRoleBindingMap(desired)
	mdelete := clusterRoleBindingMap(existing)

	for k, v := range mcreate {
		if t, ok := mdelete[k]; ok {
			tp := t.DeepCopy()
			util.InitObjectMeta(tp)

			// we can't blindly DeepCopyInto, so, we select what we bring from the new to the old object
			tp.Subjects = v.Subjects
			tp.RoleRef = v.RoleRef
			tp.ObjectMeta.OwnerReferences = v.ObjectMeta.OwnerReferences

			for k, v := range v.ObjectMeta.Annotations {
				tp.ObjectMeta.Annotations[k] = v
			}

			for k, v := range v.ObjectMeta.Labels {
				tp.ObjectMeta.Labels[k] = v
			}

			update = append(update, *tp)
			delete(mcreate, k)
			delete(mdelete, k)
		}
	}

	return ClusterRoleBinding{
		Create: clusterRoleBindingList(mcreate),
		Update: update,
		Delete: clusterRoleBindingList(mdelete),
	}
}

func clusterRoleBindingMap(deps []rbac.ClusterRoleBinding) map[string]rbac.ClusterRoleBinding {
	m := map[string]rbac.ClusterRoleBinding{}
	for _, d := range deps {
		m[fmt.Sprintf("%s.%s", d.Namespace, d.Name)] = d
	}
	return m
}

func clusterRoleBindingList(m map[string]rbac.ClusterRoleBinding) []rbac.ClusterRoleBinding {
	l := []rbac.ClusterRoleBinding{}
	for _, v := range m {
		l = append(l, v)
	}
	return l
}
