// Copyright The Jaeger Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package controllers

import (
	"context"

	"github.com/jaegertracing/jaeger-operator/internal/config"

	"github.com/go-logr/logr"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/jaegertracing/jaeger-operator/pkg/normalize"
	"github.com/jaegertracing/jaeger-operator/pkg/reconcilie"
	"github.com/jaegertracing/jaeger-operator/pkg/strategy"

	jaegertracingv2 "github.com/jaegertracing/jaeger-operator/apis/jaegertracing/v2"
)

// JaegerReconciler reconciles a Jaeger object.
type JaegerReconciler struct {
	client.Client
	Log    logr.Logger
	Scheme *runtime.Scheme
	Config config.Config
}

// +kubebuilder:rbac:groups=jaegertracing.io,resources=jaegers,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=jaegertracing.io,resources=jaegers/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=jaegertracing.io,resources=jaegers/finalizers,verbs=get;list;watch;create;update;patch
// +kubebuilder:rbac:groups=opentelemetry.io,resources=opentelemetrycollectors,verbs=get;list;watch;create;update;patch

// +kubebuilder:rbac:groups=coordination.k8s.io,resources=leases,verbs=get;list;create;update

func (r *JaegerReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	log := r.Log.WithValues("jaeger", req.NamespacedName)

	var instance jaegertracingv2.Jaeger
	if err := r.Get(ctx, req.NamespacedName, &instance); err != nil {
		if !apierrors.IsNotFound(err) {
			log.Error(err, "unable to fetch Jaeger instance")
		}

		// we'll ignore not-found errors, since they can't be fixed by an immediate
		// requeue (we'll need to wait for a new notification), and we can get them
		// on deleted requests.
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	instance = normalize.Jaeger(ctx, instance)

	params := reconcilie.Params{
		Config:   r.Config,
		Client:   r.Client,
		Instance: instance,
		Log:      log,
		Scheme:   r.Scheme,
		Strategy: strategy.For(ctx, r.Config, instance),
	}

	if err := reconcilie.Run(ctx, params); err != nil {
		return ctrl.Result{}, err
	}

	return ctrl.Result{}, nil
}

func (r *JaegerReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&jaegertracingv2.Jaeger{}).
		Complete(r)
}
