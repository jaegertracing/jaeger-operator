package inventory

import (
	appsv1 "k8s.io/api/apps/v1"
)

// DaemonSet represents the daemon set inventory based on the current and desired states
type DaemonSet struct {
	Create []appsv1.DaemonSet
	Update []appsv1.DaemonSet
	Delete []appsv1.DaemonSet
}

// ForDaemonSets builds a new daemon set inventory based on the existing and desired states
func ForDaemonSets(existing []appsv1.DaemonSet, desired []appsv1.DaemonSet) DaemonSet {
	update := []appsv1.DaemonSet{}
	mcreate := daemonsetMap(desired)
	mdelete := daemonsetMap(existing)

	for k, v := range mcreate {
		if t, ok := mdelete[k]; ok {
			tp := t.DeepCopy()

			// we can't blindly DeepCopyInto, so, we select what we bring from the new to the old object
			tp.Spec = v.Spec
			tp.ObjectMeta.Labels = v.ObjectMeta.Labels
			tp.ObjectMeta.Annotations = v.ObjectMeta.Annotations
			tp.ObjectMeta.OwnerReferences = v.ObjectMeta.OwnerReferences

			update = append(update, *tp)
			delete(mcreate, k)
			delete(mdelete, k)
		}
	}

	return DaemonSet{
		Create: daemonsetList(mcreate),
		Update: update,
		Delete: daemonsetList(mdelete),
	}
}

func daemonsetMap(deps []appsv1.DaemonSet) map[string]appsv1.DaemonSet {
	m := map[string]appsv1.DaemonSet{}
	for _, d := range deps {
		m[d.Name] = d
	}
	return m
}

func daemonsetList(m map[string]appsv1.DaemonSet) []appsv1.DaemonSet {
	l := []appsv1.DaemonSet{}
	for _, v := range m {
		l = append(l, v)
	}
	return l
}
