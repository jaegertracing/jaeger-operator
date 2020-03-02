package jaeger

import (
	"context"

	log "github.com/sirupsen/logrus"
	"go.opentelemetry.io/otel/global"
	corev1 "k8s.io/api/core/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"

	v1 "github.com/jaegertracing/jaeger-operator/pkg/apis/jaegertracing/v1"
	"github.com/jaegertracing/jaeger-operator/pkg/inventory"
	"github.com/jaegertracing/jaeger-operator/pkg/tracing"
)

func (r *ReconcileJaeger) applySecrets(ctx context.Context, jaeger v1.Jaeger, desired []corev1.Secret) error {
	tracer := global.TraceProvider().GetTracer(v1.ReconciliationTracer)
	ctx, span := tracer.Start(ctx, "applySecrets")
	defer span.End()

	opts := []client.ListOption{
		client.InNamespace(jaeger.Namespace),
		client.MatchingLabels(map[string]string{
			"app.kubernetes.io/instance":   jaeger.Name,
			"app.kubernetes.io/managed-by": "jaeger-operator",
		}),
	}
	list := &corev1.SecretList{}
	if err := r.rClient.List(ctx, list, opts...); err != nil {
		return tracing.HandleError(err, span)
	}

	inv := inventory.ForSecrets(list.Items, desired)
	for _, d := range inv.Create {
		jaeger.Logger().WithFields(log.Fields{
			"secret":    d.Name,
			"namespace": d.Namespace,
		}).Debug("creating secrets")
		if err := r.client.Create(ctx, &d); err != nil {
			return tracing.HandleError(err, span)
		}
	}

	for _, d := range inv.Update {
		jaeger.Logger().WithFields(log.Fields{
			"secret":    d.Name,
			"namespace": d.Namespace,
		}).Debug("updating secrets")
		if err := r.client.Update(ctx, &d); err != nil {
			return tracing.HandleError(err, span)
		}
	}

	for _, d := range inv.Delete {
		jaeger.Logger().WithFields(log.Fields{
			"secret":    d.Name,
			"namespace": d.Namespace,
		}).Debug("deleting secrets")
		if err := r.client.Delete(ctx, &d); err != nil {
			return tracing.HandleError(err, span)
		}
	}

	return nil
}
