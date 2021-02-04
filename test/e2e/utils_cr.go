package e2e

import (
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	v1 "github.com/jaegertracing/jaeger-operator/pkg/apis/jaegertracing/v1"
)

// GetJaegerSimpleProdWithServerUrlsCR returns simple production CR with external es server urls
func GetJaegerSimpleProdWithServerUrlsCR(name, esServerUrls string) *v1.Jaeger {
	ingressEnabled := true
	simpleProdCR := &v1.Jaeger{
		TypeMeta: metav1.TypeMeta{
			Kind:       "Jaeger",
			APIVersion: "jaegertracing.io/v1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
		},
		Spec: v1.JaegerSpec{
			Ingress: v1.JaegerIngressSpec{
				Enabled:  &ingressEnabled,
				Security: v1.IngressSecurityNoneExplicit,
			},
			Strategy: v1.DeploymentStrategyProduction,
			Storage: v1.JaegerStorageSpec{
				Type: v1.JaegerESStorage,
				Options: v1.NewOptions(map[string]interface{}{
					"es.server-urls": esServerUrls,
				}),
			},
		},
	}

	return simpleProdCR
}

// GetJaegerSelfProvSimpleProdCR returns self provisioned production simple CR
func GetJaegerSelfProvSimpleProdCR(instanceName, namespace string, nodeCount int32) *v1.Jaeger {
	ingressEnabled := true
	selfProvSimpleProdCR := &v1.Jaeger{
		TypeMeta: metav1.TypeMeta{
			Kind:       "Jaeger",
			APIVersion: "jaegertracing.io/v1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      instanceName,
			Namespace: namespace,
		},
		Spec: v1.JaegerSpec{
			Ingress: v1.JaegerIngressSpec{
				Enabled:  &ingressEnabled,
				Security: v1.IngressSecurityNoneExplicit,
			},
			Strategy: v1.DeploymentStrategyProduction,
			Storage: v1.JaegerStorageSpec{
				Type: v1.JaegerESStorage,
				Elasticsearch: v1.ElasticsearchSpec{
					NodeCount: nodeCount,
					Resources: &corev1.ResourceRequirements{
						Limits:   corev1.ResourceList{corev1.ResourceMemory: resource.MustParse("2Gi")},
						Requests: corev1.ResourceList{corev1.ResourceMemory: resource.MustParse("1Gi")},
					},
				},
			},
		},
	}

	return selfProvSimpleProdCR
}
