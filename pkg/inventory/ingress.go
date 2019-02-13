package inventory

import (
	"k8s.io/api/extensions/v1beta1"
)

// Ingress represents the inventory of ingresses based on the current and desired states
type Ingress struct {
	Create []v1beta1.Ingress
	Update []v1beta1.Ingress
	Delete []v1beta1.Ingress
}

// ForIngresses builds an inventory of ingresses based on the existing and desired states
func ForIngresses(existing []v1beta1.Ingress, desired []v1beta1.Ingress) Ingress {
	update := []v1beta1.Ingress{}
	mcreate := ingressMap(desired)
	mdelete := ingressMap(existing)

	for k, v := range mcreate {
		if _, ok := mdelete[k]; ok {
			update = append(update, v)
			delete(mcreate, k)
			delete(mdelete, k)
		}
	}

	return Ingress{
		Create: ingressList(mcreate),
		Update: update,
		Delete: ingressList(mdelete),
	}
}

func ingressMap(deps []v1beta1.Ingress) map[string]v1beta1.Ingress {
	m := map[string]v1beta1.Ingress{}
	for _, d := range deps {
		m[d.Name] = d
	}
	return m
}

func ingressList(m map[string]v1beta1.Ingress) []v1beta1.Ingress {
	l := []v1beta1.Ingress{}
	for _, v := range m {
		l = append(l, v)
	}
	return l
}
