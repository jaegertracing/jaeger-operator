package namespace

import (
	"context"
	"fmt"

	"go.opentelemetry.io/otel"
	otelattribute "go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	v1 "github.com/jaegertracing/jaeger-operator/apis/v1"
	"github.com/jaegertracing/jaeger-operator/pkg/inject"
	"github.com/jaegertracing/jaeger-operator/pkg/tracing"
)

// ReconcileNamespace reconciles a Namespace object
type ReconcileNamespace struct {
	// This client, initialized using mgr.Client() above, is a split client
	// that reads objects from the cache and writes to the apiserver
	client client.Client

	// this avoid the cache, which we need to bypass because the default client will attempt to place
	// a watch on Namespace at cluster scope, which isn't desirable to us...
	rClient client.Reader

	scheme *runtime.Scheme
}

// New creates new namespace controller
func New(client client.Client, clientReader client.Reader, scheme *runtime.Scheme) *ReconcileNamespace {
	return &ReconcileNamespace{
		client:  client,
		rClient: clientReader,
		scheme:  scheme,
	}
}

// Reconcile reads that state of the cluster for a Namespace object and makes changes based on the state read
// and what is in the Namespace.Spec
// Note:
// The Controller will requeue the Request to be processed again if the returned error is non-nil or
// Result.Requeue is true, otherwise upon completion it will remove the work from the queue.
func (r *ReconcileNamespace) Reconcile(request reconcile.Request) (reconcile.Result, error) {
	ctx := context.Background()

	tracer := otel.GetTracerProvider().Tracer(v1.ReconciliationTracer)
	ctx, span := tracer.Start(ctx, "reconcileNamespace")
	defer span.End()

	span.SetAttributes(otelattribute.String("name", request.Name), otelattribute.String("namespace", request.Namespace))
	logger := log.Log.WithValues(
		"namespace", request.Namespace,
		"name", request.Name,
	)
	logger.V(-1).Info("Reconciling Namespace")

	ns := &corev1.Namespace{}
	err := r.rClient.Get(ctx, request.NamespacedName, ns)
	if err != nil {
		if errors.IsNotFound(err) {
			// Request object not found, could have been deleted after reconcile request.
			// Owned objects are automatically garbage collected. For additional cleanup logic use finalizers.
			// Return and don't requeue
			span.SetStatus(codes.Error, err.Error())
			return reconcile.Result{}, nil
		}
		// Error reading the object - requeue the request.
		return reconcile.Result{}, tracing.HandleError(err, span)
	}

	opts := []client.ListOption{
		client.InNamespace(request.Name),
	}

	// Fetch the Deployment instance
	deps := &appsv1.DeploymentList{}
	err = r.rClient.List(ctx, deps, opts...)
	if err != nil {
		if errors.IsNotFound(err) {
			// Request object not found, could have been deleted after reconcile request.
			// Owned objects are automatically garbage collected. For additional cleanup logic use finalizers.
			// Return and don't requeue
			return reconcile.Result{}, nil
		}
		// Error reading the object - requeue the request.
		return reconcile.Result{}, tracing.HandleError(err, span)
	}

	for i := 0; i < len(deps.Items); i++ {
		dep := &deps.Items[i]
		if dep.Labels["app"] == "jaeger" {
			// Don't touch jaeger deployments
			continue
		}

		// NOTE: If a deployment does not provide an "inject" annotation and
		// has an agent, we need to verify if this is caused by a annotated
		// namespace.
		hasAgent, _ := inject.HasJaegerAgent(dep)
		_, hasDepAnnotation := dep.Annotations[inject.Annotation]
		verificationNeeded := hasAgent && !hasDepAnnotation

		if inject.Needed(dep, ns) || verificationNeeded {
			inject.IncreaseRevision(dep.Annotations)
			if err := r.client.Update(context.Background(), dep); err != nil {
				logger.V(5).Info(fmt.Sprintf("%s", err))
				return reconcile.Result{}, tracing.HandleError(err, span)
			}
		}
	}
	return reconcile.Result{}, nil
}
