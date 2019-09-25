package cronjob

import (
	"fmt"
	"strconv"
	"strings"

	batchv1 "k8s.io/api/batch/v1"
	batchv1beta1 "k8s.io/api/batch/v1beta1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/jaegertracing/jaeger-operator/pkg/account"
	"github.com/jaegertracing/jaeger-operator/pkg/apis/jaegertracing/v1"
	"github.com/jaegertracing/jaeger-operator/pkg/util"
)

// CreateEsIndexCleaner returns a new cronjob for the Elasticsearch Index Cleaner operation

// CreateEsIndexCleaner returns a new cronjob for the Elasticsearch Index Cleaner operation
func CreateEsIndexCleaner(jaeger *v1.Jaeger) *batchv1beta1.CronJob {
	esUrls := util.GetEsHostname(jaeger.Spec.Storage.Options.Map())
	trueVar := true
	one := int32(1)
	name := fmt.Sprintf("%s-es-index-cleaner", jaeger.Name)

	var envFromSource []corev1.EnvFromSource
	if len(jaeger.Spec.Storage.SecretName) > 0 {
		envFromSource = append(envFromSource, corev1.EnvFromSource{
			SecretRef: &corev1.SecretEnvSource{
				LocalObjectReference: corev1.LocalObjectReference{
					Name: jaeger.Spec.Storage.SecretName,
				},
			},
		})
	}
	envs := esScriptEnvVars(jaeger.Spec.Storage.Options)
	if val, ok := jaeger.Spec.Storage.Options.Map()["es.use-aliases"]; ok && strings.EqualFold(val, "true") {
		envs = append(envs, corev1.EnvVar{Name: "ROLLOVER", Value: "true"})
	}

	baseCommonSpec := v1.JaegerCommonSpec{
		Annotations: map[string]string{
			"prometheus.io/scrape":    "false",
			"sidecar.istio.io/inject": "false",
			"linkerd.io/inject":       "disabled",
		},
	}

	commonSpec := util.Merge([]v1.JaegerCommonSpec{jaeger.Spec.Storage.EsIndexCleaner.JaegerCommonSpec, jaeger.Spec.JaegerCommonSpec, baseCommonSpec})

	return &batchv1beta1.CronJob{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: jaeger.Namespace,
			Labels: map[string]string{
				"app":                          "jaeger",
				"app.kubernetes.io/name":       name,
				"app.kubernetes.io/instance":   jaeger.Name,
				"app.kubernetes.io/component":  "cronjob-es-index-cleaner",
				"app.kubernetes.io/part-of":    "jaeger",
				"app.kubernetes.io/managed-by": "jaeger-operator",
			},
			OwnerReferences: []metav1.OwnerReference{
				{
					APIVersion: jaeger.APIVersion,
					Kind:       jaeger.Kind,
					Name:       jaeger.Name,
					UID:        jaeger.UID,
					Controller: &trueVar,
				},
			},
		},
		Spec: batchv1beta1.CronJobSpec{
			Schedule: jaeger.Spec.Storage.EsIndexCleaner.Schedule,
			JobTemplate: batchv1beta1.JobTemplateSpec{
				Spec: batchv1.JobSpec{
					Parallelism:             &one,
					TTLSecondsAfterFinished: jaeger.Spec.Storage.EsIndexCleaner.TTLSecondsAfterFinished,
					Template: corev1.PodTemplateSpec{
						Spec: corev1.PodSpec{
							Containers: []corev1.Container{
								{
									Name:         name,
									Image:        jaeger.Spec.Storage.EsIndexCleaner.Image,
									Args:         []string{strconv.Itoa(*jaeger.Spec.Storage.EsIndexCleaner.NumberOfDays), esUrls},
									Env:          envs,
									EnvFrom:      envFromSource,
									Resources:    commonSpec.Resources,
									VolumeMounts: jaeger.Spec.VolumeMounts,
								},
							},
							RestartPolicy:      corev1.RestartPolicyNever,
							Affinity:           commonSpec.Affinity,
							Tolerations:        commonSpec.Tolerations,
							SecurityContext:    commonSpec.SecurityContext,
							ServiceAccountName: account.JaegerServiceAccountFor(jaeger, account.EsIndexCleanerComponent),
							Volumes:            jaeger.Spec.Volumes,
						},
						ObjectMeta: metav1.ObjectMeta{
							Labels:      commonSpec.Labels,
							Annotations: commonSpec.Annotations,
						},
					},
				},
			},
		},
	}
}
