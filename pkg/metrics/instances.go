package metrics

import (
	"context"
	"fmt"
	"strings"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/metric"
	"go.opentelemetry.io/otel/metric/global"
	"go.opentelemetry.io/otel/metric/instrument"
	"sigs.k8s.io/controller-runtime/pkg/client"

	v1 "github.com/jaegertracing/jaeger-operator/apis/v1"
)

const (
	metricPrefix           = "jaeger_operator_instances"
	agentStrategiesMetric  = "agent_strategies"
	storageMetric          = "storage_types"
	strategiesMetric       = "strategies"
	autoprovisioningMetric = "autoprovisioning"
	managedMetric          = "managed"
	managedByLabel         = "app.kubernetes.io/managed-by"
)

// This structure contains the labels associated with the instances and a counter of the number of instances
type instancesView struct {
	Name  string
	Label string
	Count map[string]int
	Gauge instrument.Int64ObservableGauge
	KeyFn func(jaeger v1.Jaeger) string
}

func (i *instancesView) reset() {
	for k := range i.Count {
		i.Count[k] = 0
	}
}

func (i *instancesView) Record(jaeger v1.Jaeger) {
	label := i.KeyFn(jaeger)
	if label != "" {
		i.Count[label]++
	}
}

func (i *instancesView) Report(ctx context.Context, observer metric.Observer) {
	for key, count := range i.Count {
		attrs := []attribute.KeyValue{
			attribute.String(i.Label, key),
		}
		observer.ObserveInt64(i.Gauge, int64(count), attrs...)
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

func newObservation(meter metric.Meter, name, desc, label string, keyFn func(jaeger v1.Jaeger) string) (instancesView, error) {
	observation := instancesView{
		Name:  name,
		Count: make(map[string]int),
		KeyFn: keyFn,
		Label: label,
	}

	g, err := meter.Int64ObservableGauge(instanceMetricName(name), instrument.WithDescription(desc))
	if err != nil {
		return instancesView{}, err
	}

	observation.Gauge = g
	return observation, nil
}

func (i *instancesMetric) Setup(ctx context.Context) error {
	tracer := otel.GetTracerProvider().Tracer(v1.BootstrapTracer)
	_, span := tracer.Start(ctx, "setup-jaeger-instances-metrics") // nolint:ineffassign,staticcheck
	defer span.End()
	meter := global.Meter(meterName)

	obs, err := newObservation(meter,
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

	obs, err = newObservation(meter,
		storageMetric,
		"Number of instances per storage type",
		"type",
		func(jaeger v1.Jaeger) string {
			return strings.ToLower(string(jaeger.Spec.Storage.Type))
		})
	if err != nil {
		return err
	}
	i.observations = append(i.observations, obs)

	obs, err = newObservation(meter,
		strategiesMetric,
		"Number of instances per strategy type",
		"type",
		func(jaeger v1.Jaeger) string {
			return strings.ToLower(string(jaeger.Spec.Strategy))
		})
	if err != nil {
		return err
	}
	i.observations = append(i.observations, obs)

	obs, err = newObservation(meter,
		autoprovisioningMetric,
		"Number of instances using autoprovisioning",
		"type",
		func(jaeger v1.Jaeger) string {
			if v1.ShouldInjectOpenShiftElasticsearchConfiguration(jaeger.Spec.Storage) {
				return "elasticsearch"
			}
			return ""
		})
	if err != nil {
		return err
	}
	i.observations = append(i.observations, obs)

	obs, err = newObservation(meter,
		managedMetric,
		"Instances managed by other tool",
		"tool",
		func(jaeger v1.Jaeger) string {
			managed, hasManagement := jaeger.Labels[managedByLabel]
			if !hasManagement {
				return "none"
			}
			return managed
		})
	if err != nil {
		return err
	}
	i.observations = append(i.observations, obs)

	instruments := make([]instrument.Asynchronous, 0, len(i.observations))
	for _, o := range i.observations {
		instruments = append(instruments, o.Gauge)
	}
	_, err = meter.RegisterCallback(i.callback, instruments...)
	return err
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

func (i *instancesMetric) report(ctx context.Context, observer metric.Observer) {
	for _, o := range i.observations {
		o.Report(ctx, observer)
	}
}

func (i *instancesMetric) callback(ctx context.Context, observer metric.Observer) error {
	instances := &v1.JaegerList{}
	if err := i.client.List(ctx, instances); err == nil {
		i.reset()
		for k := range instances.Items {
			jaeger := instances.Items[k]
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
		i.report(ctx, observer)
	}
	return nil
}
