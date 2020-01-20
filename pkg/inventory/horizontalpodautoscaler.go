package inventory

import (
	"fmt"

	autoscalingv2beta2 "k8s.io/api/autoscaling/v2beta2"

	"github.com/jaegertracing/jaeger-operator/pkg/util"
)

// HorizontalPodAutoscaler represents the HorizontalPodAutoscaler inventory based on the current and desired states
type HorizontalPodAutoscaler struct {
	Create []autoscalingv2beta2.HorizontalPodAutoscaler
	Update []autoscalingv2beta2.HorizontalPodAutoscaler
	Delete []autoscalingv2beta2.HorizontalPodAutoscaler
}

// ForHorizontalPodAutoscalers builds a new HorizontalPodAutoscaler inventory based on the existing and desired states
func ForHorizontalPodAutoscalers(existing []autoscalingv2beta2.HorizontalPodAutoscaler, desired []autoscalingv2beta2.HorizontalPodAutoscaler) HorizontalPodAutoscaler {
	update := []autoscalingv2beta2.HorizontalPodAutoscaler{}
	mcreate := hpaMap(desired)
	mdelete := hpaMap(existing)

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

	return HorizontalPodAutoscaler{
		Create: hpaList(mcreate),
		Update: update,
		Delete: hpaList(mdelete),
	}
}

func hpaMap(hpas []autoscalingv2beta2.HorizontalPodAutoscaler) map[string]autoscalingv2beta2.HorizontalPodAutoscaler {
	m := map[string]autoscalingv2beta2.HorizontalPodAutoscaler{}
	for _, d := range hpas {
		m[fmt.Sprintf("%s.%s", d.Namespace, d.Name)] = d
	}
	return m
}

func hpaList(m map[string]autoscalingv2beta2.HorizontalPodAutoscaler) []autoscalingv2beta2.HorizontalPodAutoscaler {
	l := []autoscalingv2beta2.HorizontalPodAutoscaler{}
	for _, v := range m {
		l = append(l, v)
	}
	return l
}
