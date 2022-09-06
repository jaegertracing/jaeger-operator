package inventory

import (
	"fmt"

	autoscalingv2 "k8s.io/api/autoscaling/v2"
	autoscalingv2beta2 "k8s.io/api/autoscaling/v2beta2"
	"k8s.io/apimachinery/pkg/runtime"

	"github.com/spf13/viper"

	v1 "github.com/jaegertracing/jaeger-operator/apis/v1"
	"github.com/jaegertracing/jaeger-operator/pkg/util"
)

// HorizontalPodAutoscaler represents the HorizontalPodAutoscaler inventory based on the current and desired states
type HorizontalPodAutoscaler struct {
	Create []runtime.Object
	Update []runtime.Object
	Delete []runtime.Object
}

// ForHorizontalPodAutoscalers builds a new HorizontalPodAutoscaler inventory based on the existing and desired states
func ForHorizontalPodAutoscalers(existing []runtime.Object, desired []runtime.Object) HorizontalPodAutoscaler {
	update := []runtime.Object{}
	mcreate := hpaMap(desired)
	mdelete := hpaMap(existing)

	autoscalingVersion := viper.GetString(v1.FlagAutoscalingVersion)

	for k, v := range mcreate {
		if t, ok := mdelete[k]; ok {
			if autoscalingVersion == v1.FlagAutoscalingVersionV2Beta2 {
				t1 := t.(*autoscalingv2beta2.HorizontalPodAutoscaler)
				v1 := v.(*autoscalingv2beta2.HorizontalPodAutoscaler)

				tp := t1.DeepCopy()
				util.InitObjectMeta(tp)

				// we can't blindly DeepCopyInto, so, we select what we bring from the new to the old object
				tp.Spec = v1.Spec
				tp.ObjectMeta.OwnerReferences = v1.ObjectMeta.OwnerReferences

				for k, v := range v1.ObjectMeta.Annotations {
					tp.ObjectMeta.Annotations[k] = v
				}

				for k, v := range v1.ObjectMeta.Labels {
					tp.ObjectMeta.Labels[k] = v
				}

				update = append(update, tp)
				delete(mcreate, k)
				delete(mdelete, k)

			} else {
				t1 := t.(*autoscalingv2.HorizontalPodAutoscaler)
				v1 := v.(*autoscalingv2.HorizontalPodAutoscaler)

				tp := t1.DeepCopy()
				util.InitObjectMeta(tp)

				// we can't blindly DeepCopyInto, so, we select what we bring from the new to the old object
				tp.Spec = v1.Spec
				tp.ObjectMeta.OwnerReferences = v1.ObjectMeta.OwnerReferences

				for k, v := range v1.ObjectMeta.Annotations {
					tp.ObjectMeta.Annotations[k] = v
				}

				for k, v := range v1.ObjectMeta.Labels {
					tp.ObjectMeta.Labels[k] = v
				}

				update = append(update, tp)
				delete(mcreate, k)
				delete(mdelete, k)
			}
		}
	}

	return HorizontalPodAutoscaler{
		Create: hpaList(mcreate),
		Update: update,
		Delete: hpaList(mdelete),
	}
}

func hpaMap(hpas []runtime.Object) map[string]runtime.Object {
	m := map[string]runtime.Object{}

	autoscalingVersion := viper.GetString(v1.FlagAutoscalingVersion)
	for _, d := range hpas {
		if autoscalingVersion == v1.FlagAutoscalingVersionV2Beta2 {
			hpa := d.(*autoscalingv2beta2.HorizontalPodAutoscaler)
			m[fmt.Sprintf("%s.%s", hpa.Namespace, hpa.Name)] = hpa
		} else {
			hpa := d.(*autoscalingv2.HorizontalPodAutoscaler)
			m[fmt.Sprintf("%s.%s", hpa.Namespace, hpa.Name)] = hpa
		}
	}
	return m
}

func hpaList(m map[string]runtime.Object) []runtime.Object {
	l := []runtime.Object{}
	for _, v := range m {
		l = append(l, v)
	}
	return l
}
