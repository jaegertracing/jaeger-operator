package metrics

import (
	"context"
	"go.opentelemetry.io/otel/attribute"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/otel/metric/global"
	"go.opentelemetry.io/otel/metric/number"
	"go.opentelemetry.io/otel/oteltest"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes/scheme"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"

	"github.com/jaegertracing/jaeger-operator/pkg/apis/jaegertracing/v1"
)

func assertLabelAndValues(t *testing.T, name string, batches [] oteltest.Batch, expectedLabels []attribute.KeyValue, expectedValue int64) {
	var label []attribute.KeyValue
	var measurement oteltest.Measurement
	var found = false
	for _, b := range batches {
		for _, m := range b.Measurements {
			if m.Instrument.Descriptor().Name() == name {
				label = b.Labels
				measurement = m
				found = true
				break
			}
		}
	}
	require.True(t, found)
	assert.Equal(t, expectedLabels, label)
	v := oteltest.ResolveNumberByKind(t, number.Int64Kind, float64(expectedValue))
	assert.Equal(t, 0, measurement.Number.CompareNumber(number.Int64Kind, v))

}

func TestValueObservedMetrics(t *testing.T) {
	s := scheme.Scheme

	nsn := types.NamespacedName{
		Name:      "TestNewJaegerInstance",
		Namespace: "test",
	}

	// Jaeger
	s.AddKnownTypes(v1.SchemeGroupVersion, &v1.Jaeger{}, &v1.JaegerList{})

	jaegerInstance := &v1.Jaeger{
		ObjectMeta: metav1.ObjectMeta{
			Name:      nsn.Name,
			Namespace: nsn.Namespace,
		},
		Spec: v1.JaegerSpec{
			Strategy: v1.DeploymentStrategyAllInOne,
			Storage: v1.JaegerStorageSpec{
				Type: v1.JaegerMemoryStorage,
			},
			Agent: v1.JaegerAgentSpec{
				Strategy: "Sidecar",
			},
		},
	}

	objs := []runtime.Object{
		jaegerInstance,
	}

	cl := fake.NewFakeClientWithScheme(s, objs...)

	meter, provider := oteltest.NewMeterProvider()
	global.SetMeterProvider(provider)

	instancesObservedValue := newInstancesMetric(cl)
	err := instancesObservedValue.Setup(context.Background())
	require.NoError(t, err)
	meter.RunAsyncInstruments()

	assert.Len(t, meter.MeasurementBatches, 3)
	assertLabelAndValues(t, metricPrefix+"_"+strategiesMetric, meter.MeasurementBatches, []attribute.KeyValue{
		attribute.String("type", "allinone"),
	}, 1)

}
