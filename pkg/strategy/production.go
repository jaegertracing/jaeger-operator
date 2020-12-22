package strategy

import (
	"context"
	"github.com/jaegertracing/jaeger-operator/apis/jaegertracing/v2"
	"github.com/jaegertracing/jaeger-operator/internal/instrument"
	"github.com/jaegertracing/jaeger-operator/pkg/collector"
	"go.opentelemetry.io/otel"
)

func newProductionStrategy(ctx context.Context, jaeger v2.Jaeger) Strategy {
	tracer := otel.GetTracerProvider().Tracer(instrument.ReconciliationTracer)
	ctx, span := tracer.Start(ctx, "newProductionStrategy")
	defer span.End()
	strategy := Strategy{Type: v2.DeploymentStrategyProduction}
	strategy.Collector = collector.Get(jaeger)
	return strategy
}
