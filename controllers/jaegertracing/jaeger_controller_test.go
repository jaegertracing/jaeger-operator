package jaegertracing_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
	"k8s.io/apimachinery/pkg/types"
	k8sconfig "sigs.k8s.io/controller-runtime/pkg/client/config"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	k8sreconcile "sigs.k8s.io/controller-runtime/pkg/reconcile"

	v1 "github.com/jaegertracing/jaeger-operator/apis/v1"
	"github.com/jaegertracing/jaeger-operator/controllers/jaegertracing"
)

func TestNewJaegerInstance(t *testing.T) {
	// prepare
	nsn := types.NamespacedName{Name: "my-instance", Namespace: "default"}
	reconciler := jaegertracing.NewReconciler(
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
	reconciler := jaegertracing.NewReconciler(
		k8sClient,
		k8sClient,
		testScheme,
	)

	// test
	err = reconciler.SetupWithManager(mgr)

	// verify
	require.NoError(t, err)
}
