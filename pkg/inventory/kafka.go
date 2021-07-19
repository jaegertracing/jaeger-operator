package inventory

import (
	"fmt"

	log "github.com/sirupsen/logrus"

	"github.com/jaegertracing/jaeger-operator/pkg/apis/kafka/v1beta2"
	"github.com/jaegertracing/jaeger-operator/pkg/util"
)

// Kafka represents the inventory of kafkas based on the current and desired states
type Kafka struct {
	Create []v1beta2.Kafka
	Update []v1beta2.Kafka
	Delete []v1beta2.Kafka
}

// ForKafkas builds an inventory of kafkas based on the existing and desired states
func ForKafkas(existing []v1beta2.Kafka, desired []v1beta2.Kafka) Kafka {
	update := []v1beta2.Kafka{}
	mcreate := kafkaMap(desired)
	mdelete := kafkaMap(existing)

	for _, k := range existing {
		log.WithFields(log.Fields{
			"kafka":     k.GetName(),
			"namespace": k.GetNamespace(),
		}).Debug("existing")
	}

	for _, k := range desired {
		log.WithFields(log.Fields{
			"kafka":     k.GetName(),
			"namespace": k.GetNamespace(),
		}).Debug("desired")
	}

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

	return Kafka{
		Create: kafkaList(mcreate),
		Update: update,
		Delete: kafkaList(mdelete),
	}
}

func kafkaMap(deps []v1beta2.Kafka) map[string]v1beta2.Kafka {
	m := map[string]v1beta2.Kafka{}
	for _, d := range deps {
		m[fmt.Sprintf("%s.%s", d.Namespace, d.Name)] = d
	}
	return m
}

func kafkaList(m map[string]v1beta2.Kafka) []v1beta2.Kafka {
	l := []v1beta2.Kafka{}
	for _, v := range m {
		l = append(l, v)
	}
	return l
}
