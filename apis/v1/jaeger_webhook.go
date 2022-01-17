package v1

import (
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/webhook"
)

const (
	defaultElasticsearchName = "elasticsearch"
)

// log is for logging in this package.
var jaegerlog = logf.Log.WithName("jaeger-resource")

// SetupWebhookWithManager adds Jaeger webook to the manager.
func (j *Jaeger) SetupWebhookWithManager(mgr ctrl.Manager) error {
	return ctrl.NewWebhookManagedBy(mgr).
		For(j).
		Complete()
}

//+kubebuilder:webhook:path=/mutate-jaegertracing-io-v1-jaeger,mutating=true,failurePolicy=fail,sideEffects=None,groups=jaegertracing.io,resources=jaegers,verbs=create;update,versions=v1,name=mjaeger.kb.io,admissionReviewVersions={v1,v1beta1}

var _ webhook.Defaulter = &Jaeger{}

// Default implements webhook.Defaulter so a webhook will be registered for the type
func (j *Jaeger) Default() {
	jaegerlog.Info("default", "name", j.Name)

	if j.Spec.Storage.Elasticsearch.Name == "" {
		j.Spec.Storage.Elasticsearch.Name = defaultElasticsearchName
	}
}

// TODO(user): change verbs to "verbs=create;update;delete" if you want to enable deletion validation.
//+kubebuilder:webhook:path=/validate-jaegertracing-io-v1-jaeger,mutating=false,failurePolicy=fail,sideEffects=None,groups=jaegertracing.io,resources=jaegers,verbs=create;update,versions=v1,name=vjaeger.kb.io,admissionReviewVersions={v1,v1beta1}

var _ webhook.Validator = &Jaeger{}

// ValidateCreate implements webhook.Validator so a webhook will be registered for the type
func (j *Jaeger) ValidateCreate() error {
	jaegerlog.Info("validate create", "name", j.Name)
	return j.ValidateUpdate(nil)
}

// ValidateUpdate implements webhook.Validator so a webhook will be registered for the type
func (j *Jaeger) ValidateUpdate(_ runtime.Object) error {
	jaegerlog.Info("validate update", "name", j.Name)
	return nil
}

// ValidateDelete implements webhook.Validator so a webhook will be registered for the type
func (j *Jaeger) ValidateDelete() error {
	jaegerlog.Info("validate delete", "name", j.Name)
	return nil
}
