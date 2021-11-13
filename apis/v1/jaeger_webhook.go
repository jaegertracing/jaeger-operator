package v1

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

// EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!

//+kubebuilder:webhook:path=/mutate-jaegertracing-io-v1-jaeger,mutating=true,failurePolicy=fail,sideEffects=None,groups=jaegertracing.io,resources=jaegers,verbs=create;update,versions=v1,name=mjaeger.kb.io,admissionReviewVersions={v1,v1beta1}

var _ webhook.Defaulter = &Jaeger{}

// Default implements webhook.Defaulter so a webhook will be registered for the type
func (r *Jaeger) Default() {
	jaegerlog.Info("default", "name", r.Name)

	// TODO(user): fill in your defaulting logic.
}

// TODO(user): change verbs to "verbs=create;update;delete" if you want to enable deletion validation.
//+kubebuilder:webhook:path=/validate-jaegertracing-io-v1-jaeger,mutating=false,failurePolicy=fail,sideEffects=None,groups=jaegertracing.io,resources=jaegers,verbs=create;update,versions=v1,name=vjaeger.kb.io,admissionReviewVersions={v1,v1beta1}

var _ webhook.Validator = &Jaeger{}

// ValidateCreate implements webhook.Validator so a webhook will be registered for the type
func (r *Jaeger) ValidateCreate() error {
	jaegerlog.Info("validate create", "name", r.Name)

	// TODO(user): fill in your validation logic upon object creation.
	return nil
}

// ValidateUpdate implements webhook.Validator so a webhook will be registered for the type
func (r *Jaeger) ValidateUpdate(old runtime.Object) error {
	jaegerlog.Info("validate update", "name", r.Name)

	// TODO(user): fill in your validation logic upon object update.
	return nil
}

// ValidateDelete implements webhook.Validator so a webhook will be registered for the type
func (r *Jaeger) ValidateDelete() error {
	jaegerlog.Info("validate delete", "name", r.Name)

	// TODO(user): fill in your validation logic upon object deletion.
	return nil
}
