package deployment

import (
	"context"

	log "github.com/sirupsen/logrus"
	"github.com/spf13/viper"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	v1 "github.com/jaegertracing/jaeger-operator/apis/v1"
	"github.com/jaegertracing/jaeger-operator/pkg/config/ca"
	"github.com/jaegertracing/jaeger-operator/pkg/inject"
	"github.com/jaegertracing/jaeger-operator/pkg/tracing"
)

// ReconcileDeployment reconciles a Deployment object
type ReconcileDeployment struct {
	// This client, initialized using mgr.Client() above, is a split client
	// that reads objects from the cache and writes to the apiserver
	client client.Client

	// this avoid the cache, which we need to bypass because the default client will attempt to place
	// a watch on Namespace at cluster scope, which isn't desirable to us...
	rClient client.Reader

	scheme *runtime.Scheme
}

// New creates new deployment controller
func New(client client.Client, clientReader client.Reader, scheme *runtime.Scheme) *ReconcileDeployment {
	return &ReconcileDeployment{
		client:  client,
		rClient: clientReader,
		scheme:  scheme,
	}
}

// Reconcile reads that state of the cluster for a Deployment object and makes changes based on the state read
// and what is in the Deployment.Spec
// Note:
// The Controller will requeue the Request to be processed again if the returned error is non-nil or
// Result.Requeue is true, otherwise upon completion it will remove the work from the queue.
func (r *ReconcileDeployment) Reconcile(ctx context.Context, request ctrl.Request) (ctrl.Result, error) {
	tracer := otel.GetTracerProvider().Tracer(v1.ReconciliationTracer)
	ctx, span := tracer.Start(ctx, "reconcileDeployment")
	span.SetAttributes(attribute.String("name", request.Name), attribute.String("namespace", request.Namespace))
	defer span.End()

	logger := log.WithFields(log.Fields{
		"namespace": request.Namespace,
		"name":      request.Name,
	})
	logger.Debug("Reconciling Deployment")

	// Fetch the Deployment instance
	dep := &appsv1.Deployment{}
	err := r.rClient.Get(ctx, request.NamespacedName, dep)
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

	if dep.Labels["app"] == "jaeger" && dep.Labels["app.kubernetes.io/component"] != "query" {
		// Don't touch jaeger deployments
		return reconcile.Result{}, nil
	}

	ns := &corev1.Namespace{}
	err = r.rClient.Get(ctx, types.NamespacedName{Name: request.Namespace}, ns)
	// we shouldn't fail if the namespace object can't be obtained
	if err != nil {
		msg := "failed to get the namespace for the deployment, skipping injection based on namespace annotation"
		logger.WithError(err).Debug(msg)
		span.AddEvent(msg, trace.WithAttributes(attribute.String("error", err.Error())))
	}

	jaegers := &v1.JaegerList{}
	opts := []client.ListOption{}

	if viper.GetString(v1.ConfigOperatorScope) == v1.OperatorScopeNamespace {
		opts = append(opts, client.InNamespace(viper.GetString(v1.ConfigWatchNamespace)))
	}

	if err := r.rClient.List(ctx, jaegers, opts...); err != nil {
		logger.WithError(err).Error("failed to get the available Jaeger pods")
		return reconcile.Result{}, tracing.HandleError(err, span)
	}

	if dep.Spec.Template.Annotations != nil {
		if owner, ok := dep.Spec.Template.Annotations[inject.AnnotationManagedBy]; ok && owner == dep.Name {
			if dep.Spec.Template.Annotations == nil {
				dep.Spec.Template.Annotations = make(map[string]string)
			}
			patch := client.MergeFrom(dep.DeepCopy())
			dep.Spec.Template.Annotations[inject.Annotation] = dep.Annotations[inject.Annotation]
			if err := r.client.Patch(ctx, dep, patch); err != nil {
				log.WithFields(log.Fields{
					"deploymentName":      dep.Name,
					"deploymentNamespace": dep.Namespace,
				}).WithError(err).Error("can not update pod annotations")
			}
		}
	}

	if inject.DeploymentNeeded(dep, ns) {
		jaeger := inject.Select(dep, ns, jaegers)
		if jaeger != nil && jaeger.GetDeletionTimestamp() == nil {
			logger := logger.WithFields(log.Fields{
				"jaeger":           jaeger.Name,
				"jaeger-namespace": jaeger.Namespace,
			})
			if jaeger.Namespace != dep.Namespace {
				if err := r.reconcileConfigMaps(ctx, jaeger, dep); err != nil {
					msg := "failed to reconcile config maps for the namespace"
					logger.WithError(err).Error(msg)
					span.AddEvent(msg)
				}
			}

			// a suitable jaeger instance was found! let's inject a sidecar pointing to it then
			// Verified that jaeger instance was found and is not marked for deletion.
			{
				msg := "injecting Jaeger Agent sidecar"
				logger.Info(msg)
				span.AddEvent(msg)
			}

			dep = inject.Sidecar(jaeger, dep)
			if err := r.client.Update(ctx, dep); err != nil {
				logger.WithError(err).Error("failed to update deployment with sidecar")
				return reconcile.Result{}, tracing.HandleError(err, span)
			}

		} else {
			msg := "no suitable Jaeger instances found to inject a sidecar"
			span.AddEvent(msg)
			logger.Debug(msg)
		}

	} else {
		hasAgent, _ := inject.HasJaegerAgent(dep.Spec.Template.Spec.Containers)
		if hasAgent {
			_, hasLabel := dep.Labels[inject.Label]
			if hasLabel {
				r.removeSidecar(ctx, dep)
			}

		}
		return reconcile.Result{}, nil
	}

	return reconcile.Result{}, nil
}

func (r *ReconcileDeployment) removeSidecar(ctx context.Context, dep *appsv1.Deployment) {
	jaegerInstance := dep.Labels[inject.Label]
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

// SyncOnJaegerChanges sync deployments with sidecars when a jaeger CR changes
func (r *ReconcileDeployment) SyncOnJaegerChanges(object client.Object) []reconcile.Request {
	reconciliations := []reconcile.Request{}
	nss := map[string]corev1.Namespace{} // namespace cache

	jaeger, ok := object.(*v1.Jaeger)
	if !ok {
		return reconciliations
	}

	deployments := appsv1.DeploymentList{}
	err := r.client.List(context.Background(), &deployments)
	if err != nil {
		return reconciliations
	}

	for _, dep := range deployments.Items {
		nsn := types.NamespacedName{Name: dep.Name, Namespace: dep.Namespace}
		req := reconcile.Request{NamespacedName: nsn}

		// if there's an assigned instance to this deployment, and it's not the one that triggered the current event,
		// we don't need to trigger a reconciliation for it
		if val, ok := dep.Labels[inject.Label]; ok && val != jaeger.Name {
			continue
		}

		// if the deployment has the sidecar annotation, trigger a reconciliation
		if _, ok := dep.Annotations[inject.Annotation]; ok {
			reconciliations = append(reconciliations, req)
			continue
		}

		// if we don't have the namespace in the cache yet, retrieve it
		var ns corev1.Namespace
		if ns, ok = nss[dep.Namespace]; !ok {
			err := r.client.Get(context.Background(), types.NamespacedName{Name: dep.Namespace}, &ns)
			if err != nil {
				continue
			}
			nss[ns.Name] = ns
		}

		// if the namespace has the sidecar annotation, trigger a reconciliation
		if _, ok := ns.Annotations[inject.Annotation]; ok {
			reconciliations = append(reconciliations, req)
			continue
		}

	}
	return reconciliations
}

func (r *ReconcileDeployment) reconcileConfigMaps(ctx context.Context, jaeger *v1.Jaeger, dep *appsv1.Deployment) error {
	tracer := otel.GetTracerProvider().Tracer(v1.ReconciliationTracer)
	ctx, span := tracer.Start(ctx, "reconcileConfigMaps")
	defer span.End()

	cms := []*corev1.ConfigMap{}
	if cm := ca.GetTrustedCABundle(jaeger); cm != nil {
		cms = append(cms, cm)
	}
	if cm := ca.GetServiceCABundle(jaeger); cm != nil {
		cms = append(cms, cm)
	}

	for _, cm := range cms {
		if err := r.reconcileConfigMap(ctx, cm, dep); err != nil {
			return tracing.HandleError(err, span)
		}
	}

	return nil
}

func (r *ReconcileDeployment) reconcileConfigMap(ctx context.Context, cm *corev1.ConfigMap, dep *appsv1.Deployment) error {
	tracer := otel.GetTracerProvider().Tracer(v1.ReconciliationTracer)
	ctx, span := tracer.Start(ctx, "reconcileConfigMap")
	defer span.End()

	// Update the namespace to be the same as the Deployment being injected
	cm.Namespace = dep.Namespace
	span.SetAttributes(attribute.String("name", cm.Name), attribute.String("namespace", cm.Namespace))

	if err := r.client.Create(ctx, cm); err != nil {
		if errors.IsAlreadyExists(err) {
			span.AddEvent("config map exists already")
		} else {
			return tracing.HandleError(err, span)
		}
	}

	return nil
}
