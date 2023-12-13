package jaeger

import (
	"context"
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	v1 "github.com/jaegertracing/jaeger-operator/apis/v1"
	"github.com/jaegertracing/jaeger-operator/pkg/strategy"
)

func TestConfigMapsCreate(t *testing.T) {
	// prepare
	nsn := types.NamespacedName{
		Name: "TestConfigMapsCreate",
	}

	objs := []client.Object{
		v1.NewJaeger(nsn),
	}

	req := reconcile.Request{
		NamespacedName: nsn,
	}

	r, cl := getReconciler(objs)
	r.strategyChooser = func(ctx context.Context, jaeger *v1.Jaeger) strategy.S {
		s := strategy.New().WithConfigMaps([]corev1.ConfigMap{{
			ObjectMeta: metav1.ObjectMeta{
				Name: nsn.Name,
			},
		}})
		return s
	}

	// test
	res, err := r.Reconcile(req)

	// verify
	require.NoError(t, err)
	assert.False(t, res.Requeue, "We don't requeue for now")

	persisted := &corev1.ConfigMap{}
	persistedName := types.NamespacedName{
		Name:      nsn.Name,
		Namespace: nsn.Namespace,
	}
	err = cl.Get(context.Background(), persistedName, persisted)
	assert.Equal(t, persistedName.Name, persisted.Name)
	require.NoError(t, err)
}

func TestConfigMapsUpdate(t *testing.T) {
	// prepare
	nsn := types.NamespacedName{
		Name: "TestConfigMapsUpdate",
	}

	orig := corev1.ConfigMap{}
	orig.Name = nsn.Name
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
		updated := corev1.ConfigMap{}
		updated.Name = orig.Name
		updated.Annotations = map[string]string{"key": "new-value"}

		s := strategy.New().WithConfigMaps([]corev1.ConfigMap{updated})
		return s
	}

	// test
	_, err := r.Reconcile(reconcile.Request{NamespacedName: nsn})
	require.NoError(t, err)

	// verify
	persisted := &corev1.ConfigMap{}
	persistedName := types.NamespacedName{
		Name:      orig.Name,
		Namespace: orig.Namespace,
	}
	err = cl.Get(context.Background(), persistedName, persisted)
	assert.Equal(t, "new-value", persisted.Annotations["key"])
	require.NoError(t, err)
}

func TestConfigMapsDelete(t *testing.T) {
	// prepare
	nsn := types.NamespacedName{
		Name: "TestConfigMapsDelete",
	}

	orig := corev1.ConfigMap{}
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
	persisted := &corev1.ConfigMap{}
	persistedName := types.NamespacedName{
		Name:      orig.Name,
		Namespace: orig.Namespace,
	}
	err = cl.Get(context.Background(), persistedName, persisted)
	assert.Empty(t, persisted.Name)
	require.Error(t, err) // not found
}

func TestConfigMapCreateExistingNameInAnotherNamespace(t *testing.T) {
	// prepare
	nsn := types.NamespacedName{
		Name:      "TestConfigMapCreateExistingNameInAnotherNamespace",
		Namespace: "tenant1",
	}
	nsnExisting := types.NamespacedName{
		Name:      "TestConfigMapCreateExistingNameInAnotherNamespace",
		Namespace: "tenant2",
	}

	objs := []client.Object{
		v1.NewJaeger(nsn),
		v1.NewJaeger(nsnExisting),
		&corev1.ConfigMap{
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
		s := strategy.New().WithConfigMaps([]corev1.ConfigMap{{
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

	persisted := &corev1.ConfigMap{}
	err = cl.Get(context.Background(), nsn, persisted)
	require.NoError(t, err)
	assert.Equal(t, nsn.Name, persisted.Name)
	assert.Equal(t, nsn.Namespace, persisted.Namespace)

	persistedExisting := &corev1.ConfigMap{}
	err = cl.Get(context.Background(), nsnExisting, persistedExisting)
	require.NoError(t, err)
	assert.Equal(t, nsnExisting.Name, persistedExisting.Name)
	assert.Equal(t, nsnExisting.Namespace, persistedExisting.Namespace)
}

func TestConfigMapsClean(t *testing.T) {
	// prepare
	nsnNonExist := types.NamespacedName{
		Name: "deleted-jaeger",
	}

	nsnExisting := types.NamespacedName{
		Name: "existing-jaeger",
	}

	// Create trusted CA config maps for non existing jaeger
	trustedCAConfig := &corev1.ConfigMap{}
	trustedCAConfig.Name = fmt.Sprintf("%s-trusted-ca", nsnNonExist.Name)
	trustedCAConfig.Labels = map[string]string{
		"app.kubernetes.io/name":       nsnNonExist.Name,
		"app.kubernetes.io/component":  "ca-configmap",
		"app.kubernetes.io/managed-by": "jaeger-operator",
	}

	serviceCAConfig := &corev1.ConfigMap{}
	serviceCAConfig.Name = fmt.Sprintf("%s-service-ca", nsnNonExist.Name)
	serviceCAConfig.Labels = map[string]string{
		"app.kubernetes.io/name":       nsnNonExist.Name,
		"app.kubernetes.io/component":  "service-ca-configmap",
		"app.kubernetes.io/managed-by": "jaeger-operator",
	}

	// Create trusted CA config maps for existing jaeger
	serviceCAConfigExist := &corev1.ConfigMap{}
	serviceCAConfigExist.Name = fmt.Sprintf("%s-service-ca", nsnExisting.Name)
	serviceCAConfigExist.Labels = map[string]string{
		"app.kubernetes.io/name":       nsnExisting.Name,
		"app.kubernetes.io/component":  "service-ca-configmap",
		"app.kubernetes.io/managed-by": "jaeger-operator",
	}

	objs := []client.Object{
		trustedCAConfig,
		serviceCAConfig,
		serviceCAConfigExist,
		v1.NewJaeger(nsnExisting),
	}

	r, cl := getReconciler(objs)

	// The three defined ConfigMaps exist
	configMaps := &corev1.ConfigMapList{}
	err := cl.List(context.Background(), configMaps)
	require.NoError(t, err)
	assert.Len(t, configMaps.Items, 3)

	// Reconcile non-exist jaeger
	_, err = r.Reconcile(reconcile.Request{NamespacedName: nsnNonExist})
	require.NoError(t, err)

	// Check that configmaps were clean up.
	err = cl.List(context.Background(), configMaps)
	require.NoError(t, err)
	assert.Len(t, configMaps.Items, 1)
	assert.Equal(t, fmt.Sprintf("%s-service-ca", nsnExisting.Name), configMaps.Items[0].Name)
}
