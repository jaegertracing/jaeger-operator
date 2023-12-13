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

func TestUpgradeDeprecatedOptionsv1_17_0(t *testing.T) {
	latestVersion := "1.17.0"
	nsn := types.NamespacedName{Name: "my-instance"}
	existing := v1.NewJaeger(nsn)
	existing.Status.Version = "1.16.0"
	existing.Spec.Collector.Options = v1.NewOptions(map[string]interface{}{
		"collector.grpc.tls":    true,
		"reporter.grpc.tls":     true,
		"es.tls":                true,
		"es-archive.tls":        true,
		"cassandra.tls":         true,
		"cassandra-archive.tls": true,
		"kafka.consumer.tls":    true,
		"kafka.producer.tls":    true,
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
	for _, prefix := range []string{"collector.grpc", "reporter.grpc", "es", "es-archive", "cassandra", "cassandra-archive", "kafka.consumer", "kafka.producer"} {
		assert.Contains(t, opts, fmt.Sprintf("%s.tls.enabled", prefix))
		assert.Equal(t, "true", opts[fmt.Sprintf("%s.tls.enabled", prefix)])
		assert.NotContains(t, opts, fmt.Sprintf("%s.tls", prefix))
	}
}

func TestAddTLSOptionsForKafka_v1_17_0(t *testing.T) {
	nsn := types.NamespacedName{Name: "my-instance"}
	jaeger := v1.NewJaeger(nsn)
	jaeger.Status.Version = "1.16.0"
	jaeger.Spec.Collector.Options = v1.NewOptions(map[string]interface{}{
		"kafka.producer.authentication": "tls",
	})
	jaeger.Spec.Ingester.Options = v1.NewOptions(map[string]interface{}{
		"kafka.producer.authentication": "tls",
		"kafka.consumer.authentication": "tls",
	})
	jaeger.Spec.Storage.Options = v1.NewOptions(map[string]interface{}{
		"kafka.producer.authentication": "tls",
		"kafka.consumer.authentication": "tls",
	})

	result, err := upgrade1_17_0(context.Background(), nil, *jaeger)

	require.NoError(t, err)
	assert.Equal(t, "true", result.Spec.Collector.Options.Map()["kafka.producer.tls.enabled"])
	assert.Equal(t, "true", result.Spec.Ingester.Options.Map()["kafka.producer.tls.enabled"])
	assert.Equal(t, "true", result.Spec.Ingester.Options.Map()["kafka.consumer.tls.enabled"])
	assert.Equal(t, "true", result.Spec.Storage.Options.Map()["kafka.producer.tls.enabled"])
	assert.Equal(t, "true", result.Spec.Storage.Options.Map()["kafka.consumer.tls.enabled"])
}
