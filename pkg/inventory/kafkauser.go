package inventory

import (
	"fmt"

	"github.com/jaegertracing/jaeger-operator/pkg/apis/kafka/v1beta2"
	"github.com/jaegertracing/jaeger-operator/pkg/util"
)

// KafkaUser represents the inventory of kafkas based on the current and desired states
type KafkaUser struct {
	Create []v1beta2.KafkaUser
	Update []v1beta2.KafkaUser
	Delete []v1beta2.KafkaUser
}

// ForKafkaUsers builds an inventory of kafkas based on the existing and desired states
func ForKafkaUsers(existing []v1beta2.KafkaUser, desired []v1beta2.KafkaUser) KafkaUser {
	update := []v1beta2.KafkaUser{}
	mcreate := kafkaUserMap(desired)
	mdelete := kafkaUserMap(existing)

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

	return KafkaUser{
		Create: kafkaUserList(mcreate),
		Update: update,
		Delete: kafkaUserList(mdelete),
	}
}

func kafkaUserMap(deps []v1beta2.KafkaUser) map[string]v1beta2.KafkaUser {
	m := map[string]v1beta2.KafkaUser{}
	for _, d := range deps {
		m[fmt.Sprintf("%s.%s", d.Namespace, d.Name)] = d
	}
	return m
}

func kafkaUserList(m map[string]v1beta2.KafkaUser) []v1beta2.KafkaUser {
	l := []v1beta2.KafkaUser{}
	for _, v := range m {
		l = append(l, v)
	}
	return l
}
