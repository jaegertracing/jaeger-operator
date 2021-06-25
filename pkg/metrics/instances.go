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
	"github.com/jaegertracing/jaeger-operator/pkg/tracing"
)

var groups = make(map[string]*instancesCounter)

// This structure contains the labels associated with the instances and a counter of the number of instances
// that have the set of labels e.g.:
//  Labels: [strategy=production, agent=sidecar, storage=es] , Count:2
//  Labels: [strategy=allinone, agent=sidecar, storage=memory] , Count:1
type instancesCounter struct {
	Labels []attribute.KeyValue
	Count  int64
}

type instancesMetric struct {
	client client.Client
	groups map[string]*instancesCounter
}

func newInstancesMetric(client client.Client) *instancesMetric {
	return &instancesMetric{
		client: client,
		groups: make(map[string]*instancesCounter), // for store the count of instances with different labels
	}
}

func (i *instancesMetric) Setup(ctx context.Context) error {
	tracer := otel.GetTracerProvider().Tracer(v1.CustomMetricsTracer)
	ctx, span := tracer.Start(ctx, "setup-jaeger-instances")
	defer span.End()
	meter := global.Meter(meterName)
	_, err := meter.NewInt64ValueObserver("operator_jaeger_instances", i.callback,
		metric.WithDescription("Number of jaeger instances in cluster"),
	)
	return tracing.HandleError(err, span)

}
func (i *instancesMetric) agentMode(jaeger v1.Jaeger) string {
	agent := string(jaeger.Spec.Agent.Strategy)
	if agent == "" {
		return "sidecar"
	}
	return agent
}
func (i *instancesMetric) storage(jaeger v1.Jaeger) string {
	storage := string(jaeger.Spec.Storage.Type)
	if storage == "" {
		storage = "memory"
	}
	return strings.ToLower(storage)

}
func (i *instancesMetric) strategy(jaeger v1.Jaeger) string {
	strategy := string(jaeger.Spec.Strategy)
	if strategy == "" {
		return "allinone"
	}
	return strings.ToLower(strategy)
}

func (i *instancesMetric) reset() {
	for _, g := range groups {
		g.Count = 0
	}
}

func (i *instancesMetric) callback(ctx context.Context, result metric.Int64ObserverResult) {
	instances := &v1.JaegerList{}
	for _, g := range groups {
		g.Count = 0
	}
	if err := i.client.List(ctx, instances); err == nil {
		for _, jaeger := range instances.Items {
			agent := i.agentMode(jaeger)
			strategy := i.strategy(jaeger)
			storage := i.storage(jaeger)
			key := fmt.Sprintf("%s_%s_%s", strategy, storage, agent)
			item, ok := groups[key]
			if !ok {
				groups[key] = &instancesCounter{
					Count: 1,
					Labels: []attribute.KeyValue{
						attribute.String("strategy", strategy),
						attribute.String("storage", storage),
						attribute.String("agent", strings.ToLower(agent)),
					},
				}
			} else {
				item.Count++
			}
		}
		for _, group := range groups {
			result.Observe(group.Count, group.Labels...)
		}
	}
}
