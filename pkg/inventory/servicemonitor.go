package inventory

import (
	"fmt"

	monitoringv1 "github.com/coreos/prometheus-operator/pkg/apis/monitoring/v1"

	"github.com/jaegertracing/jaeger-operator/pkg/util"
)

// Service represents the inventory of routes based on the current and desired states
type ServiceMonitor struct {
	Create []*monitoringv1.ServiceMonitor
	Update []*monitoringv1.ServiceMonitor
	Delete []*monitoringv1.ServiceMonitor
}

// ForServices builds an inventory of services based on the existing and desired states
func ForServiceMonitors(existing []*monitoringv1.ServiceMonitor, desired []*monitoringv1.ServiceMonitor) ServiceMonitor {
	update := []*monitoringv1.ServiceMonitor{}
	mdelete := serviceMonitorMap(existing)
	mcreate := serviceMonitorMap(desired)

	for k, v := range mcreate {
		if t, ok := mdelete[k]; ok {
			tp := t.DeepCopy()
			util.InitObjectMeta(tp)

			// we can't blindly DeepCopyInto, so, we select what we bring from the new to the old object
			tp.Spec = v.Spec
			tp.ObjectMeta.OwnerReferences = v.ObjectMeta.OwnerReferences

			// there might be annotations not created by us, so, we need to just replace the ones we care about,
			// leaving all others there
			for k, v := range v.ObjectMeta.Annotations {
				tp.ObjectMeta.Annotations[k] = v
			}

			for k, v := range v.ObjectMeta.Labels {
				tp.ObjectMeta.Labels[k] = v
			}

			update = append(update, tp)
			delete(mcreate, k)
			delete(mdelete, k)
		}
	}

	return ServiceMonitor{
		Create: serviceMonitorList(mcreate),
		Update: update,
		Delete: serviceMonitorList(mdelete),
	}
}

func serviceMonitorMap(deps []*monitoringv1.ServiceMonitor) map[string]*monitoringv1.ServiceMonitor {
	m := map[string]*monitoringv1.ServiceMonitor{}
	for _, d := range deps {
		m[fmt.Sprintf("%s.%s", d.Namespace, d.Name)] = d
	}
	return m
}

func serviceMonitorList(m map[string]*monitoringv1.ServiceMonitor) []*monitoringv1.ServiceMonitor {
	l := []*monitoringv1.ServiceMonitor{}
	for _, v := range m {
		l = append(l, v)
	}
	return l
}
