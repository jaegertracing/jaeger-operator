package inventory

import (
	"fmt"

	networkingv1 "k8s.io/api/networking/v1"

	"github.com/jaegertracing/jaeger-operator/pkg/util"
)

// Ingress represents the inventory of ingresses based on the current and desired states
type Ingress struct {
	Create []networkingv1.Ingress
	Update []networkingv1.Ingress
	Delete []networkingv1.Ingress
}

// ForIngresses builds an inventory of ingresses based on the existing and desired states
func ForIngresses(existing []networkingv1.Ingress, desired []networkingv1.Ingress) Ingress {
	update := []networkingv1.Ingress{}
	mcreate := ingressMap(desired)
	mdelete := ingressMap(existing)

	for k, v := range mcreate {
		if t, ok := mdelete[k]; ok {
			tp := t.DeepCopy()
			util.InitObjectMeta(tp)

			// we can't blindly DeepCopyInto, so, we select what we bring from the new to the old object
			tp.Spec = v.Spec
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

	return Ingress{
		Create: ingressList(mcreate),
		Update: update,
		Delete: ingressList(mdelete),
	}
}

func ingressMap(deps []networkingv1.Ingress) map[string]networkingv1.Ingress {
	m := map[string]networkingv1.Ingress{}
	for _, d := range deps {
		m[fmt.Sprintf("%s.%s", d.Namespace, d.Name)] = d
	}
	return m
}

func ingressList(m map[string]networkingv1.Ingress) []networkingv1.Ingress {
	l := []networkingv1.Ingress{}
	for _, v := range m {
		l = append(l, v)
	}
	return l
}
