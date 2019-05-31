package ingress

import (
	"fmt"
	"strings"

	extv1beta1 "k8s.io/api/extensions/v1beta1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"

	"github.com/jaegertracing/jaeger-operator/pkg/apis/jaegertracing/v1"
	"github.com/jaegertracing/jaeger-operator/pkg/service"
	"github.com/jaegertracing/jaeger-operator/pkg/util"
)

// QueryIngress builds pods for jaegertracing/jaeger-query
type QueryIngress struct {
	jaeger *v1.Jaeger
}

// NewQueryIngress builds a new QueryIngress struct based on the given spec
func NewQueryIngress(jaeger *v1.Jaeger) *QueryIngress {
	return &QueryIngress{jaeger: jaeger}
}

// Get returns an ingress specification for the current instance
func (i *QueryIngress) Get() *extv1beta1.Ingress {
	if i.jaeger.Spec.Ingress.Enabled != nil && *i.jaeger.Spec.Ingress.Enabled == false {
		return nil
	}

	trueVar := true

	baseCommonSpec := v1.JaegerCommonSpec{
		Labels: map[string]string{
			"app":                          "jaeger",
			"app.kubernetes.io/name":       fmt.Sprintf("%s-query", i.jaeger.Name),
			"app.kubernetes.io/instance":   i.jaeger.Name,
			"app.kubernetes.io/component":  "query-ingress",
			"app.kubernetes.io/part-of":    "jaeger",
			"app.kubernetes.io/managed-by": "jaeger-operator",
		},
	}

	commonSpec := util.Merge([]v1.JaegerCommonSpec{i.jaeger.Spec.Ingress.JaegerCommonSpec, i.jaeger.Spec.JaegerCommonSpec, baseCommonSpec})

	spec := extv1beta1.IngressSpec{}
	backend := extv1beta1.IngressBackend{
		ServiceName: service.GetNameForQueryService(i.jaeger),
		ServicePort: intstr.FromInt(service.GetPortForQueryService(i.jaeger)),
	}
	if _, ok := i.jaeger.Spec.AllInOne.Options.Map()["query.base-path"]; ok && strings.EqualFold(i.jaeger.Spec.Strategy, "allinone") {
		spec.Rules = append(spec.Rules, getRule(i.jaeger.Spec.AllInOne.Options, backend))
	} else if _, ok := i.jaeger.Spec.Query.Options.Map()["query.base-path"]; ok && strings.EqualFold(i.jaeger.Spec.Strategy, "production") {
		spec.Rules = append(spec.Rules, getRule(i.jaeger.Spec.Query.Options, backend))
	} else {
		spec.Backend = &backend
	}

	return &extv1beta1.Ingress{
		TypeMeta: metav1.TypeMeta{
			Kind:       "Ingress",
			APIVersion: "extensions/v1beta1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      fmt.Sprintf("%s-query", i.jaeger.Name),
			Namespace: i.jaeger.Namespace,
			Labels:    commonSpec.Labels,
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

func getRule(options v1.Options, backend extv1beta1.IngressBackend) extv1beta1.IngressRule {
	rule := extv1beta1.IngressRule{}
	rule.HTTP = &extv1beta1.HTTPIngressRuleValue{
		Paths: []extv1beta1.HTTPIngressPath{
			extv1beta1.HTTPIngressPath{
				Path:    options.Map()["query.base-path"],
				Backend: backend,
			},
		},
	}
	return rule
}
