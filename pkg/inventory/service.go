package inventory

import (
	"k8s.io/api/core/v1"
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
	mcreate := serviceMap(desired)
	mdelete := serviceMap(existing)

	for k, v := range mcreate {
		if _, ok := mdelete[k]; ok {
			update = append(update, v)
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
		m[d.Name] = d
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
