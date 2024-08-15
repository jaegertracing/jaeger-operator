package elasticsearch_test

import (
	"context"
	"testing"

	esv1 "github.com/openshift/elasticsearch-operator/apis/logging/v1"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	k8sconfig "sigs.k8s.io/controller-runtime/pkg/client/config"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	k8sreconcile "sigs.k8s.io/controller-runtime/pkg/reconcile"

	v1 "github.com/jaegertracing/jaeger-operator/apis/v1"
	"github.com/jaegertracing/jaeger-operator/controllers/elasticsearch"
)

func TestElasticSearchSetupWithManager(t *testing.T) {
	t.Skip("this test requires a real cluster, otherwise the GetConfigOrDie will die")

	// prepare
	mgr, err := manager.New(k8sconfig.GetConfigOrDie(), manager.Options{})
	require.NoError(t, err)
	reconciler := elasticsearch.NewReconciler(
		k8sClient,
		k8sClient,
	)

	// test
	err = reconciler.SetupWithManager(mgr)

	// verify
	require.NoError(t, err)
}

func TestNewElasticSearchInstance(t *testing.T) {
	// prepare
	ns := &corev1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name: "test-ns",
		},
	}

	es := &esv1.Elasticsearch{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-es",
			Namespace: "test-ns",
		},
	}

	jaeger := v1.NewJaeger(types.NamespacedName{
		Name:      "test-jaeger",
		Namespace: "test-jaeger",
	})

	esv1.AddToScheme(testScheme)
	v1.AddToScheme(testScheme)

	client := fake.NewClientBuilder().WithRuntimeObjects(ns, es, jaeger).Build()
	reconciler := elasticsearch.NewReconciler(
		client,
		client,
	)

	req := k8sreconcile.Request{
		NamespacedName: types.NamespacedName{
			Name:      "test-es",
			Namespace: "test-ns",
		},
	}

	_, err := reconciler.Reconcile(context.Background(), req)
	require.NoError(t, err)
}
