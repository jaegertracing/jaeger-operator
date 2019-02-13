package jaeger

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	appsv1 "k8s.io/api/apps/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes/scheme"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	"github.com/jaegertracing/jaeger-operator/pkg/apis/io/v1alpha1"
	"github.com/jaegertracing/jaeger-operator/pkg/strategy"
)

func TestDaemonSetsCreate(t *testing.T) {
	// prepare
	nsn := types.NamespacedName{
		Name: "TestDaemonSetsCreate",
	}

	objs := []runtime.Object{
		v1alpha1.NewJaeger(nsn.Name),
	}

	s := scheme.Scheme
	s.AddKnownTypes(v1alpha1.SchemeGroupVersion, &v1alpha1.Jaeger{})
	cl := fake.NewFakeClient(objs...)
	r := &ReconcileJaeger{client: cl, scheme: s}

	req := reconcile.Request{
		NamespacedName: nsn,
	}

	r.strategyChooser = func(jaeger *v1alpha1.Jaeger) strategy.S {
		s := strategy.New().WithDaemonSets([]appsv1.DaemonSet{{
			ObjectMeta: metav1.ObjectMeta{
				Name: nsn.Name,
			},
		}})
		return s
	}

	// test
	res, err := r.Reconcile(req)

	// verify
	assert.NoError(t, err)
	assert.False(t, res.Requeue, "We don't requeue for now")

	persisted := &appsv1.DaemonSet{}
	persistedName := types.NamespacedName{
		Name:      nsn.Name,
		Namespace: nsn.Namespace,
	}
	err = cl.Get(context.Background(), persistedName, persisted)
	assert.Equal(t, persistedName.Name, persisted.Name)
	assert.NoError(t, err)
}

func TestDaemonSetsUpdate(t *testing.T) {
	// prepare
	nsn := types.NamespacedName{
		Name: "TestDaemonSetsUpdate",
	}

	orig := appsv1.DaemonSet{}
	orig.Name = nsn.Name
	orig.Annotations = map[string]string{"key": "value"}

	objs := []runtime.Object{
		v1alpha1.NewJaeger(nsn.Name),
		&orig,
	}

	s := scheme.Scheme
	s.AddKnownTypes(v1alpha1.SchemeGroupVersion, &v1alpha1.Jaeger{})
	cl := fake.NewFakeClient(objs...)
	r := &ReconcileJaeger{client: cl, scheme: s}

	r.strategyChooser = func(jaeger *v1alpha1.Jaeger) strategy.S {
		updated := appsv1.DaemonSet{}
		updated.Name = orig.Name
		updated.Annotations = map[string]string{"key": "new-value"}

		s := strategy.New().WithDaemonSets([]appsv1.DaemonSet{updated})
		return s
	}

	// test
	_, err := r.Reconcile(reconcile.Request{NamespacedName: nsn})
	assert.NoError(t, err)

	// verify
	persisted := &appsv1.DaemonSet{}
	persistedName := types.NamespacedName{
		Name:      orig.Name,
		Namespace: orig.Namespace,
	}
	err = cl.Get(context.Background(), persistedName, persisted)
	assert.Equal(t, "new-value", persisted.Annotations["key"])
	assert.NoError(t, err)
}

func TestDaemonSetsDelete(t *testing.T) {
	// prepare
	nsn := types.NamespacedName{
		Name: "TestDaemonSetsDelete",
	}

	orig := appsv1.DaemonSet{}
	orig.Name = nsn.Name

	objs := []runtime.Object{
		v1alpha1.NewJaeger(nsn.Name),
		&orig,
	}

	s := scheme.Scheme
	s.AddKnownTypes(v1alpha1.SchemeGroupVersion, &v1alpha1.Jaeger{})
	cl := fake.NewFakeClient(objs...)
	r := &ReconcileJaeger{client: cl, scheme: s}

	r.strategyChooser = func(jaeger *v1alpha1.Jaeger) strategy.S {
		return strategy.S{}
	}

	// test
	_, err := r.Reconcile(reconcile.Request{NamespacedName: nsn})
	assert.NoError(t, err)

	// verify
	persisted := &appsv1.DaemonSet{}
	persistedName := types.NamespacedName{
		Name:      orig.Name,
		Namespace: orig.Namespace,
	}
	err = cl.Get(context.Background(), persistedName, persisted)
	assert.Empty(t, persisted.Name)
	assert.Error(t, err) // not found
}
