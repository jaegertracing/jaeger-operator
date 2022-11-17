package ingress

import (
	"fmt"

	networkingv1 "k8s.io/api/networking/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	v1 "github.com/jaegertracing/jaeger-operator/apis/v1"
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
func (i *QueryIngress) Get() *networkingv1.Ingress {
	if i.jaeger.Spec.Ingress.Enabled != nil && !*i.jaeger.Spec.Ingress.Enabled {
		return nil
	}

	trueVar := true

	baseCommonSpec := v1.JaegerCommonSpec{
		Labels: util.Labels(fmt.Sprintf("%s-query", i.jaeger.Name), "query-ingress", *i.jaeger),
	}

	commonSpec := util.Merge([]v1.JaegerCommonSpec{i.jaeger.Spec.Ingress.JaegerCommonSpec, i.jaeger.Spec.JaegerCommonSpec, baseCommonSpec})

	spec := networkingv1.IngressSpec{}

	backend := networkingv1.IngressBackend{
		Service: &networkingv1.IngressServiceBackend{
			Name: service.GetNameForQueryService(i.jaeger),
			Port: networkingv1.ServiceBackendPort{
				Number: int32(service.GetPortForQueryService(i.jaeger)),
			},
		},
	}

	i.addRulesSpec(&spec, &backend)

	i.addTLSSpec(&spec)

	if i.jaeger.Spec.Ingress.IngressClassName != nil {
		spec.IngressClassName = i.jaeger.Spec.Ingress.IngressClassName
	}

	return &networkingv1.Ingress{
		TypeMeta: metav1.TypeMeta{
			Kind:       "Ingress",
			APIVersion: "networking.k8s.io/v1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      fmt.Sprintf("%s-query", i.jaeger.Name),
			Namespace: i.jaeger.Namespace,
			Labels:    commonSpec.Labels,
			OwnerReferences: []metav1.OwnerReference{
				{
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

func (i *QueryIngress) addRulesSpec(spec *networkingv1.IngressSpec, backend *networkingv1.IngressBackend) {
	path := ""

	jaegerSpec := i.jaeger.Spec
	strategy := jaegerSpec.Strategy
	if allInOneQueryBasePath, ok := jaegerSpec.AllInOne.Options.StringMap()["query.base-path"]; ok && strategy == v1.DeploymentStrategyAllInOne {
		path = allInOneQueryBasePath
	} else if queryBasePath, ok := jaegerSpec.Query.Options.StringMap()["query.base-path"]; ok && strategy == v1.DeploymentStrategyProduction || strategy == v1.DeploymentStrategyStreaming {
		path = queryBasePath
	}

	pathType := networkingv1.PathTypeImplementationSpecific
	if pt := i.jaeger.Spec.Ingress.PathType; pt != "" {
		pathType = networkingv1.PathType(pt)
	}
	if len(i.jaeger.Spec.Ingress.Hosts) > 0 || path != "" {
		spec.Rules = append(spec.Rules, getRules(path, &pathType, i.jaeger.Spec.Ingress.Hosts, backend)...)
	} else {
		// no hosts and no custom path -> fall back to a single service Ingress
		spec.DefaultBackend = backend
	}
}

func (i *QueryIngress) addTLSSpec(spec *networkingv1.IngressSpec) {
	if len(i.jaeger.Spec.Ingress.TLS) > 0 {
		for _, tls := range i.jaeger.Spec.Ingress.TLS {
			spec.TLS = append(spec.TLS, networkingv1.IngressTLS{
				Hosts:      tls.Hosts,
				SecretName: tls.SecretName,
			})
		}
		if i.jaeger.Spec.Ingress.SecretName != "" {
			i.jaeger.Logger().V(1).Info(
				"Both 'ingress.secretName' and 'ingress.tls' are set. 'ingress.secretName' is deprecated and is therefore ignored.",
			)
		}
	} else if i.jaeger.Spec.Ingress.SecretName != "" {
		spec.TLS = append(spec.TLS, networkingv1.IngressTLS{
			SecretName: i.jaeger.Spec.Ingress.SecretName,
		})
		i.jaeger.Logger().V(1).Info(
			"'ingress.secretName' property is deprecated and will be removed in the future. Please use 'ingress.tls' instead.",
		)
	}
}

func getRules(path string, pathType *networkingv1.PathType, hosts []string, backend *networkingv1.IngressBackend) []networkingv1.IngressRule {
	if len(hosts) > 0 {
		rules := make([]networkingv1.IngressRule, len(hosts))
		for i, host := range hosts {
			rule := getRule(host, path, pathType, backend)
			rules[i] = rule
		}
		return rules
	}
	return []networkingv1.IngressRule{getRule("", path, pathType, backend)}
}

func getRule(host string, path string, pathType *networkingv1.PathType, backend *networkingv1.IngressBackend) networkingv1.IngressRule {
	rule := networkingv1.IngressRule{}
	rule.Host = host
	rule.HTTP = &networkingv1.HTTPIngressRuleValue{
		Paths: []networkingv1.HTTPIngressPath{
			{
				PathType: pathType,
				Path:     path,
				Backend:  *backend,
			},
		},
	}
	return rule
}
