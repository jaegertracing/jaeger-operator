package normalize

import (
	"context"
	"github.com/jaegertracing/jaeger-operator/apis/jaegertracing/v2"
	jaegertracingv2 "github.com/jaegertracing/jaeger-operator/apis/jaegertracing/v2"
	"github.com/jaegertracing/jaeger-operator/internal/instrument"
	"go.opentelemetry.io/otel"
)

// normalize changes the incoming Jaeger object so that the defaults are applied when
// needed and incompatible options are cleaned
func Jaeger(ctx context.Context, jaeger jaegertracingv2.Jaeger) jaegertracingv2.Jaeger {
	tracer := otel.GetTracerProvider().Tracer(instrument.ReconciliationTracer)
	ctx, span := tracer.Start(ctx, "normalize")
	defer span.End()

	// we need a name!
	if jaeger.Name == "" {
		//		jaeger.Logger().Info("This Jaeger instance was created without a name. Applying a default name.")
		jaeger.Name = "my-jaeger"
	}

	// normalize the deployment strategy
	if jaeger.Spec.Strategy != jaegertracingv2.DeploymentStrategyProduction && jaeger.Spec.Strategy != v2.DeploymentStrategyStreaming {
		jaeger.Spec.Strategy = jaegertracingv2.DeploymentStrategyAllInOne
	}

	return jaeger
}
