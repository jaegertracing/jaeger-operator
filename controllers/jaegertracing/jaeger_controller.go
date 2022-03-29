/*
Copyright 2021.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package jaegertracing

import (
	"context"

	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/source"

	"github.com/operator-framework/operator-lib/handler"

	v1 "github.com/jaegertracing/jaeger-operator/apis/v1"
	"github.com/jaegertracing/jaeger-operator/pkg/controller/jaeger"
)

// JaegerReconciler reconciles a Jaeger object
type JaegerReconciler struct {
	reconcilier *jaeger.ReconcileJaeger
}

// NewReconciler creates a new jaeger reconcilier controller
func NewReconciler(client client.Client, clientReader client.Reader, scheme *runtime.Scheme) *JaegerReconciler {
	return &JaegerReconciler{
		reconcilier: jaeger.New(client, clientReader, scheme),
	}
}

// +kubebuilder:rbac:groups=jaegertracing.io,resources=jaegers,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=jaegertracing.io,resources=jaegers/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=jaegertracing.io,resources=jaegers/finalizers,verbs=update
// +kubebuilder:rbac:groups=core,resources=configmaps;persistentvolumeclaims;pods;secrets;serviceaccounts;services;services/finalizers,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=apps,resources=deployments;daemonsets;replicasets;statefulsets,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=extensions,resources=ingresses,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=autoscaling,resources=horizontalpodautoscalers,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=networking.k8s.io,resources=ingresses,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=route.openshift.io,resources=routes,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=console.openshift.io,resources=consolelinks,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=batch,resources=jobs;cronjobs,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=monitoring.coreos.com,resources=servicemonitors,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=logging.openshift.io,resources=elasticsearches,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=kafka.strimzi.io,resources=kafkas;kafkausers,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=rbac.authorization.k8s.io,resources=clusterrolebindings,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=coordination.k8s.io,resources=leases,verbs=get;list;create;update
// +kubebuilder:rbac:groups=image.openshift.io,resources=imagestreams,verbs=get;list;watch

// Reconcile jaeger resource
func (r *JaegerReconciler) Reconcile(ctx context.Context, request ctrl.Request) (ctrl.Result, error) {
	return r.reconcilier.Reconcile(request)
}

// SetupWithManager sets up the controller with the Manager.
func (r *JaegerReconciler) SetupWithManager(mgr ctrl.Manager) error {
	err := ctrl.NewControllerManagedBy(mgr).
		For(&v1.Jaeger{}).
		Watches(&source.Kind{Type: &v1.Jaeger{}}, &handler.InstrumentedEnqueueRequestForObject{}).
		Complete(r)
	return err
}
