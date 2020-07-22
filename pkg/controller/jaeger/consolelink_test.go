package jaeger

import (
	"context"
	"testing"

	osconsolev1 "github.com/openshift/api/console/v1"
	"github.com/spf13/viper"
	"github.com/stretchr/testify/assert"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	v1 "github.com/jaegertracing/jaeger-operator/pkg/apis/jaegertracing/v1"
	"github.com/jaegertracing/jaeger-operator/pkg/strategy"
)

func TestConsoleLinkCreate(t *testing.T) {
	// prepare
	nsn := types.NamespacedName{
		Name: "my-instance",
	}
	viper.Set("platform", "openshift")
	defer viper.Reset()

	objs := []runtime.Object{
		v1.NewJaeger(nsn),
	}

	req := reconcile.Request{
		NamespacedName: nsn,
	}

	r, cl := getReconciler(objs)
	r.strategyChooser = func(ctx context.Context, jaeger *v1.Jaeger) strategy.S {
		s := strategy.New().WithConsoleLinks([]osconsolev1.ConsoleLink{{
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

	persisted := &osconsolev1.ConsoleLink{}
	persistedName := types.NamespacedName{
		Name:      nsn.Name,
		Namespace: nsn.Namespace,
	}
	err = cl.Get(context.Background(), persistedName, persisted)
	assert.Equal(t, persistedName.Name, persisted.Name)
	assert.NoError(t, err)
}

func TestConsoleLinkUpdate(t *testing.T) {
	// prepare
	nsn := types.NamespacedName{
		Name: "my-instance",
	}
	viper.Set("platform", "openshift")
	defer viper.Reset()

	orig := osconsolev1.ConsoleLink{}
	orig.Name = nsn.Name
	orig.Annotations = map[string]string{"key": "value"}
	orig.Labels = map[string]string{
		"app.kubernetes.io/instance":   orig.Name,
		"app.kubernetes.io/managed-by": "jaeger-operator",
	}

	objs := []runtime.Object{
		v1.NewJaeger(nsn),
		&orig,
	}

	r, cl := getReconciler(objs)
	r.strategyChooser = func(ctx context.Context, jaeger *v1.Jaeger) strategy.S {
		updated := osconsolev1.ConsoleLink{}
		updated.Name = orig.Name
		updated.Annotations = map[string]string{"key": "new-value"}

		s := strategy.New().WithConsoleLinks([]osconsolev1.ConsoleLink{updated})
		return s
	}

	// test
	_, err := r.Reconcile(reconcile.Request{NamespacedName: nsn})
	assert.NoError(t, err)

	// verify
	persisted := &osconsolev1.ConsoleLink{}
	persistedName := types.NamespacedName{
		Name:      orig.Name,
		Namespace: orig.Namespace,
	}
	err = cl.Get(context.Background(), persistedName, persisted)
	assert.Equal(t, "new-value", persisted.Annotations["key"])
	assert.NoError(t, err)
}

func TestConsoleLinkDelete(t *testing.T) {
	// prepare
	nsn := types.NamespacedName{
		Name: "my-instance",
	}
	viper.Set("platform", "openshift")
	defer viper.Reset()

	orig := osconsolev1.ConsoleLink{}
	orig.Name = nsn.Name
	orig.Labels = map[string]string{
		"app.kubernetes.io/instance":   orig.Name,
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
	persisted := &osconsolev1.ConsoleLink{}
	persistedName := types.NamespacedName{
		Name:      orig.Name,
		Namespace: orig.Namespace,
	}
	err = cl.Get(context.Background(), persistedName, persisted)
	assert.Empty(t, persisted.Name)
	assert.Error(t, err) // not found
}

func TestConsoleLinksCreateExistingNameInAnotherNamespace(t *testing.T) {
	// prepare
	nsn := types.NamespacedName{
		Name:      "my-instance-1",
		Namespace: "tenant1",
	}
	nsnExisting := types.NamespacedName{
		Name:      "my-instance-2",
		Namespace: "tenant2",
	}
	viper.Set("platform", "openshift")
	defer viper.Reset()

	objs := []runtime.Object{
		v1.NewJaeger(nsn),
		v1.NewJaeger(nsnExisting),
		&osconsolev1.ConsoleLink{
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
		s := strategy.New().WithConsoleLinks([]osconsolev1.ConsoleLink{{
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

	persisted := &osconsolev1.ConsoleLink{}
	err = cl.Get(context.Background(), nsn, persisted)
	assert.NoError(t, err)
	assert.Equal(t, nsn.Name, persisted.Name)
	assert.Equal(t, nsn.Namespace, persisted.Namespace)

	persistedExisting := &osconsolev1.ConsoleLink{}
	err = cl.Get(context.Background(), nsnExisting, persistedExisting)
	assert.NoError(t, err)
	assert.Equal(t, nsnExisting.Name, persistedExisting.Name)
	assert.Equal(t, nsnExisting.Namespace, persistedExisting.Namespace)
}
