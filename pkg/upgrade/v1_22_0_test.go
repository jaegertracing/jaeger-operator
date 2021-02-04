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

func TestUpgradeJaegerTagssv1_22_0(t *testing.T) {
	latestVersion := "1.22.0"
	opts := v1.NewOptions(map[string]interface{}{
		"jaeger.tags": "somekey=somevalue",
	})

	nsn := types.NamespacedName{Name: "my-instance"}
	existing := v1.NewJaeger(nsn)
	existing.Status.Version = "1.21.0"
	existing.Spec.AllInOne.Options = opts
	existing.Spec.Agent.Options = opts
	existing.Spec.Collector.Options = opts
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

	aioOpts := persisted.Spec.AllInOne.Options.Map()
	assert.Contains(t, aioOpts, "collector.tags")
	assert.Equal(t, "somekey=somevalue", aioOpts["collector.tags"])
	assert.NotContains(t, aioOpts, "jaeger.tags")

	agOpts := persisted.Spec.Agent.Options.Map()
	assert.Contains(t, agOpts, "agent.tags")
	assert.Equal(t, "somekey=somevalue", agOpts["agent.tags"])
	assert.NotContains(t, agOpts, "jaeger.tags")

	colOpts := persisted.Spec.Collector.Options.Map()
	assert.Contains(t, colOpts, "collector.tags")
	assert.Equal(t, "somekey=somevalue", colOpts["collector.tags"])
	assert.NotContains(t, colOpts, "jaeger.tags")
}
