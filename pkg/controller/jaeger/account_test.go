package jaeger

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	v1 "github.com/jaegertracing/jaeger-operator/pkg/apis/jaegertracing/v1"
	"github.com/jaegertracing/jaeger-operator/pkg/strategy"
)

func TestServiceAccountCreate(t *testing.T) {
	// prepare
	nsn := types.NamespacedName{
		Name: "TestServiceAccountCreate",
	}

	objs := []runtime.Object{
		v1.NewJaeger(nsn),
	}

	r, cl := getReconciler(objs)
	req := reconcile.Request{
		NamespacedName: nsn,
	}

	r.strategyChooser = func(jaeger *v1.Jaeger) strategy.S {
		s := strategy.New().WithAccounts([]corev1.ServiceAccount{{
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

	persisted := &corev1.ServiceAccount{}
	persistedName := types.NamespacedName{
		Name:      nsn.Name,
		Namespace: nsn.Namespace,
	}
	err = cl.Get(context.Background(), persistedName, persisted)
	assert.Equal(t, persistedName.Name, persisted.Name)
	assert.NoError(t, err)
}

func TestServiceAccountUpdate(t *testing.T) {
	// prepare
	nsn := types.NamespacedName{
		Name: "TestServiceAccountUpdate",
	}

	orig := corev1.ServiceAccount{}
	orig.Name = nsn.Name
	orig.Annotations = map[string]string{"key": "value"}

	objs := []runtime.Object{
		v1.NewJaeger(nsn),
		&orig,
	}

	r, cl := getReconciler(objs)
	r.strategyChooser = func(jaeger *v1.Jaeger) strategy.S {
		updated := corev1.ServiceAccount{}
		updated.Name = orig.Name
		updated.Annotations = map[string]string{"key": "new-value"}

		s := strategy.New().WithAccounts([]corev1.ServiceAccount{updated})
		return s
	}

	// test
	_, err := r.Reconcile(reconcile.Request{NamespacedName: nsn})
	assert.NoError(t, err)

	// verify
	persisted := &corev1.ServiceAccount{}
	persistedName := types.NamespacedName{
		Name:      orig.Name,
		Namespace: orig.Namespace,
	}
	err = cl.Get(context.Background(), persistedName, persisted)
	assert.Equal(t, "new-value", persisted.Annotations["key"])
	assert.NoError(t, err)
}

func TestServiceAccountDelete(t *testing.T) {
	// prepare
	nsn := types.NamespacedName{
		Name: "TestServiceAccountDelete",
	}

	orig := corev1.ServiceAccount{}
	orig.Name = nsn.Name

	objs := []runtime.Object{
		v1.NewJaeger(nsn),
		&orig,
	}

	r, cl := getReconciler(objs)
	r.strategyChooser = func(jaeger *v1.Jaeger) strategy.S {
		return strategy.S{}
	}

	// test
	_, err := r.Reconcile(reconcile.Request{NamespacedName: nsn})
	assert.NoError(t, err)

	// verify
	persisted := &corev1.ServiceAccount{}
	persistedName := types.NamespacedName{
		Name:      orig.Name,
		Namespace: orig.Namespace,
	}
	err = cl.Get(context.Background(), persistedName, persisted)
	assert.Empty(t, persisted.Name)
	assert.Error(t, err) // not found
}

func TestAccountCreateExistingNameInAnotherNamespace(t *testing.T) {
	// prepare
	nsn := types.NamespacedName{
		Name:      "TestAccountCreateExistingNameInAnotherNamespace",
		Namespace: "tenant1",
	}
	nsnExisting := types.NamespacedName{
		Name:      "TestAccountCreateExistingNameInAnotherNamespace",
		Namespace: "tenant2",
	}

	objs := []runtime.Object{
		v1.NewJaeger(nsn),
		v1.NewJaeger(nsnExisting),
		&corev1.ServiceAccount{
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
	r.strategyChooser = func(jaeger *v1.Jaeger) strategy.S {
		s := strategy.New().WithAccounts([]corev1.ServiceAccount{{
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

	persisted := &corev1.ServiceAccount{}
	err = cl.Get(context.Background(), nsn, persisted)
	assert.NoError(t, err)
	assert.Equal(t, nsn.Name, persisted.Name)
	assert.Equal(t, nsn.Namespace, persisted.Namespace)

	persistedExisting := &corev1.ServiceAccount{}
	err = cl.Get(context.Background(), nsnExisting, persistedExisting)
	assert.NoError(t, err)
	assert.Equal(t, nsnExisting.Name, persistedExisting.Name)
	assert.Equal(t, nsnExisting.Namespace, persistedExisting.Namespace)
}
