package namespace

import (
	"context"

	log "github.com/sirupsen/logrus"
	"github.com/spf13/viper"
	"go.opentelemetry.io/otel"
	otelattribute "go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	v1 "github.com/jaegertracing/jaeger-operator/apis/jaegertracing/v1"
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
	log.WithFields(log.Fields{
		"namespace": request.Namespace,
		"name":      request.Name,
	}).Trace("Reconciling Namespace")

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

		if inject.Needed(dep, ns) {
			jaegers := &v1.JaegerList{}
			opts := []client.ListOption{}

			if viper.GetString(v1.ConfigOperatorScope) == v1.OperatorScopeNamespace {
				opts = append(opts, client.InNamespace(viper.GetString(v1.ConfigWatchNamespace)))
			}

			if err := r.rClient.List(ctx, jaegers, opts...); err != nil {
				log.WithError(err).Error("failed to get the available Jaeger pods")
				return reconcile.Result{}, tracing.HandleError(err, span)
			}
			patch := client.MergeFrom(dep.DeepCopy())
			jaeger := inject.Select(dep, ns, jaegers)
			if jaeger != nil && jaeger.GetDeletionTimestamp() == nil {
				// a suitable jaeger instance was found! let's inject a sidecar pointing to it then
				// Verified that jaeger instance was found and is not marked for deletion.
				log.WithFields(log.Fields{
					"deployment":       dep.Name,
					"namespace":        dep.Namespace,
					"jaeger":           jaeger.Name,
					"jaeger-namespace": jaeger.Namespace,
				}).Info("Injecting Jaeger Agent sidecar")
				dep = inject.Sidecar(jaeger, dep)
				if err := r.client.Patch(ctx, dep, patch); err != nil {
					log.WithField("deployment", dep).WithError(err).Error("failed to update")
					return reconcile.Result{}, tracing.HandleError(err, span)
				}
			} else {
				log.WithField("deployment", dep.Name).Info("No suitable Jaeger instances found to inject a sidecar")
			}
		} else {
			// Don't need injection, may be need to remove the sidecar?
			// If deployment don't have the annotation and has an hasAgent, this may be injected by the namespace
			// we need to clean it.
			hasAgent, _ := inject.HasJaegerAgent(dep)
			_, hasDepAnnotation := dep.Annotations[inject.Annotation]
			if hasAgent && !hasDepAnnotation {
				jaegerInstance, hasLabel := dep.Labels[inject.Label]
				if hasLabel {
					log.WithFields(log.Fields{
						"deployment": dep.Name,
						"namespace":  dep.Namespace,
						"jaeger":     jaegerInstance,
					}).Info("Removing Jaeger Agent sidecar")
					patch := client.MergeFrom(dep.DeepCopy())
					inject.CleanSidecar(jaegerInstance, dep)
					if err := r.client.Patch(ctx, dep, patch); err != nil {
						log.WithFields(log.Fields{
							"deploymentName":      dep.Name,
							"deploymentNamespace": dep.Namespace,
						}).WithError(err).Error("error cleaning orphaned deployment")
					}
				}
			}
		}
	}

	return reconcile.Result{}, nil
}
