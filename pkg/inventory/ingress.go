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
