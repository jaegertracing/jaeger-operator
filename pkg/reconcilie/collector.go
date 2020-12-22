package reconcilie

import (
	"context"
	"fmt"
	"github.com/jaegertracing/jaeger-operator/pkg/collector"
	otelv1alpha1 "github.com/open-telemetry/opentelemetry-operator/api/v1alpha1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
)

func Collector(ctx context.Context, params Params) error {
	desired := collector.Get(params.Instance)

	// first, handle the create/update parts
	if err := controllerutil.SetControllerReference(&params.Instance, &desired, params.Scheme); err != nil {
		return fmt.Errorf("failed to set controller reference: %w", err)
	}

	existing := &otelv1alpha1.OpenTelemetryCollector{}
	nns := types.NamespacedName{Namespace: desired.Namespace, Name: desired.Name}
	err := params.Client.Get(ctx, nns, existing)

	if err != nil && k8serrors.IsNotFound(err) {
		if err := params.Client.Create(ctx, &desired); err != nil {
			return fmt.Errorf("failed to create: %w", err)
		}
		params.Log.V(2).Info("created", "collector", desired.Name, "collector.namespace", desired.Namespace)
	} else if err != nil {
		return fmt.Errorf("failed to get: %w", err)
	}

	updated := existing.DeepCopy()
	if updated.Annotations == nil {
		updated.Annotations = map[string]string{}
	}

	if updated.Labels == nil {
		updated.Labels = map[string]string{}
	}

	updated.Spec = desired.Spec
	updated.ObjectMeta.OwnerReferences = desired.ObjectMeta.OwnerReferences

	for k, v := range desired.ObjectMeta.Annotations {
		updated.ObjectMeta.Annotations[k] = v
	}

	for k, v := range desired.ObjectMeta.Labels {
		updated.ObjectMeta.Labels[k] = v
	}

	patch := client.MergeFrom(&params.Instance)
	if err := params.Client.Patch(ctx, updated, patch); err != nil {
		return fmt.Errorf("failed to apply changes: %w", err)
	}

	params.Log.V(2).Info("applied", "collector.name", desired.Name, "collector.namespace", desired.Namespace)
	return nil
}
