package start

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes/scheme"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"

	v1 "github.com/jaegertracing/jaeger-operator/pkg/apis/jaegertracing/v1"
)

func TestProvisionWithoutExistingInstance(t *testing.T) {
	// prepare
	nsn := types.NamespacedName{Name: "jaeger", Namespace: "default"}

	s := scheme.Scheme
	s.AddKnownTypes(v1.SchemeGroupVersion, &v1.Jaeger{})
	s.AddKnownTypes(v1.SchemeGroupVersion, &v1.JaegerList{})
	cl := fake.NewFakeClient()

	// test
	provisionOwnJaeger(context.Background(), cl, nsn.Namespace)

	// verify
	persisted := &v1.Jaeger{}
	assert.NoError(t, cl.Get(context.Background(), nsn, persisted))
	assert.Equal(t, nsn.Name, persisted.Name)
	assert.Equal(t, "badger", persisted.Spec.Storage.Type)
}

func TestProvisionWithExistingInstance(t *testing.T) {
	// prepare
	nsn := types.NamespacedName{Name: "jaeger", Namespace: "default"}
	existing := v1.NewJaeger(nsn)
	existing.Spec.Storage.Type = "memory"

	s := scheme.Scheme
	s.AddKnownTypes(v1.SchemeGroupVersion, &v1.Jaeger{})
	s.AddKnownTypes(v1.SchemeGroupVersion, &v1.JaegerList{})
	cl := fake.NewFakeClient(existing)

	// test
	provisionOwnJaeger(context.Background(), cl, nsn.Namespace)

	// verify
	persisted := &v1.Jaeger{}
	assert.NoError(t, cl.Get(context.Background(), nsn, persisted))
	assert.Equal(t, nsn.Name, persisted.Name)
	assert.Equal(t, "memory", persisted.Spec.Storage.Type)
}
