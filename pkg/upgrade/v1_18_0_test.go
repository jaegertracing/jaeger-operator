package upgrade

import (
	"context"
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes/scheme"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"

	v1 "github.com/jaegertracing/jaeger-operator/apis/v1"
)

func TestUpgradeDeprecatedOptionsv1_18_0(t *testing.T) {
	latestVersion := "1.18.0"
	nsn := types.NamespacedName{Name: "my-instance"}
	existing := v1.NewJaeger(nsn)
	existing.Status.Version = "1.17.1"

	flagsMap := map[string]string{
		"collector.tchan-server.host-port": "collector.port",
		"collector.http-server.host-port":  "collector.http-port",
		"collector.grpc-server.host-port":  "collector.grpc-port",
		"collector.zipkin.host-port":       "collector.zipkin.http-port",
		"admin.http.host-port":             "admin-http-port",
	}

	oldOptionsMap := map[string]interface{}{
		"collector.port":             "4445",
		"collector.http-port":        "8080",
		"collector.grpc-port":        "14250",
		"collector.zipkin.http-port": "9411",
		"admin-http-port":            "14269",
	}

	existing.Spec.Collector.Options = v1.NewOptions(oldOptionsMap)
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
	// assert.Equal(t, latest.v, persisted.Status.Version)

	opts := persisted.Spec.Collector.Options.Map()
	for _, newFlag := range []string{
		"collector.tchan-server.host-port",
		"collector.http-server.host-port",
		"collector.grpc-server.host-port",
		"collector.zipkin.host-port",
		"admin.http.host-port",
	} {
		assert.Contains(t, opts, newFlag)
		expectedValue := fmt.Sprintf(":%s", oldOptionsMap[flagsMap[newFlag]])
		assert.Equal(t, expectedValue, opts[newFlag])
	}
}

func TestUpgradeAgentWithTChannelEnablev1_18_0_(t *testing.T) {
	latestVersion := "1.18.0"
	nsn := types.NamespacedName{Name: "my-instance"}
	existing := v1.NewJaeger(nsn)
	existing.Status.Version = "1.17.1"

	agentTchanelOptions := map[string]interface{}{
		"collector.host-port":                            "4445",
		"reporter.tchannel.discovery.conn-check-timeout": "5",
		"reporter.tchannel.discovery.min-peers":          "2",
		"reporter.tchannel.host-port":                    "8080",
		"reporter.tchannel.report-timeout":               "20",
	}

	existing.Spec.Agent.Options = v1.NewOptions(agentTchanelOptions)
	objs := []runtime.Object{existing}

	s := scheme.Scheme
	s.AddKnownTypes(v1.GroupVersion, &v1.Jaeger{})
	s.AddKnownTypes(v1.GroupVersion, &v1.JaegerList{})
	cl := fake.NewClientBuilder().WithScheme(s).WithRuntimeObjects(objs...).Build()

	require.NoError(t, ManagedInstances(context.Background(), cl, cl, latestVersion))

	// verify
	persisted := &v1.Jaeger{}
	require.NoError(t, cl.Get(context.Background(), nsn, persisted))

	collectorOpts := persisted.Spec.Agent.Options.Map()

	assert.NotContains(t, collectorOpts, "reporter.grpc.host-port")
}
