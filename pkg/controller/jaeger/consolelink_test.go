package jaeger

import (
	"context"
	"testing"

	"k8s.io/apimachinery/pkg/api/errors"

	v1 "github.com/jaegertracing/jaeger-operator/apis/v1"
	"github.com/jaegertracing/jaeger-operator/pkg/consolelink"

	osconsolev1 "github.com/openshift/api/console/v1"
	osroutev1 "github.com/openshift/api/route/v1"

	"github.com/spf13/viper"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	"github.com/jaegertracing/jaeger-operator/pkg/strategy"
)

func TestConsoleLinkCreate(t *testing.T) {
	// prepare
	nsn := types.NamespacedName{
		Name: "my-instance",
	}
	viper.Set("platform", "openshift")
	viper.Set(v1.ConfigOperatorScope, v1.OperatorScopeCluster)
	defer viper.Reset()

	objs := []client.Object{
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
				Annotations: map[string]string{
					consolelink.RouteAnnotation: "my-route",
				},
			},
		}}).WithRoutes([]osroutev1.Route{{
			ObjectMeta: metav1.ObjectMeta{
				Name: "my-route",
			},
			Spec: osroutev1.RouteSpec{
				Host: "myhost",
			},
		}})
		return s
	}

	// test
	res, err := r.Reconcile(req)

	// verify
	require.NoError(t, err)
	assert.False(t, res.Requeue, "We don't requeue for now")

	persisted := &osconsolev1.ConsoleLink{}
	persistedName := types.NamespacedName{
		Name:      nsn.Name,
		Namespace: nsn.Namespace,
	}
	err = cl.Get(context.Background(), persistedName, persisted)
	assert.Equal(t, persistedName.Name, persisted.Name)
	assert.Equal(t, "https://myhost", persisted.Spec.Href)

	require.NoError(t, err)
}

func TestConsoleLinkUpdate(t *testing.T) {
	// prepare
	nsn := types.NamespacedName{
		Name: "my-instance",
	}
	viper.Set("platform", "openshift")
	viper.Set(v1.ConfigOperatorScope, v1.OperatorScopeCluster)
	defer viper.Reset()

	orig := osconsolev1.ConsoleLink{}
	orig.Name = nsn.Name
	orig.Annotations = map[string]string{"key": "value"}
	orig.Labels = map[string]string{
		"app.kubernetes.io/instance":   orig.Name,
		"app.kubernetes.io/namespace":  orig.Namespace,
		"app.kubernetes.io/managed-by": "jaeger-operator",
	}

	objs := []client.Object{
		v1.NewJaeger(nsn),
		&orig,
	}

	r, cl := getReconciler(objs)
	r.strategyChooser = func(ctx context.Context, jaeger *v1.Jaeger) strategy.S {
		updated := osconsolev1.ConsoleLink{}
		updated.Name = orig.Name
		updated.Annotations = map[string]string{
			"key":                       "new-value",
			consolelink.RouteAnnotation: "my-route",
		}

		s := strategy.New().WithConsoleLinks([]osconsolev1.ConsoleLink{updated}).WithRoutes([]osroutev1.Route{{
			ObjectMeta: metav1.ObjectMeta{
				Name: "my-route",
			},
			Spec: osroutev1.RouteSpec{
				Host: "myhost",
			},
		}})
		return s
	}

	// test
	_, err := r.Reconcile(reconcile.Request{NamespacedName: nsn})
	require.NoError(t, err)

	// verify
	persisted := &osconsolev1.ConsoleLink{}
	persistedName := types.NamespacedName{
		Name:      orig.Name,
		Namespace: orig.Namespace,
	}
	err = cl.Get(context.Background(), persistedName, persisted)
	assert.Equal(t, "new-value", persisted.Annotations["key"])
	require.NoError(t, err)
}

func TestConsoleLinkDelete(t *testing.T) {
	// prepare
	nsn := types.NamespacedName{
		Name: "my-instance",
	}
	viper.Set("platform", "openshift")
	viper.Set(v1.ConfigOperatorScope, v1.OperatorScopeCluster)
	defer viper.Reset()

	orig := osconsolev1.ConsoleLink{}
	orig.Name = nsn.Name
	orig.Labels = map[string]string{
		"app.kubernetes.io/instance":   orig.Name,
		"app.kubernetes.io/namespace":  orig.Namespace,
		"app.kubernetes.io/managed-by": "jaeger-operator",
	}

	objs := []client.Object{
		v1.NewJaeger(nsn),
		&orig,
	}

	r, cl := getReconciler(objs)
	r.strategyChooser = func(ctx context.Context, jaeger *v1.Jaeger) strategy.S {
		return strategy.S{}
	}

	// test
	_, err := r.Reconcile(reconcile.Request{NamespacedName: nsn})
	require.NoError(t, err)

	// verify
	persisted := &osconsolev1.ConsoleLink{}
	persistedName := types.NamespacedName{
		Name:      orig.Name,
		Namespace: orig.Namespace,
	}
	err = cl.Get(context.Background(), persistedName, persisted)
	assert.Empty(t, persisted.Name)
	require.Error(t, err) // not found
}

func TestConsoleLinksCreateExistingNameInAnotherNamespace(t *testing.T) {
	// This test validate that creating a new jaeger instance with the same
	// name as another existing instance but in a different namespace does not interfere each other
	// Prepare
	// New instance to be created.
	nsn := types.NamespacedName{
		Name:      "my-instance-1",
		Namespace: "tenant1",
	}

	// Existing one
	nsnExisting := types.NamespacedName{
		Name:      "my-instance-1",
		Namespace: "tenant2",
	}

	viper.Set("platform", "openshift")
	viper.Set(v1.ConfigOperatorScope, v1.OperatorScopeCluster)

	defer viper.Reset()

	// Existing console link and route
	objs := []client.Object{
		v1.NewJaeger(nsn),
		v1.NewJaeger(nsnExisting),
		&osconsolev1.ConsoleLink{
			ObjectMeta: metav1.ObjectMeta{
				Name:      nsnExisting.Name,
				Namespace: nsnExisting.Namespace,
				Annotations: map[string]string{
					consolelink.RouteAnnotation: "my-route-1",
				},
			},
			Spec: osconsolev1.ConsoleLinkSpec{
				Link: osconsolev1.Link{
					Href: "https://host1",
				},
			},
		},
		&osroutev1.Route{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "my-route-1",
				Namespace: nsnExisting.Namespace,
			},
			Spec: osroutev1.RouteSpec{
				Host: "host1",
			},
		},
	}

	req := reconcile.Request{
		NamespacedName: nsn,
	}

	r, cl := getReconciler(objs)
	// New console link but in different namespace.
	r.strategyChooser = func(ctx context.Context, jaeger *v1.Jaeger) strategy.S {
		s := strategy.New().WithConsoleLinks([]osconsolev1.ConsoleLink{{
			ObjectMeta: metav1.ObjectMeta{
				Name:      nsn.Name,
				Namespace: nsn.Namespace,
				// Same route name and annotation
				Annotations: map[string]string{
					consolelink.RouteAnnotation: "my-route-1",
				},
			},
		}}).WithRoutes([]osroutev1.Route{{
			// Same route name as existing BUT different namespace
			ObjectMeta: metav1.ObjectMeta{
				Name:      "my-route-1",
				Namespace: nsn.Namespace,
			},
			// Set different host from existing, just for validate that the new link
			// will be associated with the correct route.
			Spec: osroutev1.RouteSpec{
				Host: "host2",
			},
		}})
		return s
	}

	// test
	res, err := r.Reconcile(req)

	// verify
	require.NoError(t, err)
	assert.False(t, res.Requeue, "We don't requeue for now")

	persisted := &osconsolev1.ConsoleLink{}
	err = cl.Get(context.Background(), nsn, persisted)
	require.NoError(t, err)
	assert.Equal(t, nsn.Name, persisted.Name)
	assert.Equal(t, nsn.Namespace, persisted.Namespace)
	// New instance should have Href=host2
	assert.Equal(t, "https://host2", persisted.Spec.Href)

	persistedExisting := &osconsolev1.ConsoleLink{}
	err = cl.Get(context.Background(), nsnExisting, persistedExisting)
	require.NoError(t, err)
	assert.Equal(t, nsnExisting.Name, persistedExisting.Name)
	assert.Equal(t, nsnExisting.Namespace, persistedExisting.Namespace)
	// Existing should have Href=host1, reconciliation should not touch existing instances.
	assert.Equal(t, "https://host1", persistedExisting.Spec.Href)
}

func TestConsoleLinksSkipped(t *testing.T) {
	namespace := "observability"
	viper.Set("platform", "openshift")
	viper.Set(v1.ConfigOperatorScope, v1.OperatorScopeNamespace)
	viper.Set(v1.ConfigWatchNamespace, namespace)
	defer viper.Reset()

	nsn := types.NamespacedName{
		Name:      "my-instance",
		Namespace: namespace,
	}

	objs := []client.Object{
		v1.NewJaeger(nsn),
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
				Annotations: map[string]string{
					consolelink.RouteAnnotation: "my-route-1",
				},
			},
		}}).WithRoutes([]osroutev1.Route{{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "my-route-1",
				Namespace: nsn.Namespace,
			},
			Spec: osroutev1.RouteSpec{
				Host: "host",
			},
		}})
		return s
	}

	// test
	res, err := r.Reconcile(req)

	// verify
	require.NoError(t, err)
	assert.False(t, res.Requeue, "We don't requeue for now")

	persisted := &osconsolev1.ConsoleLink{}
	persistedName := types.NamespacedName{
		Name:      nsn.Name,
		Namespace: nsn.Namespace,
	}
	err = cl.Get(context.Background(), persistedName, persisted)
	assert.Equal(t, metav1.StatusReasonNotFound, errors.ReasonForError(err))
}
