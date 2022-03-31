package metrics

import (
	"context"
	"reflect"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/metric/global"
	"go.opentelemetry.io/otel/metric/number"
	"go.opentelemetry.io/otel/oteltest"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes/scheme"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"

	v1 "github.com/jaegertracing/jaeger-operator/apis/v1"
)

const AgentSideCar = "Sidecar"
const AgentDaemonSet = "Daemonset"

type expectedMetric struct {
	name   string
	labels []attribute.KeyValue
	value  int64
}

func assertLabelAndValues(t *testing.T, name string, batches []oteltest.Batch, expectedLabels []attribute.KeyValue, expectedValue int64) {
	var measurement oteltest.Measurement
	var found = false
	for _, b := range batches {
		for _, m := range b.Measurements {
			if m.Instrument.Descriptor().Name() == name && reflect.DeepEqual(expectedLabels, b.Labels) {
				measurement = m
				found = true
				break
			}
		}
	}
	assert.True(t, found, "Metric %s with labels %v not found", name, expectedLabels)
	v := oteltest.ResolveNumberByKind(t, number.Int64Kind, float64(expectedValue))
	assert.Equal(t, 0, measurement.Number.CompareNumber(number.Int64Kind, v),
		"Metric %s doesn't have expected value %d", name, expectedValue)

}

func newJaegerInstance(nsn types.NamespacedName, strategy v1.DeploymentStrategy,
	storage v1.JaegerStorageType, agentMode string) v1.Jaeger {
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

	cl := fake.NewFakeClientWithScheme(s, objs...)

	meter, provider := oteltest.NewMeterProvider()
	global.SetMeterProvider(provider)

	instancesObservedValue := newInstancesMetric(cl)
	err := instancesObservedValue.Setup(context.Background())
	require.NoError(t, err)
	meter.RunAsyncInstruments()

	for _, e := range expected {
		assertLabelAndValues(t, e.name, meter.MeasurementBatches, e.labels, e.value)
	}

	// Test deleting allinone
	err = cl.Delete(context.Background(), &jaegerAllInOne)
	require.NoError(t, err)

	// Reset measurement batches
	meter.MeasurementBatches = []oteltest.Batch{}
	meter.RunAsyncInstruments()

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
		assertLabelAndValues(t, e.name, meter.MeasurementBatches, e.labels, e.value)
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

	cl := fake.NewFakeClientWithScheme(s, objs...)
	meter, provider := oteltest.NewMeterProvider()
	global.SetMeterProvider(provider)

	instancesObservedValue := newInstancesMetric(cl)
	err := instancesObservedValue.Setup(context.Background())
	require.NoError(t, err)
	meter.RunAsyncInstruments()

	expectedMetric := newExpectedMetric(autoprovisioningMetric, attribute.String("type", "elasticsearch"), 1)
	assertLabelAndValues(t, expectedMetric.name, meter.MeasurementBatches, expectedMetric.labels, expectedMetric.value)

	// Test deleting autoprovisioning
	err = cl.Delete(context.Background(), &autoprovisioningInstance)
	require.NoError(t, err)

	// Reset measurement batches
	meter.MeasurementBatches = []oteltest.Batch{}
	meter.RunAsyncInstruments()

	expectedMetric = newExpectedMetric(autoprovisioningMetric, attribute.String("type", "elasticsearch"), 0)
	assertLabelAndValues(t, expectedMetric.name, meter.MeasurementBatches, expectedMetric.labels, expectedMetric.value)

	// Create no autoprovisioned instance
	err = cl.Delete(context.Background(), &noAutoProvisioningInstance)
	meter.MeasurementBatches = []oteltest.Batch{}
	meter.RunAsyncInstruments()
	expectedMetric = newExpectedMetric(autoprovisioningMetric, attribute.String("type", "elasticsearch"), 0)
	assertLabelAndValues(t, expectedMetric.name, meter.MeasurementBatches, expectedMetric.labels, expectedMetric.value)

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

	cl := fake.NewFakeClientWithScheme(s, objs...)
	meter, provider := oteltest.NewMeterProvider()
	global.SetMeterProvider(provider)

	instancesObservedValue := newInstancesMetric(cl)
	err := instancesObservedValue.Setup(context.Background())
	require.NoError(t, err)
	meter.RunAsyncInstruments()

	expectedMetric := newExpectedMetric(managedMetric, attribute.String("tool", "maistra-istio-operator"), 1)
	assertLabelAndValues(t, expectedMetric.name, meter.MeasurementBatches, expectedMetric.labels, expectedMetric.value)

	expectedMetric = newExpectedMetric(managedMetric, attribute.String("tool", "none"), 1)
	assertLabelAndValues(t, expectedMetric.name, meter.MeasurementBatches, expectedMetric.labels, expectedMetric.value)

	// Test deleting managed
	err = cl.Delete(context.Background(), &managed)
	require.NoError(t, err)

	// Reset measurement batches
	meter.MeasurementBatches = []oteltest.Batch{}
	meter.RunAsyncInstruments()

	expectedMetric = newExpectedMetric(managedMetric, attribute.String("tool", "maistra-istio-operator"), 0)
	assertLabelAndValues(t, expectedMetric.name, meter.MeasurementBatches, expectedMetric.labels, expectedMetric.value)
}
