package jaeger

import (
	"context"
	"testing"

	"github.com/spf13/viper"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	esv1 "github.com/openshift/elasticsearch-operator/apis/logging/v1"

	v1 "github.com/jaegertracing/jaeger-operator/apis/v1"
	"github.com/jaegertracing/jaeger-operator/pkg/autodetect"
	"github.com/jaegertracing/jaeger-operator/pkg/strategy"
)

func TestElasticsearchesCreate(t *testing.T) {
	autodetect.OperatorConfiguration.SetESIngration(autodetect.ESOperatorIntegrationYes)
	defer viper.Reset()

	// prepare
	nsn := types.NamespacedName{
		Name: "TestElasticsearchesCreate",
	}

	objs := []client.Object{
		v1.NewJaeger(nsn),
	}

	req := reconcile.Request{
		NamespacedName: nsn,
	}

	r, cl := getReconciler(objs)
	r.strategyChooser = func(ctx context.Context, jaeger *v1.Jaeger) strategy.S {
		s := strategy.New().WithElasticsearches([]esv1.Elasticsearch{{
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

	persisted := &esv1.Elasticsearch{}
	persistedName := types.NamespacedName{
		Name:      nsn.Name,
		Namespace: nsn.Namespace,
	}
	err = cl.Get(context.Background(), persistedName, persisted)
	assert.Equal(t, persistedName.Name, persisted.Name)
	require.NoError(t, err)
}

func TestElasticsearchesUpdate(t *testing.T) {
	autodetect.OperatorConfiguration.SetESIngration(autodetect.ESOperatorIntegrationYes)
	defer viper.Reset()

	// prepare
	nsn := types.NamespacedName{
		Name: "TestElasticsearchesUpdate",
	}

	orig := esv1.Elasticsearch{}
	orig.Name = nsn.Name
	orig.Annotations = map[string]string{"key": "value"}
	orig.Labels = map[string]string{
		"app.kubernetes.io/instance": nsn.Name,
		"app.kubernetes.io/part-of":  "jaeger",
	}

	objs := []client.Object{
		v1.NewJaeger(nsn),
		&orig,
	}

	r, cl := getReconciler(objs)
	r.strategyChooser = func(ctx context.Context, jaeger *v1.Jaeger) strategy.S {
		updated := esv1.Elasticsearch{}
		updated.Name = orig.Name
		updated.Annotations = map[string]string{"key": "new-value"}

		s := strategy.New().WithElasticsearches([]esv1.Elasticsearch{updated})
		return s
	}

	// test
	_, err := r.Reconcile(reconcile.Request{NamespacedName: nsn})
	require.NoError(t, err)

	// verify
	persisted := &esv1.Elasticsearch{}
	persistedName := types.NamespacedName{
		Name:      orig.Name,
		Namespace: orig.Namespace,
	}
	err = cl.Get(context.Background(), persistedName, persisted)
	assert.Equal(t, "new-value", persisted.Annotations["key"])
	require.NoError(t, err)
}

func TestElasticsearchesDelete(t *testing.T) {
	autodetect.OperatorConfiguration.SetESIngration(autodetect.ESOperatorIntegrationYes)
	defer viper.Reset()

	// prepare
	nsn := types.NamespacedName{
		Name: "TestElasticsearchesDelete",
	}

	orig := esv1.Elasticsearch{}
	orig.Name = nsn.Name
	orig.Labels = map[string]string{
		"app.kubernetes.io/instance": nsn.Name,
		"app.kubernetes.io/part-of":  "jaeger",
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
	persisted := &esv1.Elasticsearch{}
	persistedName := types.NamespacedName{
		Name:      orig.Name,
		Namespace: orig.Namespace,
	}
	err = cl.Get(context.Background(), persistedName, persisted)
	assert.Empty(t, persisted.Name)
	require.Error(t, err) // not found
}

func TestElasticsearchesCreateExistingNameInAnotherNamespace(t *testing.T) {
	// prepare
	autodetect.OperatorConfiguration.SetESIngration(autodetect.ESOperatorIntegrationYes)
	defer viper.Reset()

	nsn := types.NamespacedName{
		Name:      "TestElasticsearchesCreateExistingNameInAnotherNamespace",
		Namespace: "tenant1",
	}
	nsnExisting := types.NamespacedName{
		Name:      "TestElasticsearchesCreateExistingNameInAnotherNamespace",
		Namespace: "tenant2",
	}

	objs := []client.Object{
		v1.NewJaeger(nsn),
		v1.NewJaeger(nsnExisting),
		&esv1.Elasticsearch{
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
		s := strategy.New().WithElasticsearches([]esv1.Elasticsearch{{
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

	persisted := &esv1.Elasticsearch{}
	err = cl.Get(context.Background(), nsn, persisted)
	require.NoError(t, err)
	assert.Equal(t, nsn.Name, persisted.Name)
	assert.Equal(t, nsn.Namespace, persisted.Namespace)

	persistedExisting := &esv1.Elasticsearch{}
	err = cl.Get(context.Background(), nsnExisting, persistedExisting)
	require.NoError(t, err)
	assert.Equal(t, nsnExisting.Name, persistedExisting.Name)
	assert.Equal(t, nsnExisting.Namespace, persistedExisting.Namespace)
}
