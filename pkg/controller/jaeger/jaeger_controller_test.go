package jaeger

import (
	"context"
	"sync"
	"testing"

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

	dep.Status.Succeeded = 1
	wg.Wait()

	// verify
	persisted := &batchv1.Job{}
	err := cl.Get(context.Background(), nsn, persisted)
	assert.Equal(t, nsn.Name, persisted.Name)
	assert.NoError(t, err)
}
