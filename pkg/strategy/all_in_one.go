package strategy

import (
	"context"
	"github.com/jaegertracing/jaeger-operator/apis/jaegertracing/v2"
	"github.com/jaegertracing/jaeger-operator/internal/instrument"
	"go.opentelemetry.io/otel"
)

func newAllInOneStrategy(ctx context.Context, jaeger v2.Jaeger) Strategy {
	tracer := otel.GetTracerProvider().Tracer(instrument.ReconciliationTracer)
	ctx, span := tracer.Start(ctx, "newProductionStrategy")
	defer span.End()
	return Strategy{}
}
