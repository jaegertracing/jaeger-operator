package upgrade

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes/scheme"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"

	v1 "github.com/jaegertracing/jaeger-operator/pkg/apis/jaegertracing/v1"
)

func TestUpgradeDeprecatedOptionsv1_17_0(t *testing.T) {
	nsn := types.NamespacedName{Name: "my-instance"}
	existing := v1.NewJaeger(nsn)
	existing.Status.Version = "1.16.0"
	existing.Spec.Collector.Options = v1.NewOptions(map[string]interface{}{
		"collector.grpc.tls": true,
	})
	objs := []runtime.Object{existing}

	s := scheme.Scheme
	s.AddKnownTypes(v1.SchemeGroupVersion, &v1.Jaeger{})
	s.AddKnownTypes(v1.SchemeGroupVersion, &v1.JaegerList{})
	cl := fake.NewFakeClient(objs...)

	// test
	assert.NoError(t, ManagedInstances(context.Background(), cl, cl))

	// verify
	persisted := &v1.Jaeger{}
	assert.NoError(t, cl.Get(context.Background(), nsn, persisted))
	assert.Equal(t, latest.v, persisted.Status.Version)

	opts := persisted.Spec.Collector.Options.Map()
	assert.Contains(t, opts, "collector.grpc.tls.enabled")
	assert.Equal(t, "true", opts["collector.grpc.tls.enabled"])
	assert.NotContains(t, opts, "collector.grpc.tls")
}
