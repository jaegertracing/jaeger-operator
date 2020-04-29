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

func TestUpgradeDeprecatedOptionsv1_15_0(t *testing.T) {
	latestVersion := "1.15.0"
	nsn := types.NamespacedName{Name: "my-instance"}
	existing := v1.NewJaeger(nsn)
	existing.Status.Version = "1.14.0"
	existing.Spec.Collector.Options = v1.NewOptions(map[string]interface{}{
		"collector.host-port": "jaeger.example.com:14268",
	})
	objs := []runtime.Object{existing}

	s := scheme.Scheme
	s.AddKnownTypes(v1.SchemeGroupVersion, &v1.Jaeger{})
	s.AddKnownTypes(v1.SchemeGroupVersion, &v1.JaegerList{})
	cl := fake.NewFakeClient(objs...)

	// test
	assert.NoError(t, ManagedInstances(context.Background(), cl, cl, latestVersion))

	// verify
	persisted := &v1.Jaeger{}
	assert.NoError(t, cl.Get(context.Background(), nsn, persisted))
	assert.Equal(t, latestVersion, persisted.Status.Version)

	opts := persisted.Spec.Collector.Options.Map()
	assert.Contains(t, opts, "reporter.tchannel.host-port")
	assert.Equal(t, "jaeger.example.com:14268", opts["reporter.tchannel.host-port"])
	assert.NotContains(t, opts, "collector.host-port")
}

func TestRemoveDeprecatedFlagWithNoReplacementv1_15_0(t *testing.T) {
	// prepare
	nsn := types.NamespacedName{Name: "my-instance"}
	existing := v1.NewJaeger(nsn)
	existing.Status.Version = "1.14.0"
	existing.Spec.Collector.Options = v1.NewOptions(map[string]interface{}{
		"cassandra.enable-dependencies-v2": "true",
	})

	// sanity check
	assert.Contains(t, existing.Spec.Collector.Options.Map(), "cassandra.enable-dependencies-v2")
	assert.Len(t, existing.Spec.Collector.Options.Map(), 1)

	// test
	updated, err := upgrade1_15_0(context.Background(), nil, *existing)

	// verify
	assert.NoError(t, err)
	assert.Len(t, updated.Spec.Collector.Options.Map(), 0)
	assert.NotContains(t, updated.Spec.Collector.Options.Map(), "cassandra.enable-dependencies-v2")

}
