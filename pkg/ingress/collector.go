package ingress

import (
	"fmt"

	networkingv1 "k8s.io/api/networking/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	v1 "github.com/jaegertracing/jaeger-operator/apis/v1"
	"github.com/jaegertracing/jaeger-operator/pkg/service"
	"github.com/jaegertracing/jaeger-operator/pkg/util"
)

// CollectorIngress builds pods for jaegertracing/jaeger-collector
type CollectorIngress struct {
	jaeger *v1.Jaeger
}

// NewCollectorIngress builds a new CollectorIngress struct based on the given spec
func NewCollectorIngress(jaeger *v1.Jaeger) *CollectorIngress {
	return &CollectorIngress{jaeger: jaeger}
}

// Get returns an ingress specification for the current instance
func (i *CollectorIngress) Get() *networkingv1.Ingress {
	if i.jaeger.Spec.Collector.Ingress.Enabled != nil && !*i.jaeger.Spec.Collector.Ingress.Enabled {
		return nil
	}

	trueVar := true

	baseCommonSpec := v1.JaegerCommonSpec{
		Labels: util.Labels(fmt.Sprintf("%s-collector", i.jaeger.Name), "collector-ingress", *i.jaeger),
	}

	commonSpec := util.Merge([]v1.JaegerCommonSpec{i.jaeger.Spec.Collector.Ingress.JaegerCommonSpec, i.jaeger.Spec.JaegerCommonSpec, baseCommonSpec})

	spec := networkingv1.IngressSpec{}

	backend := networkingv1.IngressBackend{
		Service: &networkingv1.IngressServiceBackend{
			Name: service.GetNameForCollectorService(i.jaeger),
			Port: networkingv1.ServiceBackendPort{
				Number: int32(service.GetPortForCollectorService(i.jaeger)),
			},
		},
	}

	i.addRulesSpec(&spec, &backend)

	i.addTLSSpec(&spec)

	if i.jaeger.Spec.Collector.Ingress.IngressClassName != nil {
		spec.IngressClassName = i.jaeger.Spec.Collector.Ingress.IngressClassName
	}

	return &networkingv1.Ingress{
		TypeMeta: metav1.TypeMeta{
			Kind:       "Ingress",
			APIVersion: "networking.k8s.io/v1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      fmt.Sprintf("%s-collector", i.jaeger.Name),
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

func (i *CollectorIngress) addRulesSpec(spec *networkingv1.IngressSpec, backend *networkingv1.IngressBackend) {
	path := ""

	jaegerSpec := i.jaeger.Spec
	strategy := jaegerSpec.Strategy
	if allInOneCollectorBasePath, ok := jaegerSpec.AllInOne.Options.StringMap()["collector.base-path"]; ok && strategy == v1.DeploymentStrategyAllInOne {
		path = allInOneCollectorBasePath
	} else if collectorBasePath, ok := jaegerSpec.Collector.Options.StringMap()["collector.base-path"]; ok && strategy == v1.DeploymentStrategyProduction || strategy == v1.DeploymentStrategyStreaming {
		path = collectorBasePath
	}

	pathType := networkingv1.PathTypeImplementationSpecific
	if pt := i.jaeger.Spec.Collector.Ingress.PathType; pt != "" {
		pathType = networkingv1.PathType(pt)
	}
	if len(i.jaeger.Spec.Collector.Ingress.Hosts) > 0 || path != "" {
		spec.Rules = append(spec.Rules, getRules(path, &pathType, i.jaeger.Spec.Collector.Ingress.Hosts, backend)...)
	} else {
		// no hosts and no custom path -> fall back to a single service Ingress
		spec.DefaultBackend = backend
	}
}

func (i *CollectorIngress) addTLSSpec(spec *networkingv1.IngressSpec) {
	if len(i.jaeger.Spec.Collector.Ingress.TLS) > 0 {
		for _, tls := range i.jaeger.Spec.Collector.Ingress.TLS {
			spec.TLS = append(spec.TLS, networkingv1.IngressTLS{
				Hosts:      tls.Hosts,
				SecretName: tls.SecretName,
			})
		}
		if i.jaeger.Spec.Collector.Ingress.SecretName != "" {
			i.jaeger.Logger().V(1).Info(
				"Both 'ingress.secretName' and 'ingress.tls' are set. 'ingress.secretName' is deprecated and is therefore ignored.",
			)
		}
	} else if i.jaeger.Spec.Collector.Ingress.SecretName != "" {
		spec.TLS = append(spec.TLS, networkingv1.IngressTLS{
			SecretName: i.jaeger.Spec.Collector.Ingress.SecretName,
		})
		i.jaeger.Logger().V(1).Info(
			"'ingress.secretName' property is deprecated and will be removed in the future. Please use 'ingress.tls' instead.",
		)
	}
}
