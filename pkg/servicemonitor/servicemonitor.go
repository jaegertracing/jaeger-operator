package servicemonitor

import (
	"fmt"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	monitoringv1 "github.com/coreos/prometheus-operator/pkg/apis/monitoring/v1"

	v1 "github.com/jaegertracing/jaeger-operator/pkg/apis/jaegertracing/v1"
	"github.com/jaegertracing/jaeger-operator/pkg/util"
)

// NewServiceMonitor returns a new ServiceMonitor object used by prometheus-operator
func NewServiceMonitor(jaeger *v1.Jaeger) *monitoringv1.ServiceMonitor {
	trueVar := true

	labels := util.Labels(jaeger.Name, "metrics", *jaeger)
	// We want to select all services matching a single Jaeger instance
	matchLabels := labels
	delete(matchLabels, "app.kubernetes.io/name")
	delete(matchLabels, "app.kubernetes.io/component")

	return &monitoringv1.ServiceMonitor{
		ObjectMeta: metav1.ObjectMeta{
			Name:      fmt.Sprintf("%s-metrics", jaeger.Name),
			Namespace: jaeger.Namespace,
			Labels:    labels,
			OwnerReferences: []metav1.OwnerReference{{
				APIVersion: jaeger.APIVersion,
				Kind:       jaeger.Kind,
				Name:       jaeger.Name,
				UID:        jaeger.UID,
				Controller: &trueVar,
			}},
		},
		Spec: monitoringv1.ServiceMonitorSpec{
			Endpoints: []monitoringv1.Endpoint{{
				Port: "admin",
				Path: "/metrics",
			}},
			Selector: metav1.LabelSelector{
				MatchLabels: matchLabels,
			},
		},
	}
}
