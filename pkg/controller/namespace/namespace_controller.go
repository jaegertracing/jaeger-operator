package namespace

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
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	"sigs.k8s.io/controller-runtime/pkg/source"

	v1 "github.com/jaegertracing/jaeger-operator/pkg/apis/jaegertracing/v1"
	"github.com/jaegertracing/jaeger-operator/pkg/inject"
	"github.com/jaegertracing/jaeger-operator/pkg/tracing"
)

// Add creates a new Namespace Controller and adds it to the Manager. The Manager will set fields on the Controller
// and Start it when the Manager is Started.
func Add(mgr manager.Manager) error {
	return add(mgr, newReconciler(mgr))
}

// newReconciler returns a new reconcile.Reconciler
func newReconciler(mgr manager.Manager) reconcile.Reconciler {
	return &ReconcileNamespace{client: mgr.GetClient(), scheme: mgr.GetScheme(), rClient: mgr.GetAPIReader()}
}

// add adds a new Controller to mgr with r as the reconcile.Reconciler
func add(mgr manager.Manager, r reconcile.Reconciler) error {
	// we only create this controller if we have cluster-scope, as watching namespaces is a cluster-wide operation
	if !viper.GetBool(v1.ConfigEnableNamespaceController) {
		log.Trace("skipping reconciliation for namespaces, do not have permissions to list and watch namespaces")
		return nil
	}

	// Create a new controller
	c, err := controller.New("namespace-controller", mgr, controller.Options{Reconciler: r})
	if err != nil {
		return err
	}

	err = c.Watch(&source.Kind{Type: &corev1.Namespace{}}, &handler.EnqueueRequestForObject{})
	if err != nil {
		return err
	}

	return nil
}

var _ reconcile.Reconciler = &ReconcileNamespace{}

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

// Reconcile reads that state of the cluster for a Namespace object and makes changes based on the state read
// and what is in the Namespace.Spec
// Note:
// The Controller will requeue the Request to be processed again if the returned error is non-nil or
// Result.Requeue is true, otherwise upon completion it will remove the work from the queue.
func (r *ReconcileNamespace) Reconcile(request reconcile.Request) (reconcile.Result, error) {
	ctx := context.Background()

	tracer := global.TraceProvider().GetTracer(v1.ReconciliationTracer)
	ctx, span := tracer.Start(ctx, "reconcileNamespace")
	defer span.End()

	span.SetAttributes(key.String("name", request.Name), key.String("namespace", request.Namespace))
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
			span.SetStatus(codes.NotFound)
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
				// a suitable jaeger instance was found! let's inject a sidecar pointing to it then
				// Verified that jaeger instance was found and is not marked for deletion.
				log.WithFields(log.Fields{
					"deployment":       dep.Name,
					"namespace":        dep.Namespace,
					"jaeger":           jaeger.Name,
					"jaeger-namespace": jaeger.Namespace,
				}).Info("Injecting Jaeger Agent sidecar")
				dep = inject.Sidecar(jaeger, dep)
				if err := r.client.Update(ctx, dep); err != nil {
					log.WithField("deployment", dep).WithError(err).Error("failed to update")
					return reconcile.Result{}, tracing.HandleError(err, span)
				}
			} else {
				log.WithField("deployment", dep.Name).Info("No suitable Jaeger instances found to inject a sidecar")
			}
		}
	}

	return reconcile.Result{}, nil
}
