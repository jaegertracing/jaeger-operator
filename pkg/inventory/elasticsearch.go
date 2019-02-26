package inventory

import (
	esv1alpha1 "github.com/jaegertracing/jaeger-operator/pkg/storage/elasticsearch/v1alpha1"
)

// Elasticsearch represents the elastic search inventory based on the current and desired states
type Elasticsearch struct {
	Create []esv1alpha1.Elasticsearch
	Update []esv1alpha1.Elasticsearch
	Delete []esv1alpha1.Elasticsearch
}

// ForElasticsearches builds a new elastic search inventory based on the existing and desired states
func ForElasticsearches(existing []esv1alpha1.Elasticsearch, desired []esv1alpha1.Elasticsearch) Elasticsearch {
	update := []esv1alpha1.Elasticsearch{}
	mcreate := esMap(desired)
	mdelete := esMap(existing)

	for k, v := range mcreate {
		if t, ok := mdelete[k]; ok {
			tp := t.DeepCopy()

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

	return Elasticsearch{
		Create: esList(mcreate),
		Update: update,
		Delete: esList(mdelete),
	}
}

func esMap(deps []esv1alpha1.Elasticsearch) map[string]esv1alpha1.Elasticsearch {
	m := map[string]esv1alpha1.Elasticsearch{}
	for _, d := range deps {
		m[d.Name] = d
	}
	return m
}

func esList(m map[string]esv1alpha1.Elasticsearch) []esv1alpha1.Elasticsearch {
	l := []esv1alpha1.Elasticsearch{}
	for _, v := range m {
		l = append(l, v)
	}
	return l
}
