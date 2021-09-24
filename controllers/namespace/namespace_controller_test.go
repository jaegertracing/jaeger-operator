package namespace_test

import (
	"context"
	"testing"

	v1 "github.com/jaegertracing/jaeger-operator/apis/jaegertracing/v1"
	"github.com/jaegertracing/jaeger-operator/controllers/namespace"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"k8s.io/apimachinery/pkg/types"
	k8sconfig "sigs.k8s.io/controller-runtime/pkg/client/config"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	k8sreconcile "sigs.k8s.io/controller-runtime/pkg/reconcile"
)

func TestReconcilieNamespace(t *testing.T) {
	// prepare
	nsn := types.NamespacedName{Name: "my-instance", Namespace: "default"}
	reconciler := namespace.NewReconciler(
		k8sClient,
		k8sClient,
		testScheme,
	)

	instance := v1.NewJaeger(nsn)
	err := k8sClient.Create(context.Background(), instance)
	require.NoError(t, err)

	req := k8sreconcile.Request{
		NamespacedName: nsn,
	}

	_, err = reconciler.Reconcile(context.Background(), req)
	require.NoError(t, err)
}

func TestRegisterWithManager(t *testing.T) {
	t.Skip("this test requires a real cluster, otherwise the GetConfigOrDie will die")

	// prepare
	mgr, err := manager.New(k8sconfig.GetConfigOrDie(), manager.Options{})
	require.NoError(t, err)
	reconciler := namespace.NewReconciler(
		k8sClient,
		k8sClient,
		testScheme,
	)

	// test
	err = reconciler.SetupWithManager(mgr)

	// verify
	assert.NoError(t, err)
}
