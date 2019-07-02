package inventory

import (
	"fmt"

	v1 "k8s.io/api/core/v1"

	"github.com/jaegertracing/jaeger-operator/pkg/util"
)

// Service represents the inventory of routes based on the current and desired states
type Service struct {
	Create []v1.Service
	Update []v1.Service
	Delete []v1.Service
}

// ForServices builds an inventory of services based on the existing and desired states
func ForServices(existing []v1.Service, desired []v1.Service) Service {
	update := []v1.Service{}
	mdelete := serviceMap(existing)
	mcreate := serviceMap(desired)

	for k, v := range mcreate {
		if t, ok := mdelete[k]; ok {
			tp := t.DeepCopy()
			util.InitObjectMeta(tp)

			// we keep the ClusterIP that got assigned by the cluster, if it's empty in the "desired" and not empty on the "current"
			if v.Spec.ClusterIP == "" && len(tp.Spec.ClusterIP) > 0 {
				v.Spec.ClusterIP = tp.Spec.ClusterIP
			}

			// we can't blindly DeepCopyInto, so, we select what we bring from the new to the old object
			tp.Spec = v.Spec
			tp.ObjectMeta.OwnerReferences = v.ObjectMeta.OwnerReferences

			// there might be annotations not created by us, so, we need to just replace the ones we care about,
			// leaving all others there
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

	return Service{
		Create: serviceList(mcreate),
		Update: update,
		Delete: serviceList(mdelete),
	}
}

func serviceMap(deps []v1.Service) map[string]v1.Service {
	m := map[string]v1.Service{}
	for _, d := range deps {
		m[fmt.Sprintf("%s.%s", d.Namespace, d.Name)] = d
	}
	return m
}

func serviceList(m map[string]v1.Service) []v1.Service {
	l := []v1.Service{}
	for _, v := range m {
		l = append(l, v)
	}
	return l
}
