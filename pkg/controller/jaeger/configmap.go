package jaeger

import (
	"context"

	log "github.com/sirupsen/logrus"
	"go.opentelemetry.io/otel"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/selection"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/jaegertracing/jaeger-operator/pkg/apis/jaegertracing/v1"
	"github.com/jaegertracing/jaeger-operator/pkg/inventory"
	"github.com/jaegertracing/jaeger-operator/pkg/tracing"
	"github.com/jaegertracing/jaeger-operator/pkg/util"
)

func (r *ReconcileJaeger) applyConfigMaps(ctx context.Context, jaeger v1.Jaeger, desired []corev1.ConfigMap) error {
	tracer := otel.GetTracerProvider().Tracer(v1.ReconciliationTracer)
	ctx, span := tracer.Start(ctx, "applyConfigMaps")
	defer span.End()

	opts := []client.ListOption{
		client.InNamespace(jaeger.Namespace),
		client.MatchingLabels(map[string]string{
			"app.kubernetes.io/instance":   jaeger.Name,
			"app.kubernetes.io/managed-by": "jaeger-operator",
		}),
	}
	list := &corev1.ConfigMapList{}
	if err := r.rClient.List(ctx, list, opts...); err != nil {
		return tracing.HandleError(err, span)
	}

	inv := inventory.ForConfigMaps(list.Items, desired)
	for _, d := range inv.Create {
		jaeger.Logger().WithFields(log.Fields{
			"configMap": d.Name,
			"namespace": d.Namespace,
		}).Debug("creating config maps")
		if err := r.client.Create(ctx, &d); err != nil {
			return tracing.HandleError(err, span)
		}
	}

	for _, d := range inv.Update {
		jaeger.Logger().WithFields(log.Fields{
			"configMap": d.Name,
			"namespace": d.Namespace,
		}).Debug("updating config maps")
		if err := r.client.Update(ctx, &d); err != nil {
			return tracing.HandleError(err, span)
		}
	}

	for _, d := range inv.Delete {
		jaeger.Logger().WithFields(log.Fields{
			"configMap": d.Name,
			"namespace": d.Namespace,
		}).Debug("deleting config maps")
		if err := r.client.Delete(ctx, &d); err != nil {
			return tracing.HandleError(err, span)
		}
	}

	return nil
}

func (r *ReconcileJaeger) getSelectorsForConfigMaps(instanceName string, components []string) labels.Selector {
	// We are swallowing the errors here because we are sure that the label requirements are valid
	// e.g. "DoubleEqual" has one single value, "In" has at least 1 value, etc... so don't require to check for errors
	nameReq, _ := labels.NewRequirement("app.kubernetes.io/name", selection.DoubleEquals, []string{util.Truncate(instanceName, 63)})
	componentReq, _ := labels.NewRequirement("app.kubernetes.io/component", selection.In, components)
	managerReq, _ := labels.NewRequirement("app.kubernetes.io/managed-by", selection.Equals, []string{"jaeger-operator"})
	selector := labels.Everything()
	return selector.Add(*nameReq, *componentReq, *managerReq)
}

func (r *ReconcileJaeger) cleanConfigMaps(ctx context.Context, instanceName string) error {
	configmaps := corev1.ConfigMapList{}
	if err := r.rClient.List(ctx, &configmaps, &client.ListOptions{
		LabelSelector: r.getSelectorsForConfigMaps(instanceName, []string{"ca-configmap", "service-ca-configmap"}),
	}); err != nil {
		return err
	}

	for _, cfgMap := range configmaps.Items {
		if err := r.client.Delete(ctx, &cfgMap); err != nil {
			log.WithFields(log.Fields{
				"configMapName":      cfgMap.Name,
				"configMapNamespace": cfgMap.Namespace,
			}).WithError(err).Error("error cleaning configmap deployment")
		}
	}
	return nil
}
