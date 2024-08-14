package appsv1_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
	"k8s.io/apimachinery/pkg/types"
	k8sconfig "sigs.k8s.io/controller-runtime/pkg/client/config"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	k8sreconcile "sigs.k8s.io/controller-runtime/pkg/reconcile"

	v1 "github.com/jaegertracing/jaeger-operator/apis/v1"
	"github.com/jaegertracing/jaeger-operator/controllers/appsv1"
)

func TestNamespaceControllerRegisterWithManager(t *testing.T) {
	t.Skip("this test requires a real cluster, otherwise the GetConfigOrDie will die")

	// prepare
	mgr, err := manager.New(k8sconfig.GetConfigOrDie(), manager.Options{})
	require.NoError(t, err)
	reconciler := appsv1.NewNamespaceReconciler(
		k8sClient,
		k8sClient,
		testScheme,
	)

	// test
	err = reconciler.SetupWithManager(mgr)

	// verify
	require.NoError(t, err)
}

func TestNewNamespaceInstance(t *testing.T) {
	// prepare
	nsn := types.NamespacedName{Name: "my-instance", Namespace: "default"}
	reconciler := appsv1.NewNamespaceReconciler(
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
