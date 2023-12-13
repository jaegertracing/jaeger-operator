package jaeger

import (
	"context"
	"testing"

	"github.com/spf13/viper"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	autoscalingv2 "k8s.io/api/autoscaling/v2"
	autoscalingv2beta2 "k8s.io/api/autoscaling/v2beta2"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	v1 "github.com/jaegertracing/jaeger-operator/apis/v1"
	"github.com/jaegertracing/jaeger-operator/pkg/strategy"
)

func TestHorizontalPodAutoscalerCreateV2(t *testing.T) {
	// prepare
	viper.SetDefault(v1.FlagAutoscalingVersion, v1.FlagAutoscalingVersionV2)
	nsn := types.NamespacedName{
		Name:      "TestHorizontalPodAutoscalerCreate",
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
		hpa := &autoscalingv2.HorizontalPodAutoscaler{
			ObjectMeta: metav1.ObjectMeta{
				Name:      nsn.Name,
				Namespace: nsn.Namespace,
			},
		}
		var autoscaler runtime.Object = hpa
		s := strategy.New().WithHorizontalPodAutoscaler([]runtime.Object{autoscaler})
		return s
	}

	// test
	res, err := r.Reconcile(req)

	// verify
	require.NoError(t, err)
	assert.False(t, res.Requeue, "We don't requeue for now")

	persisted := &autoscalingv2.HorizontalPodAutoscaler{}
	persistedName := types.NamespacedName{
		Name:      nsn.Name,
		Namespace: nsn.Namespace,
	}
	err = cl.Get(context.Background(), persistedName, persisted)
	assert.Equal(t, persistedName.Name, persisted.Name)
	require.NoError(t, err)
}

func TestHorizontalPodAutoscalerCreateV2Beta2(t *testing.T) {
	// prepare
	viper.SetDefault(v1.FlagAutoscalingVersion, v1.FlagAutoscalingVersionV2Beta2)
	nsn := types.NamespacedName{
		Name:      "TestHorizontalPodAutoscalerCreate",
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
		hpa := &autoscalingv2beta2.HorizontalPodAutoscaler{
			ObjectMeta: metav1.ObjectMeta{
				Name:      nsn.Name,
				Namespace: nsn.Namespace,
			},
		}
		var autoscaler runtime.Object = hpa
		s := strategy.New().WithHorizontalPodAutoscaler([]runtime.Object{autoscaler})
		return s
	}

	// test
	res, err := r.Reconcile(req)

	// verify
	require.NoError(t, err)
	assert.False(t, res.Requeue, "We don't requeue for now")

	persisted := &autoscalingv2beta2.HorizontalPodAutoscaler{}
	persistedName := types.NamespacedName{
		Name:      nsn.Name,
		Namespace: nsn.Namespace,
	}
	err = cl.Get(context.Background(), persistedName, persisted)
	assert.Equal(t, persistedName.Name, persisted.Name)
	require.NoError(t, err)
}

func TestHorizontalPodAutoscalerUpdateV2(t *testing.T) {
	// prepare
	viper.SetDefault(v1.FlagAutoscalingVersion, v1.FlagAutoscalingVersionV2)
	nsn := types.NamespacedName{
		Name:      "TestHorizontalPodAutoscalerUpdate",
		Namespace: "tenant1",
	}

	orig := autoscalingv2.HorizontalPodAutoscaler{}
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
		depUpdated := autoscalingv2.HorizontalPodAutoscaler{}
		depUpdated.Name = orig.Name
		depUpdated.Namespace = orig.Namespace
		depUpdated.Annotations = map[string]string{"key": "new-value"}

		var hpa runtime.Object = &depUpdated

		s := strategy.New().WithHorizontalPodAutoscaler([]runtime.Object{hpa})
		return s
	}

	// test
	_, err := r.Reconcile(reconcile.Request{NamespacedName: nsn})
	require.NoError(t, err)

	// verify
	persisted := &autoscalingv2.HorizontalPodAutoscaler{}
	persistedName := types.NamespacedName{
		Name:      orig.Name,
		Namespace: orig.Namespace,
	}
	err = cl.Get(context.Background(), persistedName, persisted)
	assert.Equal(t, "new-value", persisted.Annotations["key"])
	require.NoError(t, err)
}

func TestHorizontalPodAutoscalerUpdateV2Beta2(t *testing.T) {
	// prepare
	viper.SetDefault(v1.FlagAutoscalingVersion, v1.FlagAutoscalingVersionV2Beta2)
	nsn := types.NamespacedName{
		Name:      "TestHorizontalPodAutoscalerUpdate",
		Namespace: "tenant1",
	}

	orig := autoscalingv2beta2.HorizontalPodAutoscaler{}
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
		depUpdated := autoscalingv2beta2.HorizontalPodAutoscaler{}
		depUpdated.Name = orig.Name
		depUpdated.Namespace = orig.Namespace
		depUpdated.Annotations = map[string]string{"key": "new-value"}

		var hpa runtime.Object = &depUpdated

		s := strategy.New().WithHorizontalPodAutoscaler([]runtime.Object{hpa})
		return s
	}

	// test
	_, err := r.Reconcile(reconcile.Request{NamespacedName: nsn})
	require.NoError(t, err)

	// verify
	persisted := &autoscalingv2beta2.HorizontalPodAutoscaler{}
	persistedName := types.NamespacedName{
		Name:      orig.Name,
		Namespace: orig.Namespace,
	}
	err = cl.Get(context.Background(), persistedName, persisted)
	assert.Equal(t, "new-value", persisted.Annotations["key"])
	require.NoError(t, err)
}

func TestHorizontalPodAutoscalerDeleteV2(t *testing.T) {
	// prepare
	viper.SetDefault(v1.FlagAutoscalingVersion, v1.FlagAutoscalingVersionV2)
	nsn := types.NamespacedName{
		Name: "TestHorizontalPodAutoscalerDelete",
	}

	orig := autoscalingv2.HorizontalPodAutoscaler{}
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
	persisted := &autoscalingv2.HorizontalPodAutoscaler{}
	persistedName := types.NamespacedName{
		Name:      orig.Name,
		Namespace: orig.Namespace,
	}
	err = cl.Get(context.Background(), persistedName, persisted)
	assert.Empty(t, persisted.Name)
	require.Error(t, err) // not found
}

func TestHorizontalPodAutoscalerDeleteV2Beta2(t *testing.T) {
	// prepare
	viper.SetDefault(v1.FlagAutoscalingVersion, v1.FlagAutoscalingVersionV2Beta2)
	nsn := types.NamespacedName{
		Name: "TestHorizontalPodAutoscalerDelete",
	}

	orig := autoscalingv2beta2.HorizontalPodAutoscaler{}
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
	persisted := &autoscalingv2beta2.HorizontalPodAutoscaler{}
	persistedName := types.NamespacedName{
		Name:      orig.Name,
		Namespace: orig.Namespace,
	}
	err = cl.Get(context.Background(), persistedName, persisted)
	assert.Empty(t, persisted.Name)
	require.Error(t, err) // not found
}

func TestHorizontalPodAutoscalerCreateExistingNameInAnotherNamespaceV2(t *testing.T) {
	// prepare
	viper.SetDefault(v1.FlagAutoscalingVersion, v1.FlagAutoscalingVersionV2)
	nsn := types.NamespacedName{
		Name:      "TestHorizontalPodAutoscalerCreateExistingNameInAnotherNamespace",
		Namespace: "tenant1",
	}
	nsnExisting := types.NamespacedName{
		Name:      "TestHorizontalPodAutoscalerCreateExistingNameInAnotherNamespace",
		Namespace: "tenant2",
	}

	objs := []client.Object{
		v1.NewJaeger(nsn),
		v1.NewJaeger(nsnExisting),
		&autoscalingv2.HorizontalPodAutoscaler{
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
		var hpa runtime.Object = &autoscalingv2.HorizontalPodAutoscaler{
			ObjectMeta: metav1.ObjectMeta{
				Name:      nsn.Name,
				Namespace: nsn.Namespace,
			},
		}
		s := strategy.New().WithHorizontalPodAutoscaler([]runtime.Object{hpa})
		return s
	}

	// test
	res, err := r.Reconcile(req)

	// verify
	require.NoError(t, err)
	assert.False(t, res.Requeue, "We don't requeue for now")

	persisted := &autoscalingv2.HorizontalPodAutoscaler{}
	err = cl.Get(context.Background(), nsn, persisted)
	require.NoError(t, err)
	assert.Equal(t, nsn.Name, persisted.Name)
	assert.Equal(t, nsn.Namespace, persisted.Namespace)

	persistedExisting := &autoscalingv2.HorizontalPodAutoscaler{}
	err = cl.Get(context.Background(), nsnExisting, persistedExisting)
	require.NoError(t, err)
	assert.Equal(t, nsnExisting.Name, persistedExisting.Name)
	assert.Equal(t, nsnExisting.Namespace, persistedExisting.Namespace)
}

func TestHorizontalPodAutoscalerCreateExistingNameInAnotherNamespaceV2Beta2(t *testing.T) {
	// prepare
	viper.SetDefault(v1.FlagAutoscalingVersion, v1.FlagAutoscalingVersionV2Beta2)
	nsn := types.NamespacedName{
		Name:      "TestHorizontalPodAutoscalerCreateExistingNameInAnotherNamespace",
		Namespace: "tenant1",
	}
	nsnExisting := types.NamespacedName{
		Name:      "TestHorizontalPodAutoscalerCreateExistingNameInAnotherNamespace",
		Namespace: "tenant2",
	}

	objs := []client.Object{
		v1.NewJaeger(nsn),
		v1.NewJaeger(nsnExisting),
		&autoscalingv2beta2.HorizontalPodAutoscaler{
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
		var hpa runtime.Object = &autoscalingv2beta2.HorizontalPodAutoscaler{
			ObjectMeta: metav1.ObjectMeta{
				Name:      nsn.Name,
				Namespace: nsn.Namespace,
			},
		}
		s := strategy.New().WithHorizontalPodAutoscaler([]runtime.Object{hpa})
		return s
	}

	// test
	res, err := r.Reconcile(req)

	// verify
	require.NoError(t, err)
	assert.False(t, res.Requeue, "We don't requeue for now")

	persisted := &autoscalingv2beta2.HorizontalPodAutoscaler{}
	err = cl.Get(context.Background(), nsn, persisted)
	require.NoError(t, err)
	assert.Equal(t, nsn.Name, persisted.Name)
	assert.Equal(t, nsn.Namespace, persisted.Namespace)

	persistedExisting := &autoscalingv2beta2.HorizontalPodAutoscaler{}
	err = cl.Get(context.Background(), nsnExisting, persistedExisting)
	require.NoError(t, err)
	assert.Equal(t, nsnExisting.Name, persistedExisting.Name)
	assert.Equal(t, nsnExisting.Namespace, persistedExisting.Namespace)
}
