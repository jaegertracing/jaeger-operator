package jaeger

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	batchv1beta1 "k8s.io/api/batch/v1beta1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	"github.com/jaegertracing/jaeger-operator/pkg/apis/io/v1alpha1"
	"github.com/jaegertracing/jaeger-operator/pkg/strategy"
)

func TestCronJobsCreate(t *testing.T) {
	// prepare
	nsn := types.NamespacedName{
		Name: "TestCronJobsCreate",
	}

	objs := []runtime.Object{
		v1alpha1.NewJaeger(nsn.Name),
	}

	req := reconcile.Request{
		NamespacedName: nsn,
	}

	r, cl := getReconciler(objs)
	r.strategyChooser = func(jaeger *v1alpha1.Jaeger) strategy.S {
		s := strategy.New().WithCronJobs([]batchv1beta1.CronJob{{
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

	persisted := &batchv1beta1.CronJob{}
	persistedName := types.NamespacedName{
		Name:      nsn.Name,
		Namespace: nsn.Namespace,
	}
	err = cl.Get(context.Background(), persistedName, persisted)
	assert.Equal(t, persistedName.Name, persisted.Name)
	assert.NoError(t, err)
}

func TestCronJobsUpdate(t *testing.T) {
	// prepare
	nsn := types.NamespacedName{
		Name: "TestCronJobsUpdate",
	}

	orig := batchv1beta1.CronJob{}
	orig.Name = nsn.Name
	orig.Annotations = map[string]string{"key": "value"}

	objs := []runtime.Object{
		v1alpha1.NewJaeger(nsn.Name),
		&orig,
	}

	r, cl := getReconciler(objs)
	r.strategyChooser = func(jaeger *v1alpha1.Jaeger) strategy.S {
		updated := batchv1beta1.CronJob{}
		updated.Name = orig.Name
		updated.Annotations = map[string]string{"key": "new-value"}

		s := strategy.New().WithCronJobs([]batchv1beta1.CronJob{updated})
		return s
	}

	// test
	_, err := r.Reconcile(reconcile.Request{NamespacedName: nsn})
	assert.NoError(t, err)

	// verify
	persisted := &batchv1beta1.CronJob{}
	persistedName := types.NamespacedName{
		Name:      orig.Name,
		Namespace: orig.Namespace,
	}
	err = cl.Get(context.Background(), persistedName, persisted)
	assert.Equal(t, "new-value", persisted.Annotations["key"])
	assert.NoError(t, err)
}

func TestCronJobsDelete(t *testing.T) {
	// prepare
	nsn := types.NamespacedName{
		Name: "TestCronJobsDelete",
	}

	orig := batchv1beta1.CronJob{}
	orig.Name = nsn.Name

	objs := []runtime.Object{
		v1alpha1.NewJaeger(nsn.Name),
		&orig,
	}

	r, cl := getReconciler(objs)
	r.strategyChooser = func(jaeger *v1alpha1.Jaeger) strategy.S {
		return strategy.S{}
	}

	// test
	_, err := r.Reconcile(reconcile.Request{NamespacedName: nsn})
	assert.NoError(t, err)

	// verify
	persisted := &batchv1beta1.CronJob{}
	persistedName := types.NamespacedName{
		Name:      orig.Name,
		Namespace: orig.Namespace,
	}
	err = cl.Get(context.Background(), persistedName, persisted)
	assert.Empty(t, persisted.Name)
	assert.Error(t, err) // not found
}
