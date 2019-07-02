package inventory

import (
	"fmt"

	appsv1 "k8s.io/api/apps/v1"

	"github.com/jaegertracing/jaeger-operator/pkg/util"
)

// Deployment represents the deployment inventory based on the current and desired states
type Deployment struct {
	Create []appsv1.Deployment
	Update []appsv1.Deployment
	Delete []appsv1.Deployment
}

// ForDeployments builds a new Deployment inventory based on the existing and desired states
func ForDeployments(existing []appsv1.Deployment, desired []appsv1.Deployment) Deployment {
	update := []appsv1.Deployment{}
	mcreate := deploymentMap(desired)
	mdelete := deploymentMap(existing)

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

	return Deployment{
		Create: deploymentList(mcreate),
		Update: update,
		Delete: deploymentList(mdelete),
	}
}

func deploymentMap(deps []appsv1.Deployment) map[string]appsv1.Deployment {
	m := map[string]appsv1.Deployment{}
	for _, d := range deps {
		m[fmt.Sprintf("%s.%s", d.Namespace, d.Name)] = d
	}
	return m
}

func deploymentList(m map[string]appsv1.Deployment) []appsv1.Deployment {
	l := []appsv1.Deployment{}
	for _, v := range m {
		l = append(l, v)
	}
	return l
}
