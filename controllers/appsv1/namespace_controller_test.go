package appsv1_test

import (
	"testing"

	"github.com/stretchr/testify/require"
	k8sconfig "sigs.k8s.io/controller-runtime/pkg/client/config"
	"sigs.k8s.io/controller-runtime/pkg/manager"

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
