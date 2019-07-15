package inventory

import (
	"fmt"

	v1 "k8s.io/api/core/v1"

	"github.com/jaegertracing/jaeger-operator/pkg/util"
)

// Account represents the service account inventory based on the current and desired states
type Account struct {
	Create []v1.ServiceAccount
	Update []v1.ServiceAccount
	Delete []v1.ServiceAccount
}

// ForAccounts builds a new Account inventory based on the existing and desired states
func ForAccounts(existing []v1.ServiceAccount, desired []v1.ServiceAccount) Account {
	update := []v1.ServiceAccount{}
	mcreate := accountMap(desired)
	mdelete := accountMap(existing)

	for k, v := range mcreate {
		if t, ok := mdelete[k]; ok {
			tp := t.DeepCopy()
			util.InitObjectMeta(tp)

			// we can't blindly DeepCopyInto, so, we select what we bring from the new to the old object
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

	return Account{
		Create: accountList(mcreate),
		Update: update,
		Delete: accountList(mdelete),
	}
}

func accountMap(deps []v1.ServiceAccount) map[string]v1.ServiceAccount {
	m := map[string]v1.ServiceAccount{}
	for _, d := range deps {
		m[fmt.Sprintf("%s.%s", d.Namespace, d.Name)] = d
	}
	return m
}

func accountList(m map[string]v1.ServiceAccount) []v1.ServiceAccount {
	l := []v1.ServiceAccount{}
	for _, v := range m {
		l = append(l, v)
	}
	return l
}
