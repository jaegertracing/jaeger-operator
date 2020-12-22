package strategy

import (
	"context"
	"github.com/jaegertracing/jaeger-operator/apis/jaegertracing/v2"
	"github.com/jaegertracing/jaeger-operator/internal/instrument"
	"go.opentelemetry.io/otel"
)

// For returns the appropriate Strategy for the given Jaeger instance
func For(ctx context.Context, jaeger *v2.Jaeger) Strategy {
	tracer := otel.GetTracerProvider().Tracer(instrument.ReconciliationTracer)
	ctx, span := tracer.Start(ctx, "strategy.For")
	defer span.End()

	normalize(ctx, jaeger)

	if jaeger.Spec.Strategy == v2.DeploymentStrategyAllInOne {
		return newAllInOneStrategy(ctx, *jaeger)
	}

	return newProductionStrategy(ctx, *jaeger)
}

// normalize changes the incoming Jaeger object so that the defaults are applied when
// needed and incompatible options are cleaned
func normalize(ctx context.Context, jaeger *v2.Jaeger) {
	tracer := otel.GetTracerProvider().Tracer(instrument.ReconciliationTracer)
	ctx, span := tracer.Start(ctx, "normalize")
	defer span.End()

	// we need a name!
	if jaeger.Name == "" {
		//		jaeger.Logger().Info("This Jaeger instance was created without a name. Applying a default name.")
		jaeger.Name = "my-jaeger"
	}

	// normalize the deployment strategy
	if jaeger.Spec.Strategy != v2.DeploymentStrategyProduction && jaeger.Spec.Strategy != v2.DeploymentStrategyStreaming {
		jaeger.Spec.Strategy = v2.DeploymentStrategyAllInOne
	}

}
