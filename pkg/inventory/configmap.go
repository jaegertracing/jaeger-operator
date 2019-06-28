package inventory

import (
	"fmt"

	v1 "k8s.io/api/core/v1"

	"github.com/jaegertracing/jaeger-operator/pkg/util"
)

// ConfigMap represents the config maps inventory based on the current and desired states
type ConfigMap struct {
	Create []v1.ConfigMap
	Update []v1.ConfigMap
	Delete []v1.ConfigMap
}

// ForConfigMaps builds a new Account inventory based on the existing and desired states
func ForConfigMaps(existing []v1.ConfigMap, desired []v1.ConfigMap) ConfigMap {
	update := []v1.ConfigMap{}
	mcreate := configsMap(desired)
	mdelete := configsMap(existing)

	for k, v := range mcreate {
		if t, ok := mdelete[k]; ok {
			tp := t.DeepCopy()
			util.InitObjectMeta(tp)

			// we can't blindly DeepCopyInto, so, we select what we bring from the new to the old object
			tp.Data = v.Data
			tp.BinaryData = v.BinaryData
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

	return ConfigMap{
		Create: configsList(mcreate),
		Update: update,
		Delete: configsList(mdelete),
	}
}

func configsMap(deps []v1.ConfigMap) map[string]v1.ConfigMap {
	m := map[string]v1.ConfigMap{}
	for _, d := range deps {
		m[fmt.Sprintf("%s.%s", d.Namespace, d.Name)] = d
	}
	return m
}

func configsList(m map[string]v1.ConfigMap) []v1.ConfigMap {
	l := []v1.ConfigMap{}
	for _, v := range m {
		l = append(l, v)
	}
	return l
}
