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
		if _, ok := mdelete[k]; ok {
			update = append(update, v)
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
