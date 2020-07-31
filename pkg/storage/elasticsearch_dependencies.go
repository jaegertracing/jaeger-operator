package storage

import (
	"strings"

	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	v1 "github.com/jaegertracing/jaeger-operator/pkg/apis/jaegertracing/v1"
	"github.com/jaegertracing/jaeger-operator/pkg/cronjob"
	"github.com/jaegertracing/jaeger-operator/pkg/util"
	"github.com/jaegertracing/jaeger-operator/pkg/account"
)

// EnableRollover returns true if rollover should be enabled
func EnableRollover(spec v1.JaegerStorageSpec) bool {
	useAliases := spec.Options.Map()["es.use-aliases"]
	return strings.EqualFold(spec.Type, "elasticsearch") && strings.EqualFold(useAliases, "true")
}

func elasticsearchDependencies(jaeger *v1.Jaeger) []batchv1.Job {
	name := util.Truncate("%s-es-rollover-create-mapping", 63, jaeger.Name)
	envFromSource := util.CreateEnvsFromSecret(jaeger.Spec.Storage.SecretName)
	commonSpec := &v1.JaegerCommonSpec{
		Annotations: map[string]string{
			"prometheus.io/scrape":    "false",
			"sidecar.istio.io/inject": "false",
			"linkerd.io/inject":       "disabled",
		},
		Labels: util.Labels(name, "job-es-rollover-create-mapping", *jaeger),
	}
	commonSpec = util.Merge([]v1.JaegerCommonSpec{jaeger.Spec.Storage.EsRollover.JaegerCommonSpec, jaeger.Spec.JaegerCommonSpec, *commonSpec})
	job := batchv1.Job{
		ObjectMeta: metav1.ObjectMeta{
			Name:            name,
			Namespace:       jaeger.Namespace,
			Labels:          commonSpec.Labels,
			OwnerReferences: []metav1.OwnerReference{util.AsOwner(jaeger)},
		},
		Spec: batchv1.JobSpec{
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Annotations: commonSpec.Annotations,
					Labels:      commonSpec.Labels,
				},
				Spec: corev1.PodSpec{
					RestartPolicy: corev1.RestartPolicyOnFailure,
					Affinity:           commonSpec.Affinity,
					Tolerations:        commonSpec.Tolerations,
					SecurityContext:    commonSpec.SecurityContext,
					ServiceAccountName: account.JaegerServiceAccountFor(jaeger, account.EsRolloverComponent),
					Volumes:       commonSpec.Volumes,
					Containers: []corev1.Container{
						{
							Name:         name,
							Image:        util.ImageName(jaeger.Spec.Storage.EsRollover.Image, "jaeger-es-rollover-image"),
							Args:         []string{"init", util.GetEsHostname(jaeger.Spec.Storage.Options.Map())},
							Env:          util.RemoveEmptyVars(envVars(jaeger.Spec.Storage.Options)),
							EnvFrom:      envFromSource,
							Resources:    commonSpec.Resources,
							VolumeMounts: commonSpec.VolumeMounts,
						},
					},
				},
			},
		},
	}
	return []batchv1.Job{job}
}

func envVars(opts v1.Options) []corev1.EnvVar {
	var envs = cronjob.EsScriptEnvVars(opts)
	scriptEnvVars := []struct {
		flag   string
		envVar string
	}{
		{flag: "es.num-shards", envVar: "SHARDS"},
		{flag: "es.num-replicas", envVar: "REPLICAS"},
	}
	options := opts.Map()
	for _, x := range scriptEnvVars {
		if val, ok := options[x.flag]; ok {
			envs = append(envs, corev1.EnvVar{Name: x.envVar, Value: val})
		}
	}
	return envs
}
