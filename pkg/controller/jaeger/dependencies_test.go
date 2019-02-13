package jaeger

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	batchv1 "k8s.io/api/batch/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes/scheme"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	"github.com/jaegertracing/jaeger-operator/pkg/apis/io/v1alpha1"
	"github.com/jaegertracing/jaeger-operator/pkg/strategy"
)

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
