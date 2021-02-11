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

package v2

import (
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/webhook"
)

// log is for logging in this package.
var jaegerlog = logf.Log.WithName("jaeger-resource")

func (r *Jaeger) SetupWebhookWithManager(mgr ctrl.Manager) error {
	return ctrl.NewWebhookManagedBy(mgr).
		For(r).
		Complete()
}

// +kubebuilder:webhook:path=/mutate-jaegertracing-io-v2-jaeger,mutating=true,failurePolicy=fail,groups=jaegertracing.io,resources=jaegers,verbs=create;update,versions=v2,name=mjaeger.kb.io

var _ webhook.Defaulter = &Jaeger{}

// Default implements webhook.Defaulter so a webhook will be registered for the type.
func (r *Jaeger) Default() {
	jaegerlog.Info("default", "name", r.Name)

	// TODO(user): fill in your defaulting logic.
}

// +kubebuilder:webhook:verbs=create;update;delete,path=/validate-jaegertracing-io-v2-jaeger,mutating=false,failurePolicy=fail,groups=jaegertracing.io,resources=jaegers,versions=v2,name=vjaeger.kb.io

var _ webhook.Validator = &Jaeger{}

// ValidateCreate implements webhook.Validator so a webhook will be registered for the type.
func (r *Jaeger) ValidateCreate() error {
	jaegerlog.Info("validate create", "name", r.Name)
	return nil
}

// ValidateUpdate implements webhook.Validator so a webhook will be registered for the type.
func (r *Jaeger) ValidateUpdate(old runtime.Object) error {
	jaegerlog.Info("validate update", "name", r.Name)
	return nil
}

// ValidateDelete implements webhook.Validator so a webhook will be registered for the type.
func (r *Jaeger) ValidateDelete() error {
	jaegerlog.Info("validate delete", "name", r.Name)
	return nil
}
