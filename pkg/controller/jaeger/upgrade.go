package jaeger

import (
	"context"

	"go.opentelemetry.io/otel/global"

	v1 "github.com/jaegertracing/jaeger-operator/pkg/apis/jaegertracing/v1"
	"github.com/jaegertracing/jaeger-operator/pkg/tracing"
	"github.com/jaegertracing/jaeger-operator/pkg/upgrade"
	"github.com/jaegertracing/jaeger-operator/pkg/version"
)

func (r *ReconcileJaeger) applyUpgrades(ctx context.Context, jaeger v1.Jaeger) (v1.Jaeger, error) {
	tracer := global.TraceProvider().GetTracer(v1.ReconciliationTracer)
	ctx, span := tracer.Start(ctx, "applyUpgrades")
	defer span.End()

	currentVersions := version.Get()

	if len(jaeger.Status.Version) > 0 {
		if jaeger.Status.Version != currentVersions.Jaeger {
			// in theory, the version from the Status could be higher than currentVersions.Jaeger, but we let the upgrade routine
			// check/handle it
			upgraded, err := upgrade.ManagedInstance(ctx, r.client, jaeger, currentVersions.Jaeger)
			if err != nil {
				return jaeger, tracing.HandleError(err, span)
			}
			jaeger = upgraded
		}
	}

	// at this point, the Jaeger we are managing is in sync with the Operator's version
	// if this is a new object, no upgrade was made, so, we just set the version
	jaeger.Status.Version = currentVersions.Jaeger
	return jaeger, nil
}
