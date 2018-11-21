package jaeger

import (
	"context"
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
)

type fakeStrategy struct {
	dependencies func() []batchv1.Job
	create       func() []runtime.Object
	update       func() []runtime.Object
}

func (c *fakeStrategy) Dependencies() []batchv1.Job {
	if nil == c.dependencies {
		return []batchv1.Job{}
	}
	return c.dependencies()
}

func (c *fakeStrategy) Create() []runtime.Object {
	if nil == c.create {
		return []runtime.Object{}
	}
	return c.create()
}

func (c *fakeStrategy) Update() []runtime.Object {
	if nil == c.update {
		return []runtime.Object{}
	}
	return c.update()
}

func TestNewJaegerInstance(t *testing.T) {
	// prepare
	jaeger := v1alpha1.NewJaeger("TestNewJaegerInstance")
	objs := []runtime.Object{
		jaeger,
	}

	s := scheme.Scheme
	s.AddKnownTypes(v1alpha1.SchemeGroupVersion, jaeger)
	cl := fake.NewFakeClient(objs...)
	r := &ReconcileJaeger{client: cl, scheme: s}

	req := reconcile.Request{
		NamespacedName: types.NamespacedName{
			Name:      jaeger.Name,
			Namespace: jaeger.Namespace,
		},
	}

	r.strategyChooser = func(jaeger *v1alpha1.Jaeger) Controller {
		jaeger.Spec.Strategy = "custom-strategy"
		return &fakeStrategy{}
	}

	// test
	res, err := r.Reconcile(req)

	// verify
	assert.NoError(t, err)
	assert.False(t, res.Requeue, "We don't requeue for now")
	assert.Empty(t, jaeger.Spec.Strategy)

	persisted := &v1alpha1.Jaeger{}
	err = cl.Get(context.Background(), req.NamespacedName, persisted)
	assert.Equal(t, persisted.Name, jaeger.Name)
	assert.NoError(t, err)

	// these are filled with default values
	assert.Equal(t, "custom-strategy", persisted.Spec.Strategy)
}

func TestDeletedInstance(t *testing.T) {
	// prepare

	// we should just not fail, as there won't be anything to do
	// all our objects should have an OwnerReference, so that when the jaeger object is deleted, the owned objects are deleted as well
	jaeger := v1alpha1.NewJaeger("TestNewJaegerInstance")
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

func TestHandleDependenciesSuccess(t *testing.T) {
	// prepare
	jaeger := v1alpha1.NewJaeger("TestHandleDependenciesSuccess")
	objs := []runtime.Object{
		jaeger,
	}

	s := scheme.Scheme
	s.AddKnownTypes(v1alpha1.SchemeGroupVersion, jaeger)
	cl := fake.NewFakeClient(objs...)
	r := &ReconcileJaeger{client: cl, scheme: s}

	req := reconcile.Request{
		NamespacedName: types.NamespacedName{
			Name:      jaeger.Name,
			Namespace: jaeger.Namespace,
		},
	}

	deadline := int64(1)
	batchJob := batchv1.Job{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "dependency",
			Namespace: jaeger.Namespace,
		},
		Spec: batchv1.JobSpec{
			ActiveDeadlineSeconds: &deadline,
		},
		Status: batchv1.JobStatus{
			Succeeded: 1,
		},
	}

	handled := false
	r.strategyChooser = func(jaeger *v1alpha1.Jaeger) Controller {
		return &fakeStrategy{
			dependencies: func() []batchv1.Job {
				handled = true
				return []batchv1.Job{
					batchJob,
				}
			},
			create: func() []runtime.Object {
				assert.True(t, handled) // dependencies have been handled at this point!
				return []runtime.Object{}
			},
		}
	}

	// test
	res, err := r.Reconcile(req)

	// verify
	assert.NoError(t, err)
	assert.False(t, res.Requeue)
	assert.True(t, handled)
}

func TestHandleDependenciesDoesNotComplete(t *testing.T) {
	// prepare
	jaeger := v1alpha1.NewJaeger("TestHandleDependenciesDoesNotComplete")
	objs := []runtime.Object{
		jaeger,
	}

	s := scheme.Scheme
	s.AddKnownTypes(v1alpha1.SchemeGroupVersion, jaeger)
	cl := fake.NewFakeClient(objs...)
	r := &ReconcileJaeger{client: cl, scheme: s}

	req := reconcile.Request{
		NamespacedName: types.NamespacedName{
			Name:      jaeger.Name,
			Namespace: jaeger.Namespace,
		},
	}

	deadline := int64(2) // 2 seconds deadline
	batchJob := batchv1.Job{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "dependency",
			Namespace: jaeger.Namespace,
		},
		Spec: batchv1.JobSpec{
			ActiveDeadlineSeconds: &deadline,
		},
		Status: batchv1.JobStatus{
			Succeeded: 0,
		},
	}

	handled := false
	r.strategyChooser = func(jaeger *v1alpha1.Jaeger) Controller {
		return &fakeStrategy{
			dependencies: func() []batchv1.Job {
				handled = true
				return []batchv1.Job{
					batchJob,
				}
			},
			create: func() []runtime.Object {
				assert.Fail(t, "Create should not have been called at all")
				return []runtime.Object{}
			},
		}
	}

	// test
	res, err := r.Reconcile(req)

	// verify
	assert.Error(t, err)
	assert.False(t, res.Requeue)
	assert.True(t, handled)
}

func TestHandleCreate(t *testing.T) {
	// prepare
	jaeger := v1alpha1.NewJaeger("TestHandleCreate")
	objs := []runtime.Object{
		jaeger,
	}

	s := scheme.Scheme
	s.AddKnownTypes(v1alpha1.SchemeGroupVersion, jaeger)
	cl := fake.NewFakeClient(objs...)
	r := &ReconcileJaeger{client: cl, scheme: s}

	req := reconcile.Request{
		NamespacedName: types.NamespacedName{
			Name:      jaeger.Name,
			Namespace: jaeger.Namespace,
		},
	}

	handled := false
	nsn := types.NamespacedName{
		Name:      "custom-deployment",
		Namespace: jaeger.Namespace,
	}
	r.strategyChooser = func(jaeger *v1alpha1.Jaeger) Controller {
		return &fakeStrategy{
			create: func() []runtime.Object {
				handled = true
				return []runtime.Object{
					&appsv1.Deployment{
						ObjectMeta: metav1.ObjectMeta{
							Name:      nsn.Name,
							Namespace: nsn.Namespace,
						},
					},
				}
			},
		}
	}

	// test
	res, err := r.Reconcile(req)

	// verify
	assert.NoError(t, err)
	assert.False(t, res.Requeue)
	assert.True(t, handled)

	retrieved := &appsv1.Deployment{}
	err = cl.Get(context.Background(), nsn, retrieved)
	assert.NoError(t, err)
	assert.Equal(t, nsn.Name, retrieved.Name)
}

func TestHandleUpdate(t *testing.T) {
	// prepare
	jaeger := v1alpha1.NewJaeger("TestHandleUpdate")
	objs := []runtime.Object{
		jaeger,
	}

	s := scheme.Scheme
	s.AddKnownTypes(v1alpha1.SchemeGroupVersion, jaeger)
	cl := fake.NewFakeClient(objs...)
	r := &ReconcileJaeger{client: cl, scheme: s}

	req := reconcile.Request{
		NamespacedName: types.NamespacedName{
			Name:      jaeger.Name,
			Namespace: jaeger.Namespace,
		},
	}

	handled := false
	nsn := types.NamespacedName{
		Name:      "custom-deployment-to-update",
		Namespace: jaeger.Namespace,
	}
	r.strategyChooser = func(jaeger *v1alpha1.Jaeger) Controller {
		return &fakeStrategy{
			create: func() []runtime.Object {
				handled = true
				return []runtime.Object{
					&appsv1.Deployment{
						ObjectMeta: metav1.ObjectMeta{
							Name:      nsn.Name,
							Namespace: nsn.Namespace,
							Annotations: map[string]string{
								"version": "1",
							},
						},
					},
				}
			},
			update: func() []runtime.Object {
				handled = true
				return []runtime.Object{
					&appsv1.Deployment{
						ObjectMeta: metav1.ObjectMeta{
							Name:      nsn.Name,
							Namespace: nsn.Namespace,
							Annotations: map[string]string{
								"version": "2",
							},
						},
					},
				}
			},
		}
	}

	// test
	res, err := r.Reconcile(req)

	// verify
	assert.NoError(t, err)
	assert.False(t, res.Requeue)
	assert.True(t, handled)

	retrieved := &appsv1.Deployment{}
	err = cl.Get(context.Background(), nsn, retrieved)
	assert.NoError(t, err)
	assert.Equal(t, nsn.Name, retrieved.Name)
	assert.Equal(t, "2", retrieved.Annotations["version"])
}
