package upgrade

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes/scheme"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"

	v1 "github.com/jaegertracing/jaeger-operator/apis/v1"
)

func TestUpgradeDeprecatedOptionsv1_20_0NonConflicting(t *testing.T) {
	latestVersion := "1.20.0"
	nsn := types.NamespacedName{Name: "my-instance"}
	existing := v1.NewJaeger(nsn)
	existing.Status.Version = "1.19.0"
	existing.Spec.Collector.Options = v1.NewOptions(map[string]interface{}{
		"es.max-num-spans":         "100",
		"es-archive.max-num-spans": "101",
	})
	objs := []runtime.Object{existing}

	s := scheme.Scheme
	s.AddKnownTypes(v1.GroupVersion, &v1.Jaeger{})
	s.AddKnownTypes(v1.GroupVersion, &v1.JaegerList{})
	cl := fake.NewClientBuilder().WithScheme(s).WithRuntimeObjects(objs...).Build()

	// test
	require.NoError(t, ManagedInstances(context.Background(), cl, cl, latestVersion))

	// verify
	persisted := &v1.Jaeger{}
	require.NoError(t, cl.Get(context.Background(), nsn, persisted))
	assert.Equal(t, latestVersion, persisted.Status.Version)

	opts := persisted.Spec.Collector.Options.Map()
	assert.Contains(t, opts, "es.max-doc-count")
	assert.Equal(t, "100", opts["es.max-doc-count"])
	assert.NotContains(t, opts, "es.max-num-spans")

	assert.Contains(t, opts, "es-archive.max-doc-count")
	assert.Equal(t, "101", opts["es-archive.max-doc-count"])
	assert.NotContains(t, opts, "es-archive.max-num-spans")
}
