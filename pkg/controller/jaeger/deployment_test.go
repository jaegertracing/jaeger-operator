package jaeger

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	appsv1 "k8s.io/api/apps/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	v1 "github.com/jaegertracing/jaeger-operator/apis/v1"
	"github.com/jaegertracing/jaeger-operator/pkg/strategy"
)

func TestDeploymentCreate(t *testing.T) {
	// prepare
	nsn := types.NamespacedName{
		Name:      "TestDeploymentCreate",
		Namespace: "tenant1",
	}

	objs := []client.Object{
		v1.NewJaeger(nsn),
	}

	req := reconcile.Request{
		NamespacedName: nsn,
	}

	r, cl := getReconciler(objs)
	r.strategyChooser = func(ctx context.Context, jaeger *v1.Jaeger) strategy.S {
		s := strategy.New().WithDeployments([]appsv1.Deployment{{
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
	require.NoError(t, err)
	assert.False(t, res.Requeue, "We don't requeue for now")

	persisted := &appsv1.Deployment{}
	persistedName := types.NamespacedName{
		Name:      nsn.Name,
		Namespace: nsn.Namespace,
	}
	err = cl.Get(context.Background(), persistedName, persisted)
	assert.Equal(t, persistedName.Name, persisted.Name)
	require.NoError(t, err)
}

func TestDeploymentUpdate(t *testing.T) {
	// prepare
	nsn := types.NamespacedName{
		Name:      "TestDeploymentUpdate",
		Namespace: "tenant1",
	}

	orig := appsv1.Deployment{}
	orig.Name = nsn.Name
	orig.Namespace = nsn.Namespace
	orig.Annotations = map[string]string{"key": "value"}
	orig.Labels = map[string]string{
		"app.kubernetes.io/instance":   orig.Name,
		"app.kubernetes.io/managed-by": "jaeger-operator",
	}

	objs := []client.Object{
		v1.NewJaeger(nsn),
		&orig,
	}

	r, cl := getReconciler(objs)
	r.strategyChooser = func(ctx context.Context, jaeger *v1.Jaeger) strategy.S {
		depUpdated := appsv1.Deployment{}
		depUpdated.Name = orig.Name
		depUpdated.Namespace = orig.Namespace
		depUpdated.Annotations = map[string]string{"key": "new-value"}

		s := strategy.New().WithDeployments([]appsv1.Deployment{depUpdated})
		return s
	}

	// test
	_, err := r.Reconcile(reconcile.Request{NamespacedName: nsn})
	require.NoError(t, err)

	// verify
	persisted := &appsv1.Deployment{}
	persistedName := types.NamespacedName{
		Name:      orig.Name,
		Namespace: orig.Namespace,
	}
	err = cl.Get(context.Background(), persistedName, persisted)
	assert.Equal(t, "new-value", persisted.Annotations["key"])
	require.NoError(t, err)
}

func TestDeploymentDelete(t *testing.T) {
	// prepare
	nsn := types.NamespacedName{
		Name: "TestDeploymentDelete",
	}

	orig := appsv1.Deployment{}
	orig.Name = nsn.Name
	orig.Labels = map[string]string{
		"app.kubernetes.io/instance":   orig.Name,
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
	persisted := &appsv1.Deployment{}
	persistedName := types.NamespacedName{
		Name:      orig.Name,
		Namespace: orig.Namespace,
	}
	err = cl.Get(context.Background(), persistedName, persisted)
	assert.Empty(t, persisted.Name)
	require.Error(t, err) // not found
}

func TestDeploymentDeleteAfterCreate(t *testing.T) {
	// prepare
	nsn := types.NamespacedName{
		Name: "TestDeploymentDeleteAfterCreate",
	}

	// the deployment to be deleted
	depToDelete := appsv1.Deployment{}
	depToDelete.Name = nsn.Name + "-delete"
	depToDelete.Labels = map[string]string{
		"app.kubernetes.io/instance":   nsn.Name,
		"app.kubernetes.io/managed-by": "jaeger-operator",
	}
	objs := []client.Object{v1.NewJaeger(nsn), &depToDelete}

	// the deployment to be created
	dep := appsv1.Deployment{}
	dep.Name = nsn.Name
	dep.Status.Replicas = 2
	dep.Status.ReadyReplicas = 1

	r, cl := getReconciler(objs)
	r.strategyChooser = func(ctx context.Context, jaeger *v1.Jaeger) strategy.S {
		s := strategy.New().WithDeployments([]appsv1.Deployment{dep})
		return s
	}

	// sanity check that the deployment to be removed is indeed there in the first place...
	persistedDelete := &appsv1.Deployment{}
	require.NoError(t, cl.Get(context.Background(), types.NamespacedName{Name: depToDelete.Name, Namespace: depToDelete.Namespace}, persistedDelete))
	assert.Equal(t, depToDelete.Name, persistedDelete.Name)

	// test
	wg := &sync.WaitGroup{}
	wg.Add(1)
	go func() {
		// will block until it finishes, which should happen after the deployments
		// have achieved stability and everything has been created/updated/deleted
		_, err := r.Reconcile(reconcile.Request{NamespacedName: nsn})
		require.NoError(t, err)
		wg.Done()
	}()

	// we assume that this sleep time is enough for the reconcile to reach the "wait" logic
	time.Sleep(100 * time.Millisecond)

	persisted := &appsv1.Deployment{}
	require.NoError(t, cl.Get(context.Background(), nsn, persisted))
	persisted.Status.ReadyReplicas = 2
	require.NoError(t, cl.Status().Update(context.Background(), persisted))

	wg.Wait() // will block until the reconcile logic finishes

	// verify that the deployment to be created was created
	persisted = &appsv1.Deployment{}
	require.NoError(t, cl.Get(context.Background(), nsn, persisted))
	assert.Equal(t, nsn.Name, persisted.Name)

	// check that the deployment to be deleted was indeed deleted
	persistedDelete = &appsv1.Deployment{}
	require.Error(t, cl.Get(context.Background(), types.NamespacedName{Name: depToDelete.Name, Namespace: depToDelete.Namespace}, persistedDelete))
	assert.Empty(t, persistedDelete.Name)
}

func TestDeploymentCreateExistingNameInAnotherNamespace(t *testing.T) {
	// prepare
	nsn := types.NamespacedName{
		Name:      "TestDeploymentCreateExistingNameInAnotherNamespace",
		Namespace: "tenant1",
	}
	nsnExisting := types.NamespacedName{
		Name:      "TestDeploymentCreateExistingNameInAnotherNamespace",
		Namespace: "tenant2",
	}

	objs := []client.Object{
		v1.NewJaeger(nsn),
		v1.NewJaeger(nsnExisting),
		&appsv1.Deployment{
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
		s := strategy.New().WithDeployments([]appsv1.Deployment{{
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
	require.NoError(t, err)
	assert.False(t, res.Requeue, "We don't requeue for now")

	persisted := &appsv1.Deployment{}
	err = cl.Get(context.Background(), nsn, persisted)
	require.NoError(t, err)
	assert.Equal(t, nsn.Name, persisted.Name)
	assert.Equal(t, nsn.Namespace, persisted.Namespace)

	persistedExisting := &appsv1.Deployment{}
	err = cl.Get(context.Background(), nsnExisting, persistedExisting)
	require.NoError(t, err)
	assert.Equal(t, nsnExisting.Name, persistedExisting.Name)
	assert.Equal(t, nsnExisting.Namespace, persistedExisting.Namespace)
}
