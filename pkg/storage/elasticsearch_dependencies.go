package storage

import (
	"fmt"
	"strings"

	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	v1 "github.com/jaegertracing/jaeger-operator/pkg/apis/jaegertracing/v1"
	"github.com/jaegertracing/jaeger-operator/pkg/util"
)

// EnableRollover returns true if rollover should be enabled
func EnableRollover(spec v1.JaegerStorageSpec) bool {
	useAliases := spec.Options.Map()["es.use-aliases"]
	return strings.EqualFold(spec.Type, "elasticsearch") && strings.EqualFold(useAliases, "true")
}

func elasticsearchDependencies(jaeger *v1.Jaeger) []batchv1.Job {
	name := fmt.Sprintf("%s-es-rollover-create-mapping", jaeger.Name)
	job := batchv1.Job{
		ObjectMeta: metav1.ObjectMeta{
			Name:            name,
			Namespace:       jaeger.Namespace,
			Labels:          util.Labels(name, "job-es-rollover-create-mapping", *jaeger),
			OwnerReferences: []metav1.OwnerReference{util.AsOwner(jaeger)},
		},
		Spec: batchv1.JobSpec{
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Annotations: map[string]string{
						"prometheus.io/scrape":    "false",
						"sidecar.istio.io/inject": "false",
						"linkerd.io/inject":       "disabled",
					},
				},
				Spec: corev1.PodSpec{
					RestartPolicy: corev1.RestartPolicyOnFailure,
					Containers: []corev1.Container{
						{
							Name:  name,
							Image: jaeger.Spec.Storage.EsRollover.Image,
							Args:  []string{"init", util.GetEsHostname(jaeger.Spec.Storage.Options.Map())},
							Env:   envVars(jaeger.Spec.Storage.Options),
						},
					},
				},
			},
		},
	}
	return []batchv1.Job{job}
}

func envVars(spec v1.Options) []corev1.EnvVar {
	var envs []corev1.EnvVar
	if val, ok := spec.Map()["es.index-prefix"]; ok {
		envs = append(envs, corev1.EnvVar{Name: "INDEX_PREFIX", Value: val})
	}
	if val, ok := spec.Map()["es.num-shards"]; ok {
		envs = append(envs, corev1.EnvVar{Name: "SHARDS", Value: val})
	}
	if val, ok := spec.Map()["es.num-replicas"]; ok {
		envs = append(envs, corev1.EnvVar{Name: "REPLICAS", Value: val})
	}
	if val, ok := spec.Map()["es.username"]; ok {
		envs = append(envs, corev1.EnvVar{Name: "ES_USERNAME", Value: val})
	}
	if val, ok := spec.Map()["es.password"]; ok {
		envs = append(envs, corev1.EnvVar{Name: "ES_PASSWORD", Value: val})
	}
	return envs
}
