package ingress

import (
	networkingv1 "k8s.io/api/networking/v1"
)

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
