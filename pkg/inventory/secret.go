package inventory

import (
	"fmt"

	v1 "k8s.io/api/core/v1"

	"github.com/jaegertracing/jaeger-operator/pkg/util"
)

// Secret represents the secrets inventory based on the current and desired states
type Secret struct {
	Create []v1.Secret
	Update []v1.Secret
	Delete []v1.Secret
}

// ForSecrets builds a new secret inventory based on the existing and desired states
func ForSecrets(existing []v1.Secret, desired []v1.Secret) Secret {
	update := []v1.Secret{}
	mcreate := secretsMap(desired)
	mdelete := secretsMap(existing)

	for k, v := range mcreate {
		if t, ok := mdelete[k]; ok {
			tp := t.DeepCopy()
			util.InitObjectMeta(tp)

			tp.Data = v.Data
			tp.StringData = v.StringData
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

	return Secret{
		Create: secretsList(mcreate),
		Update: update,
		Delete: secretsList(mdelete),
	}
}

func secretsMap(deps []v1.Secret) map[string]v1.Secret {
	m := map[string]v1.Secret{}
	for _, d := range deps {
		m[fmt.Sprintf("%s.%s", d.Namespace, d.Name)] = d
	}
	return m
}

func secretsList(m map[string]v1.Secret) []v1.Secret {
	l := []v1.Secret{}
	for _, v := range m {
		l = append(l, v)
	}
	return l
}
