package metrics

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/metric/global"
	"go.opentelemetry.io/otel/sdk/metric"
	"go.opentelemetry.io/otel/sdk/metric/metricdata"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes/scheme"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"

	v1 "github.com/jaegertracing/jaeger-operator/apis/v1"
)

const (
	AgentSideCar   = "Sidecar"
	AgentDaemonSet = "Daemonset"
)

type expectedMetric struct {
	name   string
	labels []attribute.KeyValue
	value  int64
}

func assertLabelAndValues(t *testing.T, name string, metrics metricdata.ResourceMetrics, expectedAttrs []attribute.KeyValue, expectedValue int64) {
	var matchingMetric metricdata.Metrics
	found := false
	for _, sm := range metrics.ScopeMetrics {
		for _, m := range sm.Metrics {
			if m.Name == name {
				matchingMetric = m
				found = true
				break
			}
		}
	}

	assert.True(t, found, "Metric %s not found", name)

	gauge, ok := matchingMetric.Data.(metricdata.Gauge[int64])
	assert.True(t, ok,
		"Metric %s doesn't have expected type %T, got %T", metricdata.Gauge[int64]{}, matchingMetric.Data,
	)

	expectedAttrSet := attribute.NewSet(expectedAttrs...)
	var matchingDP metricdata.DataPoint[int64]
	found = false
	for _, dp := range gauge.DataPoints {
		if expectedAttrSet.Equals(&dp.Attributes) {
			matchingDP = dp
			found = true
			break
		}
	}

	assert.True(t, found, "Metric %s doesn't have expected attributes %v", expectedAttrs)
	assert.Equal(t, expectedValue, matchingDP.Value,
		"Metric %s doesn't have expected value %d, got %d", name, expectedValue, matchingDP.Value)
}

func newJaegerInstance(nsn types.NamespacedName, strategy v1.DeploymentStrategy,
	storage v1.JaegerStorageType, agentMode string,
) v1.Jaeger {
	return v1.Jaeger{
		ObjectMeta: metav1.ObjectMeta{
			Name:      nsn.Name,
			Namespace: nsn.Namespace,
		},
		Spec: v1.JaegerSpec{
			Strategy: strategy,
			Storage: v1.JaegerStorageSpec{
				Type: storage,
			},
			Agent: v1.JaegerAgentSpec{
				Strategy: agentMode,
			},
		},
	}
}

func newExpectedMetric(name string, keyPair attribute.KeyValue, value int64) expectedMetric {
	return expectedMetric{
		name: instanceMetricName(name),
		labels: []attribute.KeyValue{
			keyPair,
		},
		value: value,
	}
}

func TestValueObservedMetrics(t *testing.T) {
	s := scheme.Scheme

	// Add jaeger to schema
	s.AddKnownTypes(v1.GroupVersion, &v1.Jaeger{}, &v1.JaegerList{})

	// Create jaeger instances
	jaegerAllInOne := newJaegerInstance(types.NamespacedName{
		Name:      "my-jaeger-allinone",
		Namespace: "test",
	}, v1.DeploymentStrategyAllInOne, v1.JaegerMemoryStorage, AgentSideCar)

	jaegerProd := newJaegerInstance(types.NamespacedName{
		Name:      "my-jaeger-prod",
		Namespace: "test",
	}, v1.DeploymentStrategyProduction, v1.JaegerESStorage, AgentSideCar)

	jaegerOtherProd := newJaegerInstance(types.NamespacedName{
		Name:      "my-jaeger-other-prod",
		Namespace: "test",
	}, v1.DeploymentStrategyProduction, v1.JaegerESStorage, AgentDaemonSet)

	jaegerStream := newJaegerInstance(types.NamespacedName{
		Name:      "my-jaeger-stream",
		Namespace: "test",
	}, v1.DeploymentStrategyStreaming, v1.JaegerKafkaStorage, AgentSideCar)

	objs := []runtime.Object{
		&jaegerAllInOne,
		&jaegerProd,
		&jaegerOtherProd,
		&jaegerStream,
	}
	expected := []expectedMetric{
		newExpectedMetric(strategiesMetric, attribute.String("type", "allinone"), 1),
		newExpectedMetric(strategiesMetric, attribute.String("type", "production"), 2),
		newExpectedMetric(storageMetric, attribute.String("type", "memory"), 1),
		newExpectedMetric(storageMetric, attribute.String("type", "elasticsearch"), 2),
		newExpectedMetric(storageMetric, attribute.String("type", "kafka"), 1),
		newExpectedMetric(agentStrategiesMetric, attribute.String("type", "sidecar"), 3),
		newExpectedMetric(agentStrategiesMetric, attribute.String("type", "daemonset"), 1),
	}

	cl := fake.NewClientBuilder().WithScheme(s).WithRuntimeObjects(objs...).Build()

	reader := metric.NewManualReader()
	provider := metric.NewMeterProvider(metric.WithReader(reader))
	global.SetMeterProvider(provider)

	instancesObservedValue := newInstancesMetric(cl)
	err := instancesObservedValue.Setup(context.Background())
	require.NoError(t, err)

	metrics, err := reader.Collect(context.Background())
	require.NoError(t, err)

	for _, e := range expected {
		assertLabelAndValues(t, e.name, metrics, e.labels, e.value)
	}

	// Test deleting allinone
	err = cl.Delete(context.Background(), &jaegerAllInOne)
	require.NoError(t, err)

	// Reset measurement batches
	reader.ForceFlush(context.Background())
	metrics, err = reader.Collect(context.Background())
	require.NoError(t, err)

	// Set new numbers
	expected = []expectedMetric{
		newExpectedMetric(strategiesMetric, attribute.String("type", "allinone"), 0),
		newExpectedMetric(strategiesMetric, attribute.String("type", "production"), 2),
		newExpectedMetric(storageMetric, attribute.String("type", "memory"), 0),
		newExpectedMetric(storageMetric, attribute.String("type", "elasticsearch"), 2),
		newExpectedMetric(storageMetric, attribute.String("type", "kafka"), 1),
		newExpectedMetric(agentStrategiesMetric, attribute.String("type", "sidecar"), 2),
		newExpectedMetric(agentStrategiesMetric, attribute.String("type", "daemonset"), 1),
	}
	for _, e := range expected {
		assertLabelAndValues(t, e.name, metrics, e.labels, e.value)
	}
}

func TestAutoProvisioningESObservedMetric(t *testing.T) {
	s := scheme.Scheme
	s.AddKnownTypes(v1.GroupVersion, &v1.Jaeger{}, &v1.JaegerList{})

	nsn := types.NamespacedName{
		Name:      "my-jaeger-prod",
		Namespace: "test",
	}

	esOptionsMap := map[string]interface{}{
		"es.server-urls": "http://localhost:9200",
	}

	noAutoProvisioningInstance := v1.Jaeger{
		ObjectMeta: metav1.ObjectMeta{
			Name:      nsn.Name,
			Namespace: nsn.Namespace,
		},
		Spec: v1.JaegerSpec{
			Strategy: "production",
			Storage: v1.JaegerStorageSpec{
				Type:    v1.JaegerESStorage,
				Options: v1.NewOptions(esOptionsMap),
			},
		},
	}

	autoprovisioningInstance := v1.Jaeger{
		ObjectMeta: metav1.ObjectMeta{
			Name:      nsn.Name,
			Namespace: nsn.Namespace,
		},
		Spec: v1.JaegerSpec{
			Strategy: "production",
			Storage: v1.JaegerStorageSpec{
				Type: v1.JaegerESStorage,
			},
		},
	}

	objs := []runtime.Object{
		&autoprovisioningInstance,
	}

	cl := fake.NewClientBuilder().WithScheme(s).WithRuntimeObjects(objs...).Build()

	reader := metric.NewManualReader()
	provider := metric.NewMeterProvider(metric.WithReader(reader))
	global.SetMeterProvider(provider)

	instancesObservedValue := newInstancesMetric(cl)
	err := instancesObservedValue.Setup(context.Background())
	require.NoError(t, err)

	metrics, err := reader.Collect(context.Background())
	require.NoError(t, err)

	expectedMetric := newExpectedMetric(autoprovisioningMetric, attribute.String("type", "elasticsearch"), 1)
	assertLabelAndValues(t, expectedMetric.name, metrics, expectedMetric.labels, expectedMetric.value)

	// Test deleting autoprovisioning
	err = cl.Delete(context.Background(), &autoprovisioningInstance)
	require.NoError(t, err)

	// Reset measurement batches
	reader.ForceFlush(context.Background())
	metrics, err = reader.Collect(context.Background())
	require.NoError(t, err)

	expectedMetric = newExpectedMetric(autoprovisioningMetric, attribute.String("type", "elasticsearch"), 0)
	assertLabelAndValues(t, expectedMetric.name, metrics, expectedMetric.labels, expectedMetric.value)

	// Create no autoprovisioned instance
	_ = cl.Delete(context.Background(), &noAutoProvisioningInstance)

	reader.ForceFlush(context.Background())
	metrics, err = reader.Collect(context.Background())
	require.NoError(t, err)

	expectedMetric = newExpectedMetric(autoprovisioningMetric, attribute.String("type", "elasticsearch"), 0)
	assertLabelAndValues(t, expectedMetric.name, metrics, expectedMetric.labels, expectedMetric.value)
}

func TestManagerByMetric(t *testing.T) {
	s := scheme.Scheme
	s.AddKnownTypes(v1.GroupVersion, &v1.Jaeger{}, &v1.JaegerList{})

	managed := v1.Jaeger{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "jaeger-managed",
			Namespace: "ns",
			Labels: map[string]string{
				managedByLabel: "maistra-istio-operator",
			},
		},
		Spec: v1.JaegerSpec{
			Strategy: "allInOne",
			Storage: v1.JaegerStorageSpec{
				Type: v1.JaegerMemoryStorage,
			},
		},
	}

	nonManaged := v1.Jaeger{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "jaeger-no-managed",
			Namespace: "ns",
		},
		Spec: v1.JaegerSpec{
			Strategy: "allInOne",
			Storage: v1.JaegerStorageSpec{
				Type: v1.JaegerMemoryStorage,
			},
		},
	}

	objs := []runtime.Object{
		&managed,
		&nonManaged,
	}

	cl := fake.NewClientBuilder().WithScheme(s).WithRuntimeObjects(objs...).Build()

	reader := metric.NewManualReader()
	provider := metric.NewMeterProvider(metric.WithReader(reader))
	global.SetMeterProvider(provider)

	instancesObservedValue := newInstancesMetric(cl)
	err := instancesObservedValue.Setup(context.Background())
	require.NoError(t, err)

	metrics, err := reader.Collect(context.Background())
	require.NoError(t, err)

	expectedMetric := newExpectedMetric(managedMetric, attribute.String("tool", "maistra-istio-operator"), 1)
	assertLabelAndValues(t, expectedMetric.name, metrics, expectedMetric.labels, expectedMetric.value)

	expectedMetric = newExpectedMetric(managedMetric, attribute.String("tool", "none"), 1)
	assertLabelAndValues(t, expectedMetric.name, metrics, expectedMetric.labels, expectedMetric.value)

	// Test deleting managed
	err = cl.Delete(context.Background(), &managed)
	require.NoError(t, err)

	// Reset measurement batches
	reader.ForceFlush(context.Background())
	metrics, err = reader.Collect(context.Background())
	require.NoError(t, err)

	expectedMetric = newExpectedMetric(managedMetric, attribute.String("tool", "maistra-istio-operator"), 0)
	assertLabelAndValues(t, expectedMetric.name, metrics, expectedMetric.labels, expectedMetric.value)
}
