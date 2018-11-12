package ingress

import (
	"fmt"
	"strings"

	"k8s.io/api/extensions/v1beta1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"

	"github.com/jaegertracing/jaeger-operator/pkg/apis/io/v1alpha1"
	"github.com/jaegertracing/jaeger-operator/pkg/service"
	"github.com/jaegertracing/jaeger-operator/pkg/util"
)

// QueryIngress builds pods for jaegertracing/jaeger-query
type QueryIngress struct {
	jaeger *v1alpha1.Jaeger
}

// NewQueryIngress builds a new QueryIngress struct based on the given spec
func NewQueryIngress(jaeger *v1alpha1.Jaeger) *QueryIngress {
	return &QueryIngress{jaeger: jaeger}
}

// Get returns an ingress specification for the current instance
func (i *QueryIngress) Get() *v1beta1.Ingress {
	if i.jaeger.Spec.Ingress.Enabled != nil && *i.jaeger.Spec.Ingress.Enabled == false {
		return nil
	}

	trueVar := true

	commonSpec := util.Merge([]v1alpha1.JaegerCommonSpec{i.jaeger.Spec.Ingress.JaegerCommonSpec, i.jaeger.Spec.JaegerCommonSpec})

	spec := v1beta1.IngressSpec{}
	backend := v1beta1.IngressBackend{
		ServiceName: service.GetNameForQueryService(i.jaeger),
		ServicePort: intstr.FromInt(service.GetPortForQueryService(i.jaeger)),
	}
	if _, ok := i.jaeger.Spec.AllInOne.Options.Map()["query.base-path"]; ok && strings.ToLower(i.jaeger.Spec.Strategy) == "allinone" {
		spec.Rules = append(spec.Rules, getRule(i.jaeger.Spec.AllInOne.Options, backend))
	} else if _, ok := i.jaeger.Spec.Query.Options.Map()["query.base-path"]; ok && strings.ToLower(i.jaeger.Spec.Strategy) == "production" {
		spec.Rules = append(spec.Rules, getRule(i.jaeger.Spec.Query.Options, backend))
	} else {
		spec.Backend = &backend
	}

	return &v1beta1.Ingress{
		TypeMeta: metav1.TypeMeta{
			Kind:       "Ingress",
			APIVersion: "extensions/v1beta1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      fmt.Sprintf("%s-query", i.jaeger.Name),
			Namespace: i.jaeger.Namespace,
			OwnerReferences: []metav1.OwnerReference{
				metav1.OwnerReference{
					APIVersion: i.jaeger.APIVersion,
					Kind:       i.jaeger.Kind,
					Name:       i.jaeger.Name,
					UID:        i.jaeger.UID,
					Controller: &trueVar,
				},
			},
			Annotations: commonSpec.Annotations,
		},
		Spec: spec,
	}
}

func getRule(options v1alpha1.Options, backend v1beta1.IngressBackend) v1beta1.IngressRule {
	rule := v1beta1.IngressRule{}
	rule.HTTP = &v1beta1.HTTPIngressRuleValue{
		Paths: []v1beta1.HTTPIngressPath{
			v1beta1.HTTPIngressPath{
				Path:    options.Map()["query.base-path"],
				Backend: backend,
			},
		},
	}
	return rule
}
