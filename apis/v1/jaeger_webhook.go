package v1

import (
	"context"
	"encoding/json"
	"fmt"
	"regexp"

	esv1 "github.com/openshift/elasticsearch-operator/apis/logging/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/webhook"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"
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

//+kubebuilder:webhook:path=/mutate-jaegertracing-io-v1-jaeger,mutating=true,failurePolicy=fail,sideEffects=None,groups=jaegertracing.io,resources=jaegers,verbs=create;update,versions=v1,name=mjaeger.kb.io,admissionReviewVersions={v1}

var _ webhook.Defaulter = &Jaeger{}

func (j *Jaeger) objsWithOptions() []*Options {
	return []*Options{
		&j.Spec.AllInOne.Options, &j.Spec.Query.Options, &j.Spec.Collector.Options,
		&j.Spec.Ingester.Options, &j.Spec.Agent.Options, &j.Spec.Storage.Options,
	}
}

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

	for _, opt := range j.objsWithOptions() {
		optCopy := opt.DeepCopy()
		if f := getAdditionalTLSFlags(optCopy.ToArgs()); f != nil {
			newOpts := optCopy.GenericMap()
			for k, v := range f {
				newOpts[k] = v
			}

			if err := opt.parse(newOpts); err != nil {
				jaegerlog.Error(err, "name", j.Name, "method", "Option.Parse")
			}
		}
	}
}

// TODO(user): change verbs to "verbs=create;update;delete" if you want to enable deletion validation.
//+kubebuilder:webhook:path=/validate-jaegertracing-io-v1-jaeger,mutating=false,failurePolicy=fail,sideEffects=None,groups=jaegertracing.io,resources=jaegers,verbs=create;update,versions=v1,name=vjaeger.kb.io,admissionReviewVersions={v1}

var _ webhook.Validator = &Jaeger{}

// ValidateCreate implements webhook.Validator so a webhook will be registered for the type
func (j *Jaeger) ValidateCreate() (admission.Warnings, error) {
	jaegerlog.Info("validate create", "name", j.Name)
	return j.ValidateUpdate(nil)
}

// ValidateUpdate implements webhook.Validator so a webhook will be registered for the type
func (j *Jaeger) ValidateUpdate(_ runtime.Object) (admission.Warnings, error) {
	jaegerlog.Info("validate update", "name", j.Name)

	got, err := json.Marshal(j.Spec.Agent)
	if err != nil {
		return nil, err
	}
	want, _ := json.Marshal(JaegerAgentSpec{})
	if string(got) != string(want) {
		return nil, fmt.Errorf("Jaeger agent configuration is no longer supported. Please remove any agent configuration. For more details see https://github.com/jaegertracing/jaeger/issues/4739.")
	}

	if ShouldInjectOpenShiftElasticsearchConfiguration(j.Spec.Storage) && j.Spec.Storage.Elasticsearch.DoNotProvision {
		// check if ES instance exists
		es := &esv1.Elasticsearch{}
		err := cl.Get(context.Background(), types.NamespacedName{
			Namespace: j.Namespace,
			Name:      j.Spec.Storage.Elasticsearch.Name,
		}, es)
		if errors.IsNotFound(err) {
			return nil, fmt.Errorf("elasticsearch instance not found: %w", err)
		}
	}

	for _, opt := range j.objsWithOptions() {
		got := opt.DeepCopy().ToArgs()
		if f := getAdditionalTLSFlags(got); f != nil {
			return nil, fmt.Errorf("tls flags incomplete, got: %v", got)
		}
	}

	return nil, nil
}

// ValidateDelete implements webhook.Validator so a webhook will be registered for the type
func (j *Jaeger) ValidateDelete() (admission.Warnings, error) {
	jaegerlog.Info("validate delete", "name", j.Name)
	return nil, nil
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

var (
	tlsFlag          = regexp.MustCompile("--.*tls.*=")
	tlsFlagIdx       = regexp.MustCompile("--.*tls")
	tlsEnabledExists = regexp.MustCompile("--.*tls.enabled")
)

// getAdditionalTLSFlags returns additional tls arguments based on the argument
// list. If no additional argument is needed, nil is returned.
func getAdditionalTLSFlags(args []string) map[string]interface{} {
	var res map[string]interface{}
	for _, arg := range args {
		a := []byte(arg)
		if tlsEnabledExists.Match(a) {
			// NOTE: if flag exists, we are done.
			return nil
		}
		if tlsFlag.Match(a) && res == nil {
			idx := tlsFlagIdx.FindIndex(a)
			res = make(map[string]interface{})
			res[arg[idx[0]+2:idx[1]]+".enabled"] = "true"
		}
	}
	return res
}
