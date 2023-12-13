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

func TestUpgradeJaegerTagssv1_22_0(t *testing.T) {
	latestVersion := "1.22.0"

	opts := v1.NewOptions(map[string]interface{}{
		"jaeger.tags": "somekey=somevalue",
	})

	storageOpts := v1.NewOptions(map[string]interface{}{
		"server-urls": "https://example:9200",
	})

	ingressOpts := v1.NewOptions(map[string]interface{}{
		"ingres-option": "value",
	})

	nsn := types.NamespacedName{Name: "my-instance"}
	existing := v1.NewJaeger(nsn)
	existing.Status.Version = "1.21.0"
	existing.Spec.AllInOne.Options = opts
	existing.Spec.Agent.Options = opts
	existing.Spec.Collector.Options = opts
	existing.Spec.Storage.Options = storageOpts
	existing.Spec.Ingress.Options = ingressOpts

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

	assert.Equal(t, storageOpts.Map(), persisted.Spec.Storage.Options.Map())
	assert.Equal(t, ingressOpts.Map(), persisted.Spec.Ingress.Options.Map())
}

func TestDeleteQueryRemovedFlags(t *testing.T) {
	latestVersion := "1.22.0"
	opts := v1.NewOptions(map[string]interface{}{
		"downsampling.hashsalt": "somevalue",
		"downsampling.ratio":    "0.25",
	})

	nsn := types.NamespacedName{Name: "my-instance"}
	existing := v1.NewJaeger(nsn)
	existing.Status.Version = "1.21.0"
	existing.Spec.Query.Options = opts

	objs := []runtime.Object{existing}

	s := scheme.Scheme
	s.AddKnownTypes(v1.GroupVersion, &v1.Jaeger{})
	s.AddKnownTypes(v1.GroupVersion, &v1.JaegerList{})
	cl := fake.NewClientBuilder().WithScheme(s).WithRuntimeObjects(objs...).Build()
	require.NoError(t, ManagedInstances(context.Background(), cl, cl, latestVersion))

	persisted := &v1.Jaeger{}
	require.NoError(t, cl.Get(context.Background(), nsn, persisted))
	assert.Equal(t, latestVersion, persisted.Status.Version)
	assert.Empty(t, persisted.Spec.Query.Options.Map())
	assert.NotContains(t, persisted.Spec.Query.Options.Map(), "downsampling.hashsalt")
	assert.NotContains(t, persisted.Spec.Query.Options.Map(), "downsampling.ratio")
}

func TestCassandraVerifyHostFlags(t *testing.T) {
	oldFlag := "cassandra.tls.verify-host"
	newFlag := "cassandra.tls.skip-host-verify"

	tests := []struct {
		testName    string
		opts        v1.Options
		flagPresent bool
		flagValue   string
	}{
		{
			testName: "verify-host=true",
			opts: v1.NewOptions(map[string]interface{}{
				oldFlag: "true",
			}),
			flagPresent: false,
		},
		{
			testName: "verify-host=false",
			opts: v1.NewOptions(map[string]interface{}{
				oldFlag: "false",
			}),
			flagPresent: true,
			flagValue:   "true",
		},
	}
	latestVersion := "1.22.0"
	for _, tt := range tests {
		t.Run(tt.testName, func(t *testing.T) {
			nsn := types.NamespacedName{Name: "my-instance"}
			existing := v1.NewJaeger(nsn)
			existing.Status.Version = "1.21.0"
			existing.Spec.Collector.Options = tt.opts

			objs := []runtime.Object{existing}
			s := scheme.Scheme
			s.AddKnownTypes(v1.GroupVersion, &v1.Jaeger{})
			s.AddKnownTypes(v1.GroupVersion, &v1.JaegerList{})
			cl := fake.NewClientBuilder().WithScheme(s).WithRuntimeObjects(objs...).Build()
			require.NoError(t, ManagedInstances(context.Background(), cl, cl, latestVersion))

			persisted := &v1.Jaeger{}
			require.NoError(t, cl.Get(context.Background(), nsn, persisted))
			assert.Equal(t, latestVersion, persisted.Status.Version)
			if tt.flagPresent {
				assert.Len(t, persisted.Spec.Collector.Options.Map(), 1)
				assert.NotContains(t, persisted.Spec.Collector.Options.Map(), oldFlag)
				assert.Contains(t, persisted.Spec.Collector.Options.Map(), newFlag)
				assert.Equal(t, tt.flagValue, persisted.Spec.Collector.Options.Map()[newFlag])
			} else {
				assert.Empty(t, persisted.Spec.Collector.Options.Map())
				assert.NotContains(t, persisted.Spec.Collector.Options.Map(), oldFlag)
			}
		})
	}
}

func TestMigrateQueryHostPortFlagsv1_22_0(t *testing.T) {
	tests := []struct {
		testName    string
		opts        v1.Options
		expectedOps map[string]string
	}{
		{
			testName: "no old flags",
			opts: v1.NewOptions(map[string]interface{}{
				"query.grpc-server.host-port": ":8080",
				"query.http-server.host-port": ":8081",
			}),
			expectedOps: map[string]string{
				"query.grpc-server.host-port": ":8080",
				"query.http-server.host-port": ":8081",
			},
		},

		{
			testName: "both old flags",
			opts: v1.NewOptions(map[string]interface{}{
				"query.port":      "8080",
				"query.host-port": "localhost:8081",
			}),
			expectedOps: map[string]string{
				"query.grpc-server.host-port": ":8080",
				"query.http-server.host-port": ":8080",
			},
		},

		{
			testName: "with query.host-port",
			opts: v1.NewOptions(map[string]interface{}{
				"query.host-port": "localhost:8081",
			}),
			expectedOps: map[string]string{
				"query.grpc-server.host-port": "localhost:8081",
				"query.http-server.host-port": "localhost:8081",
			},
		},
		{
			testName: "with grpc-server.host-port set",
			opts: v1.NewOptions(map[string]interface{}{
				"query.host-port":             "localhost:8081",
				"query.grpc-server.host-port": "other:7777",
			}),
			expectedOps: map[string]string{
				"query.grpc-server.host-port": "other:7777",
				"query.http-server.host-port": "localhost:8081",
			},
		},
		{
			testName: "with grpc-server.host-port set and query.port",
			opts: v1.NewOptions(map[string]interface{}{
				"query.port":                  "8081",
				"query.grpc-server.host-port": "other:7777",
			}),
			expectedOps: map[string]string{
				"query.grpc-server.host-port": "other:7777",
				"query.http-server.host-port": ":8081",
			},
		},
		{
			testName: "with grpc/http-server.host-port set",
			opts: v1.NewOptions(map[string]interface{}{
				"query.host-port":             "localhost:8081",
				"query.grpc-server.host-port": "other:7777",
				"query.http-server.host-port": "other:9999",
			}),
			expectedOps: map[string]string{
				"query.grpc-server.host-port": "other:7777",
				"query.http-server.host-port": "other:9999",
			},
		},
	}
	latestVersion := "1.22.0"
	for _, tt := range tests {
		nsn := types.NamespacedName{Name: "my-instance"}
		existing := v1.NewJaeger(nsn)
		existing.Status.Version = "1.21.0"
		existing.Spec.Query.Options = tt.opts

		objs := []runtime.Object{existing}
		s := scheme.Scheme
		s.AddKnownTypes(v1.GroupVersion, &v1.Jaeger{})
		s.AddKnownTypes(v1.GroupVersion, &v1.JaegerList{})
		cl := fake.NewClientBuilder().WithScheme(s).WithRuntimeObjects(objs...).Build()
		require.NoError(t, ManagedInstances(context.Background(), cl, cl, latestVersion))

		persisted := &v1.Jaeger{}
		require.NoError(t, cl.Get(context.Background(), nsn, persisted))
		assert.Equal(t, latestVersion, persisted.Status.Version)
		assert.Equal(t, tt.expectedOps, persisted.Spec.Query.Options.StringMap())

	}
}
