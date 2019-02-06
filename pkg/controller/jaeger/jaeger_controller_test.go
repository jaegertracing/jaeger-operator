package jaeger

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	appsv1 "k8s.io/api/apps/v1"
	batchv1 "k8s.io/api/batch/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes/scheme"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	"github.com/jaegertracing/jaeger-operator/pkg/apis/io/v1alpha1"
	"github.com/jaegertracing/jaeger-operator/pkg/strategy"
)

func TestNewJaegerInstance(t *testing.T) {
	// prepare
	nsn := types.NamespacedName{
		Name: "TestNewJaegerInstance",
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
		jaeger.Spec.Strategy = "custom-strategy"
		return strategy.S{}
	}

	// test
	res, err := r.Reconcile(req)

	// verify
	assert.NoError(t, err)
	assert.False(t, res.Requeue, "We don't requeue for now")

	persisted := &v1alpha1.Jaeger{}
	err = cl.Get(context.Background(), req.NamespacedName, persisted)
	assert.Equal(t, persisted.Name, nsn.Name)
	assert.NoError(t, err)

	// these are filled with default values
	assert.Equal(t, "custom-strategy", persisted.Spec.Strategy)
}

func TestDeletedInstance(t *testing.T) {
	// prepare

	// we should just not fail, as there won't be anything to do
	// all our objects should have an OwnerReference, so that when the jaeger object is deleted, the owned objects are deleted as well
	jaeger := v1alpha1.NewJaeger("TestDeletedInstance")
	s := scheme.Scheme
	s.AddKnownTypes(v1alpha1.SchemeGroupVersion, jaeger)

	// no known objects
	cl := fake.NewFakeClient()
	r := &ReconcileJaeger{client: cl, scheme: s}

	req := reconcile.Request{
		NamespacedName: types.NamespacedName{
			Name:      jaeger.Name,
			Namespace: jaeger.Namespace,
		},
	}

	// test
	res, err := r.Reconcile(req)

	// verify
	assert.NoError(t, err)
	assert.False(t, res.Requeue, "We don't requeue for now")

	persisted := &v1alpha1.Jaeger{}
	err = cl.Get(context.Background(), req.NamespacedName, persisted)
	assert.NotEmpty(t, jaeger.Name)
	assert.Empty(t, persisted.Name) // this means that the object wasn't found
}

func TestCreateDeployments(t *testing.T) {
	// prepare
	nsn := types.NamespacedName{
		Name: "TestCreateDeployments",
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
		s := strategy.New().WithDeployments([]appsv1.Deployment{{
			ObjectMeta: metav1.ObjectMeta{
				Name: "my-deployment",
			},
		}})
		return s
	}

	// test
	res, err := r.Reconcile(req)

	// verify
	assert.NoError(t, err)
	assert.False(t, res.Requeue, "We don't requeue for now")

	persisted := &appsv1.Deployment{}
	persistedName := types.NamespacedName{
		Name:      "my-deployment",
		Namespace: nsn.Namespace,
	}
	err = cl.Get(context.Background(), persistedName, persisted)
	assert.Equal(t, persistedName.Name, persisted.Name)
	assert.NoError(t, err)
}

func TestUpdateDeployments(t *testing.T) {
	// prepare
	nsn := types.NamespacedName{
		Name: "TestUpdateDeployments",
	}

	depOriginal := appsv1.Deployment{}
	depOriginal.Name = nsn.Name
	depOriginal.Annotations = map[string]string{"key": "value"}

	objs := []runtime.Object{
		v1alpha1.NewJaeger(nsn.Name),
		&depOriginal,
	}

	s := scheme.Scheme
	s.AddKnownTypes(v1alpha1.SchemeGroupVersion, &v1alpha1.Jaeger{})
	cl := fake.NewFakeClient(objs...)
	r := &ReconcileJaeger{client: cl, scheme: s}

	r.strategyChooser = func(jaeger *v1alpha1.Jaeger) strategy.S {
		depUpdated := appsv1.Deployment{}
		depUpdated.Name = depOriginal.Name
		depUpdated.Annotations = map[string]string{"key": "new-value"}

		s := strategy.New().WithDeployments([]appsv1.Deployment{depUpdated})
		return s
	}

	// test
	_, err := r.Reconcile(reconcile.Request{NamespacedName: nsn})
	assert.NoError(t, err)

	// verify
	persisted := &appsv1.Deployment{}
	persistedName := types.NamespacedName{
		Name:      depOriginal.Name,
		Namespace: depOriginal.Namespace,
	}
	err = cl.Get(context.Background(), persistedName, persisted)
	assert.Equal(t, "new-value", persisted.Annotations["key"])
	assert.NoError(t, err)
}

func TestDeleteDeployments(t *testing.T) {
	// prepare
	nsn := types.NamespacedName{
		Name: "TestDeleteDeployments",
	}

	depOriginal := appsv1.Deployment{}
	depOriginal.Name = nsn.Name

	objs := []runtime.Object{
		v1alpha1.NewJaeger(nsn.Name),
		&depOriginal,
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
	persisted := &appsv1.Deployment{}
	persistedName := types.NamespacedName{
		Name:      depOriginal.Name,
		Namespace: depOriginal.Namespace,
	}
	err = cl.Get(context.Background(), persistedName, persisted)
	assert.Empty(t, persisted.Name)
	assert.Error(t, err) // not found
}

func TestDeleteOnlyAfterSuccessfulUpdate(t *testing.T) {
	// prepare
	nsn := types.NamespacedName{
		Name: "TestDeleteOnlyAfterSuccessfulUpdate",
	}

	// the deployment to be deleted
	depToDelete := appsv1.Deployment{}
	depToDelete.Name = nsn.Name + "-delete"
	depToDelete.Annotations = map[string]string{
		"app.kubernetes.io/instance":   nsn.Name,
		"app.kubernetes.io/managed-by": "jaeger-operator",
	}
	objs := []runtime.Object{v1alpha1.NewJaeger(nsn.Name), &depToDelete}

	s := scheme.Scheme
	s.AddKnownTypes(v1alpha1.SchemeGroupVersion, &v1alpha1.Jaeger{})
	cl := fake.NewFakeClient(objs...)
	r := &ReconcileJaeger{client: cl, scheme: s}

	// the deployment to be created
	dep := appsv1.Deployment{}
	dep.Name = nsn.Name
	dep.Status.Replicas = 2
	dep.Status.ReadyReplicas = 1

	r.strategyChooser = func(jaeger *v1alpha1.Jaeger) strategy.S {
		s := strategy.New().WithDeployments([]appsv1.Deployment{dep})
		return s
	}

	// sanity check that the deployment to be removed is indeed there in the first place...
	persistedDelete := &appsv1.Deployment{}
	assert.NoError(t, cl.Get(context.Background(), types.NamespacedName{Name: depToDelete.Name, Namespace: depToDelete.Namespace}, persistedDelete))
	assert.Equal(t, depToDelete.Name, persistedDelete.Name)

	// test
	wg := &sync.WaitGroup{}
	wg.Add(1)
	go func() {
		// will block until it finishes, which should happen after the deployments
		// have achieved stability and everything has been created/updated/deleted
		_, err := r.Reconcile(reconcile.Request{NamespacedName: nsn})
		assert.NoError(t, err)
		wg.Done()
	}()

	// we assume that this sleep time is enough for the reconcile to reach the "wait" logic
	time.Sleep(100 * time.Millisecond)

	persisted := &appsv1.Deployment{}
	assert.NoError(t, cl.Get(context.Background(), nsn, persisted))
	persisted.Status.ReadyReplicas = 2
	assert.NoError(t, cl.Status().Update(context.Background(), persisted))

	wg.Wait() // will block until the reconcile logic finishes

	// verify that the deployment to be created was created
	persisted = &appsv1.Deployment{}
	assert.NoError(t, cl.Get(context.Background(), nsn, persisted))
	assert.Equal(t, nsn.Name, persisted.Name)

	// check that the deployment to be deleted was indeed deleted
	persistedDelete = &appsv1.Deployment{}
	assert.Error(t, cl.Get(context.Background(), types.NamespacedName{Name: depToDelete.Name, Namespace: depToDelete.Namespace}, persistedDelete))
	assert.Empty(t, persistedDelete.Name)
}

func TestHandleDependencies(t *testing.T) {
	// prepare
	nsn := types.NamespacedName{
		Name: "TestHandleDependencies",
	}

	objs := []runtime.Object{v1alpha1.NewJaeger(nsn.Name)}

	s := scheme.Scheme
	s.AddKnownTypes(v1alpha1.SchemeGroupVersion, &v1alpha1.Jaeger{})
	cl := fake.NewFakeClient(objs...)
	r := &ReconcileJaeger{client: cl, scheme: s}

	dep := batchv1.Job{}
	dep.Name = nsn.Name
	r.strategyChooser = func(jaeger *v1alpha1.Jaeger) strategy.S {
		s := strategy.New().WithDependencies([]batchv1.Job{dep})
		return s
	}

	// test
	wg := &sync.WaitGroup{}
	wg.Add(1)
	go func() {
		_, err := r.Reconcile(reconcile.Request{NamespacedName: nsn})
		assert.NoError(t, err)
		wg.Done()
	}()

	// we assume that this sleep time is enough for the reconcile to reach the "wait" logic
	time.Sleep(100 * time.Millisecond)

	persisted := &batchv1.Job{}
	assert.NoError(t, cl.Get(context.Background(), nsn, persisted))
	persisted.Status.Succeeded = 1
	assert.NoError(t, cl.Status().Update(context.Background(), persisted))

	wg.Wait()

	// verify
	persisted = &batchv1.Job{}
	assert.NoError(t, cl.Get(context.Background(), nsn, persisted))
	assert.Equal(t, nsn.Name, persisted.Name)
}
