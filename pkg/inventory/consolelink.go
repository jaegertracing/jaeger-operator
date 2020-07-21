package inventory

import (
	osconsolev1 "github.com/openshift/api/console/v1"

	"github.com/jaegertracing/jaeger-operator/pkg/util"
)

// ConsoleLink represents the inventory of console links based on the current and desired states
type ConsoleLink struct {
	Create []osconsolev1.ConsoleLink
	Update []osconsolev1.ConsoleLink
	Delete []osconsolev1.ConsoleLink
}

// ForConsoleLinks builds an inventory of console links based on the existing and desired states
func ForConsoleLinks(existing []osconsolev1.ConsoleLink, desired []osconsolev1.ConsoleLink) ConsoleLink {
	update := []osconsolev1.ConsoleLink{}
	mcreate := consoleLinkMap(desired)
	mdelete := consoleLinkMap(existing)

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

	return ConsoleLink{
		Create: consoleLinkList(mcreate),
		Update: update,
		Delete: consoleLinkList(mdelete),
	}
}

func consoleLinkMap(deps []osconsolev1.ConsoleLink) map[string]osconsolev1.ConsoleLink {
	m := map[string]osconsolev1.ConsoleLink{}
	for _, d := range deps {
		m[d.Name] = d
	}
	return m
}

func consoleLinkList(m map[string]osconsolev1.ConsoleLink) []osconsolev1.ConsoleLink {
	l := []osconsolev1.ConsoleLink{}
	for _, v := range m {
		l = append(l, v)
	}
	return l
}
