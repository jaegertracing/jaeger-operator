package appsv1_test

import (
	"testing"

	"github.com/stretchr/testify/require"
	k8sconfig "sigs.k8s.io/controller-runtime/pkg/client/config"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/webhook"

	"github.com/jaegertracing/jaeger-operator/controllers/appsv1"
)

func TestPodWebhookRegisterWithManager(t *testing.T) {
	t.Skip("this test requires a real cluster, otherwise the GetConfigOrDie will die")

	// prepare
	mgr, err := manager.New(k8sconfig.GetConfigOrDie(), manager.Options{})
	require.NoError(t, err)

	// test
	mgr.GetWebhookServer().Register("/mutate-v1-pod", &webhook.Admission{
		Handler: appsv1.NewPodInjectorWebhook(
			k8sClient,
		),
	})
}
