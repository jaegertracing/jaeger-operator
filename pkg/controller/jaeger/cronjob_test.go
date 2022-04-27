package jaeger

import (
	"context"
	"testing"

	"github.com/spf13/viper"
	batchv1 "k8s.io/api/batch/v1"

	"github.com/stretchr/testify/assert"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	v1 "github.com/jaegertracing/jaeger-operator/apis/v1"
	"github.com/jaegertracing/jaeger-operator/pkg/strategy"
)

func init() {
	viper.SetDefault(v1.FlagCronJobsVersion, v1.FlagCronJobsVersionBatchV1)
}

func TestCronJobsCreate(t *testing.T) {
	// prepare
	nsn := types.NamespacedName{
		Name: "TestCronJobsCreate",
	}

	objs := []runtime.Object{
		v1.NewJaeger(nsn),
	}

	req := reconcile.Request{
		NamespacedName: nsn,
	}

	r, cl := getReconciler(objs)
	r.strategyChooser = func(ctx context.Context, jaeger *v1.Jaeger) strategy.S {
		cj := &batchv1.CronJob{
			ObjectMeta: metav1.ObjectMeta{
				Name: nsn.Name,
			},
		}

		var cronjob runtime.Object = cj
		cronjobs := []runtime.Object{cronjob}

		s := strategy.New().WithCronJobs(cronjobs)
		return s
	}

	// test
	res, err := r.Reconcile(req)

	// verify
	assert.NoError(t, err)
	assert.False(t, res.Requeue, "We don't requeue for now")

	persisted := &batchv1.CronJob{}
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

	orig := batchv1.CronJob{}
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
		updated := batchv1.CronJob{}
		updated.Name = orig.Name
		updated.Annotations = map[string]string{"key": "new-value"}

		var updatedCronJob runtime.Object = &updated
		x := []runtime.Object{updatedCronJob}

		s := strategy.New().WithCronJobs(x)
		return s
	}

	// test
	_, err := r.Reconcile(reconcile.Request{NamespacedName: nsn})
	assert.NoError(t, err)

	// verify
	persisted := &batchv1.CronJob{}
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

	orig := batchv1.CronJob{}
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
	persisted := &batchv1.CronJob{}
	persistedName := types.NamespacedName{
		Name:      orig.Name,
		Namespace: orig.Namespace,
	}
	err = cl.Get(context.Background(), persistedName, persisted)
	assert.Empty(t, persisted.Name)
	assert.Error(t, err) // not found
}

func TestCronJobsCreateExistingNameInAnotherNamespace(t *testing.T) {
	// prepare
	nsn := types.NamespacedName{
		Name:      "TestCronJobsCreateExistingNameInAnotherNamespace",
		Namespace: "tenant1",
	}
	nsnExisting := types.NamespacedName{
		Name:      "TestCronJobsCreateExistingNameInAnotherNamespace",
		Namespace: "tenant2",
	}

	objs := []runtime.Object{
		v1.NewJaeger(nsn),
		v1.NewJaeger(nsnExisting),
		&batchv1.CronJob{
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

	cj := &batchv1.CronJob{
		ObjectMeta: metav1.ObjectMeta{
			Name:      nsn.Name,
			Namespace: nsn.Namespace,
		},
	}

	var updatedCronJob runtime.Object = cj
	cronjobs := []runtime.Object{updatedCronJob}

	r.strategyChooser = func(ctx context.Context, jaeger *v1.Jaeger) strategy.S {
		s := strategy.New().WithCronJobs(cronjobs)
		return s
	}

	// test
	res, err := r.Reconcile(req)

	// verify
	assert.NoError(t, err)
	assert.False(t, res.Requeue, "We don't requeue for now")

	persisted := &batchv1.CronJob{}
	err = cl.Get(context.Background(), nsn, persisted)
	assert.NoError(t, err)
	assert.Equal(t, nsn.Name, persisted.Name)
	assert.Equal(t, nsn.Namespace, persisted.Namespace)

	persistedExisting := &batchv1.CronJob{}
	err = cl.Get(context.Background(), nsnExisting, persistedExisting)
	assert.NoError(t, err)
	assert.Equal(t, nsnExisting.Name, persistedExisting.Name)
	assert.Equal(t, nsnExisting.Namespace, persistedExisting.Namespace)
}
