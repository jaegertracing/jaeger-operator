package elasticsearch

import (
	"context"
	"testing"

	esv1 "github.com/openshift/elasticsearch-operator/apis/logging/v1"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes/scheme"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	v1 "github.com/jaegertracing/jaeger-operator/apis/v1"
)

func TestControllerReconcile(t *testing.T) {
	ns := &corev1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name: "prod",
		},
	}
	es := &esv1.Elasticsearch{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "my-es",
			Namespace: "prod",
		},
		Spec: esv1.ElasticsearchSpec{
			Nodes: []esv1.ElasticsearchNode{
				{
					Roles:     []esv1.ElasticsearchNodeRole{esv1.ElasticsearchRoleMaster, esv1.ElasticsearchRoleClient, esv1.ElasticsearchRoleData},
					NodeCount: 3,
				},
			},
		},
	}
	jaeger := &v1.Jaeger{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "jaeger",
			Namespace: "prod",
		},
		Spec: v1.JaegerSpec{
			Strategy: v1.DeploymentStrategyProduction,
			Storage: v1.JaegerStorageSpec{
				Type: "elasticsearch",
				Elasticsearch: v1.ElasticsearchSpec{
					// this will be updated
					NodeCount: 1,
					Name:      "my-es",
				},
			},
		},
	}

	esv1.AddToScheme(scheme.Scheme)
	v1.AddToScheme(scheme.Scheme)
	cl := fake.NewClientBuilder().WithRuntimeObjects(ns, es, jaeger).Build()
	reconciler := New(cl, cl)

	result, err := reconciler.Reconcile(context.Background(), reconcile.Request{
		NamespacedName: types.NamespacedName{
			Namespace: "prod",
			Name:      "my-es",
		},
	})
	require.NoError(t, err)
	assert.Equal(t, reconcile.Result{}, result)

	updated := &v1.Jaeger{}
	cl.Get(context.Background(), types.NamespacedName{
		Namespace: "prod",
		Name:      "jaeger",
	}, updated)
	assert.Equal(t, int32(3), updated.Spec.Storage.Elasticsearch.NodeCount)
}

func TestControllerReconcile_not_found(t *testing.T) {
	esv1.AddToScheme(scheme.Scheme)
	v1.AddToScheme(scheme.Scheme)
	cl := fake.NewClientBuilder().WithRuntimeObjects().Build()
	reconciler := New(cl, cl)

	result, err := reconciler.Reconcile(context.Background(), reconcile.Request{
		NamespacedName: types.NamespacedName{
			Namespace: "prod",
			Name:      "my-es",
		},
	})
	require.NoError(t, err)
	assert.Equal(t, reconcile.Result{}, result)
}
