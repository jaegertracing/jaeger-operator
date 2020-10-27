package jaeger

import (
	"context"
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	v1 "github.com/jaegertracing/jaeger-operator/pkg/apis/jaegertracing/v1"
	"github.com/jaegertracing/jaeger-operator/pkg/strategy"

	monitoringv1 "github.com/coreos/prometheus-operator/pkg/apis/monitoring/v1"
)

func TestServiceMonitorsCreate(t *testing.T) {
	nsn := types.NamespacedName{
		Name: "TestServiceMonitorsCreate",
	}
	jaeger := v1.NewJaeger(nsn)
	trueVal := true
	jaeger.Spec.ServiceMonitor.Enabled = &trueVal
	objs := []runtime.Object{
		jaeger,
	}

	req := reconcile.Request{
		NamespacedName: nsn,
	}

	r, cl := getReconciler(objs)
	r.strategyChooser = func(ctx context.Context, jaeger *v1.Jaeger) strategy.S {
		s := strategy.New().WithServiceMonitors([]*monitoringv1.ServiceMonitor{{
			ObjectMeta: metav1.ObjectMeta{
				Name: nsn.Name,
			},
		}})
		return s
	}

	res, err := r.Reconcile(req)

	assert.NoError(t, err)
	assert.False(t, res.Requeue, "We don't requeue for now")

	persisted := &monitoringv1.ServiceMonitor{}
	persistedName := types.NamespacedName{
		Name:      nsn.Name,
		Namespace: nsn.Namespace,
	}
	err = cl.Get(context.Background(), persistedName, persisted)
	assert.Equal(t, persistedName.Name, persisted.Name)
	assert.NoError(t, err)
}

func TestServiceMonitorsUpdate(t *testing.T) {
	nsn := types.NamespacedName{
		Name: "TestServiceMonitorsUpdate",
	}

	orig := monitoringv1.ServiceMonitor{}
	orig.Name = nsn.Name
	orig.Spec.Endpoints = []monitoringv1.Endpoint{{
		Port: "80",
		Path: "/metrics",
	}}
	orig.Labels = map[string]string{
		"app.kubernetes.io/instance":   orig.Name,
		"app.kubernetes.io/managed-by": "jaeger-operator",
	}
	jaeger := v1.NewJaeger(nsn)
	trueVal := true
	jaeger.Spec.ServiceMonitor.Enabled = &trueVal
	objs := []runtime.Object{
		jaeger,
		&orig,
	}

	req := reconcile.Request{
		NamespacedName: nsn,
	}

	r, cl := getReconciler(objs)
	new := &monitoringv1.ServiceMonitor{
		ObjectMeta: metav1.ObjectMeta{
			Name: nsn.Name,
		},
		Spec: monitoringv1.ServiceMonitorSpec{
			Endpoints: []monitoringv1.Endpoint{{
				Port: "8080",
				Path: "/metrics",
			}},
		},
	}
	r.strategyChooser = func(ctx context.Context, jaeger *v1.Jaeger) strategy.S {
		return strategy.New().WithServiceMonitors([]*monitoringv1.ServiceMonitor{
			new,
		})
	}

	_, err := r.Reconcile(req)

	persisted := &monitoringv1.ServiceMonitor{}
	persistedName := types.NamespacedName{
		Name:      nsn.Name,
		Namespace: nsn.Namespace,
	}
	err = cl.Get(context.Background(), persistedName, persisted)
	assert.NoError(t, err)
	assert.Equal(t, new.Spec, persisted.Spec)
}

func TestServiceMonitorsDelete(t *testing.T) {
	nsn := types.NamespacedName{
		Name: "TestServiceMonitorsDelete",
	}

	orig := monitoringv1.ServiceMonitor{}
	orig.Name = nsn.Name
	orig.Labels = map[string]string{
		"app.kubernetes.io/instance":   orig.Name,
		"app.kubernetes.io/managed-by": "jaeger-operator",
	}
	jaeger := v1.NewJaeger(nsn)
	trueVal := true
	jaeger.Spec.ServiceMonitor.Enabled = &trueVal
	objs := []runtime.Object{
		jaeger,
		&orig,
	}

	req := reconcile.Request{
		NamespacedName: nsn,
	}

	r, cl := getReconciler(objs)
	r.strategyChooser = func(ctx context.Context, jaeger *v1.Jaeger) strategy.S {
		return strategy.S{}
	}

	_, err := r.Reconcile(req)

	persisted := &monitoringv1.ServiceMonitor{}
	persistedName := types.NamespacedName{
		Name:      nsn.Name,
		Namespace: nsn.Namespace,
	}
	err = cl.Get(context.Background(), persistedName, persisted)
	assert.EqualError(t, err, fmt.Sprintf("servicemonitors.monitoring.coreos.com \"%s\" not found", nsn.Name))
}
