package jaeger

import (
	"context"
	"errors"
	"sync"
	"time"

	"go.opentelemetry.io/otel"
	appsv1 "k8s.io/api/apps/v1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/wait"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"

	v1 "github.com/jaegertracing/jaeger-operator/apis/v1"
	"github.com/jaegertracing/jaeger-operator/pkg/inventory"
	"github.com/jaegertracing/jaeger-operator/pkg/tracing"
)

// ErrDeploymentRemoved is returned when a deployment existed but has been removed
var ErrDeploymentRemoved = errors.New("deployment has been removed")

func (r *ReconcileJaeger) applyDeployments(ctx context.Context, jaeger v1.Jaeger, desired []appsv1.Deployment) error {
	tracer := otel.GetTracerProvider().Tracer(v1.ReconciliationTracer)
	ctx, span := tracer.Start(ctx, "applyDeployments")
	defer span.End()

	opts := []client.ListOption{
		client.InNamespace(jaeger.Namespace),
		client.MatchingLabels(map[string]string{
			"app.kubernetes.io/instance":   jaeger.Name,
			"app.kubernetes.io/managed-by": "jaeger-operator",
		}),
	}
	depList := &appsv1.DeploymentList{}
	if err := r.rClient.List(ctx, depList, opts...); err != nil {
		return tracing.HandleError(err, span)
	}

	// we now traverse the list, so that we end up with three lists:
	// 1) deployments that are on both `desired` and `existing` (update)
	// 2) deployments that are only on `desired` (create)
	// 3) deployments that are only on `existing` (delete)
	depInventory := inventory.ForDeployments(depList.Items, desired)
	for i := range depInventory.Create {
		d := depInventory.Create[i]
		jaeger.Logger().V(-1).Info(
			"creating deployment",
			"deployment", d.Name,
			"namespace", d.Namespace,
		)
		if err := r.client.Create(ctx, &d); err != nil {
			return tracing.HandleError(err, span)
		}
	}

	for i := range depInventory.Update {
		d := depInventory.Update[i]
		jaeger.Logger().V(-1).Info(
			"updating deployment",
			"deployment", d.Name,
			"namespace", d.Namespace,
		)
		if err := r.client.Update(ctx, &d); err != nil {
			return tracing.HandleError(err, span)
		}
	}

	// wait for the created and updated pods to stabilize, before we move on with
	// the removal of the old deployments
	for _, d := range depInventory.Create {
		if err := r.waitForStability(ctx, d); err != nil {
			return tracing.HandleError(err, span)
		}
	}
	for _, d := range depInventory.Update {
		if err := r.waitForStability(ctx, d); err != nil {
			return tracing.HandleError(err, span)
		}
	}

	for i := range depInventory.Delete {
		d := depInventory.Delete[i]
		jaeger.Logger().V(-1).V(-1).Info(
			"deleting deployment",
			"deployment", d.Name,
			"namespace", d.Namespace,
		)
		if err := r.client.Delete(ctx, &d); err != nil {
			return tracing.HandleError(err, span)
		}
	}

	return nil
}

func (r *ReconcileJaeger) waitForStability(ctx context.Context, dep appsv1.Deployment) error {
	tracer := otel.GetTracerProvider().Tracer(v1.ReconciliationTracer)
	ctx, span := tracer.Start(ctx, "waitForStability")
	defer span.End()

	// TODO: decide what's a good timeout... the first cold run might take a while to download
	// the images, subsequent runs should take only a few seconds

	seen := false
	once := &sync.Once{}
	return wait.PollImmediate(time.Second, 5*time.Minute, func() (done bool, err error) {
		d := &appsv1.Deployment{}
		if err := r.client.Get(ctx, types.NamespacedName{Name: dep.Name, Namespace: dep.Namespace}, d); err != nil {
			if k8serrors.IsNotFound(err) {
				if seen {
					// we have seen this object before, but it doesn't exist anymore!
					// we don't have anything else to do here, break the poll
					log.Log.V(1).Info(
						"Deployment has been removed.",
						"namespace", dep.Namespace,
						"name", dep.Name,
					)
					return true, ErrDeploymentRemoved
				}

				// the object might have not been created yet
				log.Log.V(-1).Info(
					"Deployment doesn't exist yet.",
					"namespace", dep.Namespace,
					"name", dep.Name,
				)
				return false, nil
			}
			return false, tracing.HandleError(err, span)
		}

		seen = true
		if d.Status.ReadyReplicas != d.Status.Replicas {
			once.Do(func() {
				log.Log.V(-1).Info(
					"Waiting for deployment to stabilize",
					"namespace", dep.Namespace,
					"name", dep.Name,
					"ready", d.Status.ReadyReplicas,
					"desired", d.Status.Replicas,
				)
			})
			return false, nil
		}

		log.Log.V(-1).Info(
			"Deployment has stabilized",
			"namespace", dep.Namespace,
			"name", dep.Name,
			"ready", d.Status.ReadyReplicas,
			"desired", d.Status.Replicas,
		)
		return true, nil
	})
}
