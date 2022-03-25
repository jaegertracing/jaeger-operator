package deployment

import (
	"context"
	"strconv"

	log "github.com/sirupsen/logrus"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	v1 "github.com/jaegertracing/jaeger-operator/apis/v1"
	"github.com/jaegertracing/jaeger-operator/pkg/inject"
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

	logger log.FieldLogger
}

// New creates new deployment controller
func New(client client.Client, clientReader client.Reader, scheme *runtime.Scheme) *ReconcileDeployment {
	return &ReconcileDeployment{
		client:  client,
		rClient: clientReader,
		scheme:  scheme,
		logger:  log.WithField("compoennt", "deployment-reconciler"),
	}
}

// Reconcile reads that state of the cluster for a Deployment object and makes changes based on the state read
// and what is in the Deployment.Spec
// Note:
// The Controller will requeue the Request to be processed again if the returned error is non-nil or
// Result.Requeue is true, otherwise upon completion it will remove the work from the queue.
func (r *ReconcileDeployment) Reconcile(ctx context.Context, request ctrl.Request) (ctrl.Result, error) {
	r.logger.Info("disabled")
	return reconcile.Result{}, nil
}

// SyncOnJaegerChanges sync deployments with sidecars when a jaeger CR changes
func (r *ReconcileDeployment) SyncOnJaegerChanges(object client.Object) []reconcile.Request {
	deps := []appsv1.Deployment{}
	nss := map[string]corev1.Namespace{} // namespace cache

	reconciliations := []reconcile.Request{}

	jaeger, ok := object.(*v1.Jaeger)
	if !ok {
		return reconciliations
	}

	deployments := appsv1.DeploymentList{}
	err := r.rClient.List(context.Background(), &deployments)
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
			revStr := "0"
			v := dep.Annotations[inject.AnnotationRev]
			if rev, err := strconv.Atoi(v); err == nil {
				revStr = strconv.Itoa(rev + 1)
			}
			dep.Annotations[inject.AnnotationRev] = revStr
			deps = append(deps, dep)
			continue
		}

		// if we don't have the namespace in the cache yet, retrieve it
		var ns corev1.Namespace
		if ns, ok = nss[dep.Namespace]; !ok {
			err := r.rClient.Get(context.Background(), types.NamespacedName{Name: dep.Namespace}, &ns)
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
	for _, dep := range deps {
		if err := r.client.Update(context.Background(), &dep); err != nil {
			r.logger.Error(err)
		}
	}
	return reconciliations
}
