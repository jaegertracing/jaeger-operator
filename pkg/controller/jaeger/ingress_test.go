package jaeger

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	networkingv1 "k8s.io/api/networking/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	v1 "github.com/jaegertracing/jaeger-operator/apis/v1"
	"github.com/jaegertracing/jaeger-operator/pkg/strategy"
)

func TestIngressesCreate(t *testing.T) {
	// prepare
	nsn := types.NamespacedName{
		Name: "TestIngressesCreate",
	}

	objs := []runtime.Object{
		v1.NewJaeger(nsn),
	}

	req := reconcile.Request{
		NamespacedName: nsn,
	}

	r, cl := getReconciler(objs)
	r.strategyChooser = func(ctx context.Context, jaeger *v1.Jaeger) strategy.S {
		s := strategy.New().WithIngresses([]networkingv1.Ingress{{
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

	persisted := &networkingv1.Ingress{}
	persistedName := types.NamespacedName{
		Name:      nsn.Name,
		Namespace: nsn.Namespace,
	}
	err = cl.Get(context.Background(), persistedName, persisted)
	assert.Equal(t, persistedName.Name, persisted.Name)
	assert.NoError(t, err)
}

func TestIngressesUpdate(t *testing.T) {
	// prepare
	nsn := types.NamespacedName{
		Name: "TestIngressesUpdate",
	}

	orig := networkingv1.Ingress{}
	orig.Name = nsn.Name
	orig.Annotations = map[string]string{"key": "value"}
	orig.Labels = map[string]string{
		"app.kubernetes.io/instance":   nsn.Name,
		"app.kubernetes.io/managed-by": "jaeger-operator",
	}

	objs := []runtime.Object{
		v1.NewJaeger(nsn),
		&orig,
	}

	r, cl := getReconciler(objs)
	r.strategyChooser = func(ctx context.Context, jaeger *v1.Jaeger) strategy.S {
		updated := networkingv1.Ingress{}
		updated.Name = orig.Name
		updated.Annotations = map[string]string{"key": "new-value"}

		s := strategy.New().WithIngresses([]networkingv1.Ingress{updated})
		return s
	}

	// test
	_, err := r.Reconcile(reconcile.Request{NamespacedName: nsn})
	assert.NoError(t, err)

	// verify
	persisted := &networkingv1.Ingress{}
	persistedName := types.NamespacedName{
		Name:      orig.Name,
		Namespace: orig.Namespace,
	}
	err = cl.Get(context.Background(), persistedName, persisted)
	assert.Equal(t, "new-value", persisted.Annotations["key"])
	assert.NoError(t, err)
}

func TestIngressesDelete(t *testing.T) {
	// prepare
	nsn := types.NamespacedName{
		Name: "TestIngressesDelete",
	}

	orig := networkingv1.Ingress{}
	orig.Name = nsn.Name
	orig.Labels = map[string]string{
		"app.kubernetes.io/instance":   nsn.Name,
		"app.kubernetes.io/managed-by": "jaeger-operator",
	}

	objs := []runtime.Object{
		v1.NewJaeger(nsn),
		&orig,
	}

	r, cl := getReconciler(objs)
	r.strategyChooser = func(ctx context.Context, jaeger *v1.Jaeger) strategy.S {
		return strategy.S{}
	}

	// test
	_, err := r.Reconcile(reconcile.Request{NamespacedName: nsn})
	assert.NoError(t, err)

	// verify
	persisted := &networkingv1.Ingress{}
	persistedName := types.NamespacedName{
		Name:      orig.Name,
		Namespace: orig.Namespace,
	}
	err = cl.Get(context.Background(), persistedName, persisted)
	assert.Empty(t, persisted.Name)
	assert.Error(t, err) // not found
}

func TestIngressesCreateExistingNameInAnotherNamespace(t *testing.T) {
	// prepare
	nsn := types.NamespacedName{
		Name:      "TestIngressesCreateExistingNameInAnotherNamespace",
		Namespace: "tenant1",
	}
	nsnExisting := types.NamespacedName{
		Name:      "TestIngressesCreateExistingNameInAnotherNamespace",
		Namespace: "tenant2",
	}

	objs := []runtime.Object{
		v1.NewJaeger(nsn),
		v1.NewJaeger(nsnExisting),
		&networkingv1.Ingress{
			ObjectMeta: metav1.ObjectMeta{
				Name:      nsnExisting.Name,
				Namespace: nsnExisting.Namespace,
			},
		},
	}

	req := reconcile.Request{
		NamespacedName: nsn,
	}

	r, cl := getReconciler(objs)
	r.strategyChooser = func(ctx context.Context, jaeger *v1.Jaeger) strategy.S {
		s := strategy.New().WithIngresses([]networkingv1.Ingress{{
			ObjectMeta: metav1.ObjectMeta{
				Name:      nsn.Name,
				Namespace: nsn.Namespace,
			},
		}})
		return s
	}

	// test
	res, err := r.Reconcile(req)

	// verify
	assert.NoError(t, err)
	assert.False(t, res.Requeue, "We don't requeue for now")

	persisted := &networkingv1.Ingress{}
	err = cl.Get(context.Background(), nsn, persisted)
	assert.NoError(t, err)
	assert.Equal(t, nsn.Name, persisted.Name)
	assert.Equal(t, nsn.Namespace, persisted.Namespace)

	persistedExisting := &networkingv1.Ingress{}
	err = cl.Get(context.Background(), nsnExisting, persistedExisting)
	assert.NoError(t, err)
	assert.Equal(t, nsnExisting.Name, persistedExisting.Name)
	assert.Equal(t, nsnExisting.Namespace, persistedExisting.Namespace)
}
