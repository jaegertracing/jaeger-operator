package jaeger

import (
	"context"

	"go.opentelemetry.io/otel"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/selection"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"

	v1 "github.com/jaegertracing/jaeger-operator/apis/v1"
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
	for i := range inv.Create {
		d := inv.Create[i]
		jaeger.Logger().V(-1).Info(
			"creating config maps",
			"configMap", d.Name,
			"namespace", d.Namespace,
		)
		if err := r.client.Create(ctx, &d); err != nil {
			return tracing.HandleError(err, span)
		}
	}

	for i := range inv.Update {
		d := inv.Update[i]
		jaeger.Logger().V(-1).Info(
			"updating config maps",
			"configMap", d.Name,
			"namespace", d.Namespace,
		)
		if err := r.client.Update(ctx, &d); err != nil {
			return tracing.HandleError(err, span)
		}
	}

	for i := range inv.Delete {
		d := inv.Delete[i]
		jaeger.Logger().V(-1).Info(
			"deleting config maps",
			"configMap", d.Name,
			"namespace", d.Namespace,
		)
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

	for i := range configmaps.Items {
		cfgMap := configmaps.Items[i]
		if err := r.client.Delete(ctx, &cfgMap); err != nil {
			log.Log.Error(
				err,
				"error cleaning configmap deployment",
				"configMapName", cfgMap.Name,
				"configMapNamespace", cfgMap.Namespace,
			)
		}
	}
	return nil
}
