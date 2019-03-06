package jaeger

import (
	"context"
	"testing"

	"github.com/spf13/viper"
	"github.com/stretchr/testify/assert"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	"github.com/jaegertracing/jaeger-operator/pkg/apis/jaegertracing/v1"
	esv1alpha1 "github.com/jaegertracing/jaeger-operator/pkg/storage/elasticsearch/v1alpha1"
	"github.com/jaegertracing/jaeger-operator/pkg/strategy"
)

func TestElasticsearchesCreate(t *testing.T) {
	viper.Set("es-provision", v1.FlagProvisionElasticsearchTrue)
	defer viper.Reset()

	// prepare
	nsn := types.NamespacedName{
		Name: "TestElasticsearchesCreate",
	}

	objs := []runtime.Object{
		v1.NewJaeger(nsn.Name),
	}

	req := reconcile.Request{
		NamespacedName: nsn,
	}

	r, cl := getReconciler(objs)
	r.strategyChooser = func(jaeger *v1.Jaeger) strategy.S {
		s := strategy.New().WithElasticsearches([]esv1alpha1.Elasticsearch{{
			ObjectMeta: metav1.ObjectMeta{
				Name: nsn.Name,
			},
		}})
		return s
	}

	// test
	res, err := r.Reconcile(req)

	// verify
	assert.NoError(t, err)
	assert.False(t, res.Requeue, "We don't requeue for now")

	persisted := &esv1alpha1.Elasticsearch{}
	persistedName := types.NamespacedName{
		Name:      nsn.Name,
		Namespace: nsn.Namespace,
	}
	err = cl.Get(context.Background(), persistedName, persisted)
	assert.Equal(t, persistedName.Name, persisted.Name)
	assert.NoError(t, err)
}

func TestElasticsearchesUpdate(t *testing.T) {
	viper.Set("es-provision", v1.FlagProvisionElasticsearchTrue)
	defer viper.Reset()

	// prepare
	nsn := types.NamespacedName{
		Name: "TestElasticsearchesUpdate",
	}

	orig := esv1alpha1.Elasticsearch{}
	orig.Name = nsn.Name
	orig.Annotations = map[string]string{"key": "value"}

	objs := []runtime.Object{
		v1.NewJaeger(nsn.Name),
		&orig,
	}

	r, cl := getReconciler(objs)
	r.strategyChooser = func(jaeger *v1.Jaeger) strategy.S {
		updated := esv1alpha1.Elasticsearch{}
		updated.Name = orig.Name
		updated.Annotations = map[string]string{"key": "new-value"}

		s := strategy.New().WithElasticsearches([]esv1alpha1.Elasticsearch{updated})
		return s
	}

	// test
	_, err := r.Reconcile(reconcile.Request{NamespacedName: nsn})
	assert.NoError(t, err)

	// verify
	persisted := &esv1alpha1.Elasticsearch{}
	persistedName := types.NamespacedName{
		Name:      orig.Name,
		Namespace: orig.Namespace,
	}
	err = cl.Get(context.Background(), persistedName, persisted)
	assert.Equal(t, "new-value", persisted.Annotations["key"])
	assert.NoError(t, err)
}

func TestElasticsearchesDelete(t *testing.T) {
	viper.Set("es-provision", v1.FlagProvisionElasticsearchTrue)
	defer viper.Reset()

	// prepare
	nsn := types.NamespacedName{
		Name: "TestElasticsearchesDelete",
	}

	orig := esv1alpha1.Elasticsearch{}
	orig.Name = nsn.Name

	objs := []runtime.Object{
		v1.NewJaeger(nsn.Name),
		&orig,
	}

	r, cl := getReconciler(objs)
	r.strategyChooser = func(jaeger *v1.Jaeger) strategy.S {
		return strategy.S{}
	}

	// test
	_, err := r.Reconcile(reconcile.Request{NamespacedName: nsn})
	assert.NoError(t, err)

	// verify
	persisted := &esv1alpha1.Elasticsearch{}
	persistedName := types.NamespacedName{
		Name:      orig.Name,
		Namespace: orig.Namespace,
	}
	err = cl.Get(context.Background(), persistedName, persisted)
	assert.Empty(t, persisted.Name)
	assert.Error(t, err) // not found
}
