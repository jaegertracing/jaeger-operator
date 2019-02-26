package inventory

import (
	osv1 "github.com/openshift/api/route/v1"
)

// Route represents the inventory of routes based on the current and desired states
type Route struct {
	Create []osv1.Route
	Update []osv1.Route
	Delete []osv1.Route
}

// ForRoutes builds an inventory of routes based on the existing and desired states
func ForRoutes(existing []osv1.Route, desired []osv1.Route) Route {
	update := []osv1.Route{}
	mcreate := routeMap(desired)
	mdelete := routeMap(existing)

	for k, v := range mcreate {
		if t, ok := mdelete[k]; ok {
			tp := t.DeepCopy()

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

	return Route{
		Create: routeList(mcreate),
		Update: update,
		Delete: routeList(mdelete),
	}
}

func routeMap(deps []osv1.Route) map[string]osv1.Route {
	m := map[string]osv1.Route{}
	for _, d := range deps {
		m[d.Name] = d
	}
	return m
}

func routeList(m map[string]osv1.Route) []osv1.Route {
	l := []osv1.Route{}
	for _, v := range m {
		l = append(l, v)
	}
	return l
}
