package jaeger

import (
	"context"
	"testing"

	osv1 "github.com/openshift/api/route/v1"
	"github.com/stretchr/testify/assert"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes/scheme"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	v1 "github.com/jaegertracing/jaeger-operator/pkg/apis/jaegertracing/v1"
	esv1 "github.com/jaegertracing/jaeger-operator/pkg/storage/elasticsearch/v1"
	"github.com/jaegertracing/jaeger-operator/pkg/strategy"
)

func TestNewJaegerInstance(t *testing.T) {
	// prepare
	nsn := types.NamespacedName{
		Name: "TestNewJaegerInstance",
	}

	objs := []runtime.Object{
		v1.NewJaeger(nsn),
	}

	req := reconcile.Request{
		NamespacedName: nsn,
	}

	r, cl := getReconciler(objs)
	r.strategyChooser = func(jaeger *v1.Jaeger) strategy.S {
		jaeger.Spec.Strategy = "custom-strategy"
		return strategy.S{}
	}

	// test
	res, err := r.Reconcile(req)

	// verify
	assert.NoError(t, err)
	assert.False(t, res.Requeue, "We don't requeue for now")

	persisted := &v1.Jaeger{}
	err = cl.Get(context.Background(), req.NamespacedName, persisted)
	assert.Equal(t, persisted.Name, nsn.Name)
	assert.NoError(t, err)

	// these are filled with default values
	// TODO(jpkroehling): enable the assertion when the following issue is fixed:
	// https://github.com/jaegertracing/jaeger-operator/issues/231
	// assert.Equal(t, "custom-strategy", persisted.Spec.Strategy)
}

func TestDeletedInstance(t *testing.T) {
	// prepare

	// we should just not fail, as there won't be anything to do
	// all our objects should have an OwnerReference, so that when the jaeger object is deleted, the owned objects are deleted as well
	jaeger := v1.NewJaeger(types.NamespacedName{Name: "TestDeletedInstance"})
	s := scheme.Scheme
	s.AddKnownTypes(v1.SchemeGroupVersion, jaeger)

	// no known objects
	cl := fake.NewFakeClient()
	r := &ReconcileJaeger{client: cl, scheme: s}

	req := reconcile.Request{
		NamespacedName: types.NamespacedName{
			Name:      jaeger.Name,
			Namespace: jaeger.Namespace,
		},
	}

	// test
	res, err := r.Reconcile(req)

	// verify
	assert.NoError(t, err)
	assert.False(t, res.Requeue, "We don't requeue for now")

	persisted := &v1.Jaeger{}
	err = cl.Get(context.Background(), req.NamespacedName, persisted)
	assert.NotEmpty(t, jaeger.Name)
	assert.Empty(t, persisted.Name) // this means that the object wasn't found
}

func getReconciler(objs []runtime.Object) (*ReconcileJaeger, client.Client) {
	s := scheme.Scheme

	// OpenShift Route
	osv1.Install(s)

	// Jaeger
	s.AddKnownTypes(v1.SchemeGroupVersion, &v1.Jaeger{})

	// Jaeger's Elasticsearch
	s.AddKnownTypes(v1.SchemeGroupVersion, &esv1.Elasticsearch{}, &esv1.ElasticsearchList{})

	cl := fake.NewFakeClient(objs...)
	return &ReconcileJaeger{client: cl, scheme: s}, cl
}
