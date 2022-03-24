package v1

import (
	"context"
	"fmt"

	esv1 "github.com/openshift/elasticsearch-operator/apis/logging/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/webhook"
)

const (
	defaultElasticsearchName = "elasticsearch"
)

// log is for logging in this package.
var (
	jaegerlog = logf.Log.WithName("jaeger-resource")
	cl        client.Client
)

// SetupWebhookWithManager adds Jaeger webook to the manager.
func (j *Jaeger) SetupWebhookWithManager(mgr ctrl.Manager) error {
	cl = mgr.GetClient()
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

	if ShouldInjectOpenShiftElasticsearchConfiguration(j.Spec.Storage) && j.Spec.Storage.Elasticsearch.DoNotProvision {
		// check if ES instance exists
		es := &esv1.Elasticsearch{}
		err := cl.Get(context.Background(), types.NamespacedName{
			Namespace: j.Namespace,
			Name:      j.Spec.Storage.Elasticsearch.Name,
		}, es)
		if errors.IsNotFound(err) {
			return
		}
		j.Spec.Storage.Elasticsearch.NodeCount = OpenShiftElasticsearchNodeCount(es.Spec)
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

	if ShouldInjectOpenShiftElasticsearchConfiguration(j.Spec.Storage) && j.Spec.Storage.Elasticsearch.DoNotProvision {
		// check if ES instance exists
		es := &esv1.Elasticsearch{}
		err := cl.Get(context.Background(), types.NamespacedName{
			Namespace: j.Namespace,
			Name:      j.Spec.Storage.Elasticsearch.Name,
		}, es)
		if errors.IsNotFound(err) {
			return fmt.Errorf("elasticsearch instance not found: %v", err)
		}
	}
	return nil
}

// ValidateDelete implements webhook.Validator so a webhook will be registered for the type
func (j *Jaeger) ValidateDelete() error {
	jaegerlog.Info("validate delete", "name", j.Name)
	return nil
}

// OpenShiftElasticsearchNodeCount returns total node count of Elasticsearch nodes.
func OpenShiftElasticsearchNodeCount(spec esv1.ElasticsearchSpec) int32 {
	nodes := int32(0)
	for i := 0; i < len(spec.Nodes); i++ {
		nodes += spec.Nodes[i].NodeCount
	}
	return nodes
}

// ShouldInjectOpenShiftElasticsearchConfiguration returns true if OpenShift Elasticsearch is used and its configuration should be used.
func ShouldInjectOpenShiftElasticsearchConfiguration(s JaegerStorageSpec) bool {
	if s.Type != JaegerESStorage {
		return false
	}
	_, ok := s.Options.Map()["es.server-urls"]
	return !ok
}
