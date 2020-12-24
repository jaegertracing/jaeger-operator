package strategy

import (
	"context"
	"github.com/jaegertracing/jaeger-operator/apis/jaegertracing/v2"
	"github.com/jaegertracing/jaeger-operator/internal/instrument"
	"go.opentelemetry.io/otel"
)

// For returns the appropriate Strategy for the given Jaeger instance
func For(ctx context.Context, jaeger v2.Jaeger) Strategy {
	tracer := otel.GetTracerProvider().Tracer(instrument.ReconciliationTracer)
	ctx, span := tracer.Start(ctx, "strategy.For")
	defer span.End()

	if jaeger.Spec.Strategy == v2.DeploymentStrategyAllInOne {
		return newAllInOneStrategy(ctx, jaeger)
	}

	return newProductionStrategy(ctx, jaeger)
}
