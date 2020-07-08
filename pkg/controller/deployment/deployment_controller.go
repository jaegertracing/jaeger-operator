package deployment

import (
	"context"

	log "github.com/sirupsen/logrus"
	"github.com/spf13/viper"
	"go.opentelemetry.io/otel/api/key"
	"go.opentelemetry.io/otel/global"
	"google.golang.org/grpc/codes"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	"sigs.k8s.io/controller-runtime/pkg/source"

	v1 "github.com/jaegertracing/jaeger-operator/pkg/apis/jaegertracing/v1"
	"github.com/jaegertracing/jaeger-operator/pkg/config/ca"
	"github.com/jaegertracing/jaeger-operator/pkg/inject"
	"github.com/jaegertracing/jaeger-operator/pkg/tracing"
)

// Add creates a new Deployment Controller and adds it to the Manager. The Manager will set fields on the Controller
// and Start it when the Manager is Started.
func Add(mgr manager.Manager) error {
	return add(mgr, newReconciler(mgr))
}

// newReconciler returns a new reconcile.Reconciler
func newReconciler(mgr manager.Manager) reconcile.Reconciler {
	return &ReconcileDeployment{client: mgr.GetClient(), scheme: mgr.GetScheme(), rClient: mgr.GetAPIReader()}
}

// add adds a new Controller to mgr with r as the reconcile.Reconciler
func add(mgr manager.Manager, r reconcile.Reconciler) error {
	// Create a new controller
	c, err := controller.New("deployment-controller", mgr, controller.Options{Reconciler: r})
	if err != nil {
		return err
	}

	// Watch for changes to primary resource Deployment
	err = c.Watch(&source.Kind{Type: &appsv1.Deployment{}}, &handler.EnqueueRequestForObject{})
	if err != nil {
		return err
	}

	return nil
}

var _ reconcile.Reconciler = &ReconcileDeployment{}

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

// Reconcile reads that state of the cluster for a Deployment object and makes changes based on the state read
// and what is in the Deployment.Spec
// Note:
// The Controller will requeue the Request to be processed again if the returned error is non-nil or
// Result.Requeue is true, otherwise upon completion it will remove the work from the queue.
func (r *ReconcileDeployment) Reconcile(request reconcile.Request) (reconcile.Result, error) {
	ctx := context.Background()

	tracer := global.TraceProvider().GetTracer(v1.ReconciliationTracer)
	ctx, span := tracer.Start(ctx, "reconcileDeployment")
	defer span.End()

	span.SetAttributes(key.String("name", request.Name), key.String("namespace", request.Namespace))
	log.WithFields(log.Fields{
		"namespace": request.Namespace,
		"name":      request.Name,
	}).Trace("Reconciling Deployment")

	// Fetch the Deployment instance
	dep := &appsv1.Deployment{}
	err := r.rClient.Get(ctx, request.NamespacedName, dep)
	if err != nil {
		if errors.IsNotFound(err) {
			// Request object not found, could have been deleted after reconcile request.
			// Owned objects are automatically garbage collected. For additional cleanup logic use finalizers.
			// Return and don't requeue
			span.SetStatus(codes.NotFound)
			return reconcile.Result{}, nil
		}
		// Error reading the object - requeue the request.
		return reconcile.Result{}, tracing.HandleError(err, span)
	}

	ns := &corev1.Namespace{}
	err = r.rClient.Get(ctx, types.NamespacedName{Name: request.Namespace}, ns)
	// we shouldn't fail if the namespace object can't be obtained
	if err != nil {
		log.WithField("namespace", request.Namespace).WithError(err).Trace("failed to get the namespace for the deployment, skipping injection based on namespace annotation")
		tracing.HandleError(err, span)
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

		jaeger := inject.Select(dep, ns, jaegers)
		if jaeger != nil && jaeger.GetDeletionTimestamp() == nil {
			if jaeger.Namespace != request.Namespace {
				log.WithFields(log.Fields{
					"jaeger-namespace": jaeger.Namespace,
					"app-namespace":    request.Namespace,
				}).Debug("different namespaces, so check whether trusted CA bundle configmap should be created")
				if cm := ca.GetTrustedCABundle(jaeger); cm != nil {
					// Update the namespace to be the same as the Deployment being injected
					cm.Namespace = request.Namespace
					jaeger.Logger().WithFields(log.Fields{
						"configMap": cm.Name,
						"namespace": cm.Namespace,
					}).Debug("creating Trusted CA bundle config maps")
					if err := r.client.Create(ctx, cm); err != nil && !errors.IsAlreadyExists(err) {
						log.WithField("namespace", request.Namespace).WithError(err).Error("failed to create trusted CA bundle")
						return reconcile.Result{}, tracing.HandleError(err, span)
					}
				}

				if cm := ca.GetServiceCABundle(jaeger); cm != nil {
					// Update the namespace to be the same as the Deployment being injected
					cm.Namespace = request.Namespace
					jaeger.Logger().WithFields(log.Fields{
						"configMap": cm.Name,
						"namespace": cm.Namespace,
					}).Debug("creating service CA config map")
					if err := r.client.Create(ctx, cm); err != nil && !errors.IsAlreadyExists(err) {
						log.WithField("namespace", request.Namespace).WithError(err).Error("failed to create trusted CA bundle")
						return reconcile.Result{}, tracing.HandleError(err, span)
					}
				}
			}

			// a suitable jaeger instance was found! let's inject a sidecar pointing to it then
			// Verified that jaeger instance was found and is not marked for deletion.
			log.WithFields(log.Fields{
				"deployment":       dep.Name,
				"namespace":        dep.Namespace,
				"jaeger":           jaeger.Name,
				"jaeger-namespace": jaeger.Namespace,
			}).Info("Injecting Jaeger Agent sidecar")

			injectedDep := inject.Sidecar(jaeger, dep.DeepCopy())

			if !inject.EqualSidecar(injectedDep, dep) {
				if err := r.client.Update(ctx, injectedDep); err != nil {
					log.WithField("deployment", injectedDep).WithError(err).Error("failed to update")
					return reconcile.Result{}, tracing.HandleError(err, span)
				}
			}

		} else {
			log.WithField("deployment", dep.Name).Info("No suitable Jaeger instances found to inject a sidecar")
		}
	}

	return reconcile.Result{}, nil
}
