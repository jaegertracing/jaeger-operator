package metrics

import (
	"context"
	"fmt"
	"strings"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/metric"
	"go.opentelemetry.io/otel/metric/global"
	"sigs.k8s.io/controller-runtime/pkg/client"

	v1 "github.com/jaegertracing/jaeger-operator/pkg/apis/jaegertracing/v1"
)

const metricPrefix = "jaeger_operator_instances"
const agentStrategiesMetric = "agent_strategies"
const storageMetric = "storage_types"
const strategiesMetric = "strategies"

// This structure contains the labels associated with the instances and a counter of the number of instances
type instancesView struct {
	Name     string
	Label    string
	Count    map[string]int
	Observer *metric.Int64ValueObserver
	KeyFn    func(jaeger v1.Jaeger) string
}

func (i *instancesView) reset() {
	for k := range i.Count {
		i.Count[k] = 0
	}
}

func (i *instancesView) Record(jaeger v1.Jaeger) {
	i.Count[i.KeyFn(jaeger)]++
}

func (i *instancesView) Report(result metric.BatchObserverResult) {
	for key, count := range i.Count {
		result.Observe([]attribute.KeyValue{
			attribute.String(i.Label, key),
		}, i.Observer.Observation(int64(count)))
	}
}

type instancesMetric struct {
	client       client.Client
	observations []instancesView
}

func instanceMetricName(name string) string {
	return fmt.Sprintf("%s_%s", metricPrefix, name)
}

func newInstancesMetric(client client.Client) *instancesMetric {
	return &instancesMetric{
		client: client,
	}
}

func newObservation(batch metric.BatchObserver, name, desc, label string, keyFn func(jaeger v1.Jaeger) string) (instancesView, error) {
	observation := instancesView{
		Name:  name,
		Count: make(map[string]int),
		KeyFn: keyFn,
		Label: label,
	}
	obs, err := batch.NewInt64ValueObserver(instanceMetricName(name), metric.WithDescription(desc))
	if err != nil {
		return instancesView{}, err
	}
	observation.Observer = &obs
	return observation, nil
}

func (i *instancesMetric) Setup(ctx context.Context) error {
	tracer := otel.GetTracerProvider().Tracer(v1.BootstrapTracer)
	ctx, span := tracer.Start(ctx, "setup-jaeger-instances-metrics")
	defer span.End()
	meter := global.Meter(meterName)
	batch := meter.NewBatchObserver(i.callback)
	obs, err := newObservation(batch,
		agentStrategiesMetric,
		"Number of instances per agent strategy",
		"type",
		func(jaeger v1.Jaeger) string {
			return strings.ToLower(string(jaeger.Spec.Agent.Strategy))
		})

	if err != nil {
		return err
	}
	i.observations = append(i.observations, obs)
	obs, err = newObservation(batch, storageMetric,
		"Number of instances per storage type",
		"type",
		func(jaeger v1.Jaeger) string {
			return strings.ToLower(string(jaeger.Spec.Storage.Type))
		})
	if err != nil {
		return err
	}
	i.observations = append(i.observations, obs)

	obs, err = newObservation(batch, strategiesMetric,
		"Number of instances per strategy type",
		"type",
		func(jaeger v1.Jaeger) string {
			return strings.ToLower(string(jaeger.Spec.Strategy))
		})
	if err != nil {
		return err
	}
	i.observations = append(i.observations, obs)
	return nil
}

func isInstanceNormalized(jaeger v1.Jaeger) bool {
	return !(jaeger.Spec.Storage.Type == "" || jaeger.Spec.Strategy == "")
}

func normalizeAgentStrategy(jaeger *v1.Jaeger) {
	if jaeger.Spec.Agent.Strategy == "" {
		jaeger.Spec.Agent.Strategy = "Sidecar"
	}
}

func (i *instancesMetric) reset() {
	for _, o := range i.observations {
		o.reset()
	}
}

func (i *instancesMetric) report(result metric.BatchObserverResult) {
	for _, o := range i.observations {
		o.Report(result)
	}
}

func (i *instancesMetric) callback(ctx context.Context, result metric.BatchObserverResult) {
	instances := &v1.JaegerList{}
	if err := i.client.List(ctx, instances); err == nil {
		i.reset()
		for _, jaeger := range instances.Items {
			// Is this instance is already normalized by the reconciliation process
			// count it on the metrics, otherwise not.
			if isInstanceNormalized(jaeger) {
				// Normalization doesn't normalize agent mode. so we need to do it here.
				normalizeAgentStrategy(&jaeger)
				for _, o := range i.observations {
					o.Record(jaeger)
				}
			}
		}
		i.report(result)
	}
}
