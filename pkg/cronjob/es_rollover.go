package cronjob

import (
	"fmt"
	"math/big"
	"strconv"
	"time"

	batchv1 "k8s.io/api/batch/v1"
	batchv1beta1 "k8s.io/api/batch/v1beta1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	v1 "github.com/jaegertracing/jaeger-operator/pkg/apis/jaegertracing/v1"
	"github.com/jaegertracing/jaeger-operator/pkg/util"
)

type durationUnits string

const (
	seconds durationUnits = "seconds"
	minutes durationUnits = "minutes"
	hours   durationUnits = "hours"
)

// CreateRollover returns objects which are necessary to run rolover actions for indices
func CreateRollover(jaeger *v1.Jaeger) []batchv1beta1.CronJob {
	return []batchv1beta1.CronJob{rollover(jaeger), lookback(jaeger)}
}

func rollover(jaeger *v1.Jaeger) batchv1beta1.CronJob {
	name := fmt.Sprintf("%s-es-rollover", jaeger.Name)
	envs := esScriptEnvVars(jaeger.Spec.Storage.Options)
	if jaeger.Spec.Storage.EsRollover.Conditions != "" {
		envs = append(envs, corev1.EnvVar{Name: "CONDITIONS", Value: jaeger.Spec.Storage.EsRollover.Conditions})
	}
	one := int32(1)
	return batchv1beta1.CronJob{
		ObjectMeta: metav1.ObjectMeta{
			Name:            name,
			Namespace:       jaeger.Namespace,
			Labels:          util.Labels(name, "cronjob-es-rollover", *jaeger),
			OwnerReferences: []metav1.OwnerReference{util.AsOwner(jaeger)},
		},
		Spec: batchv1beta1.CronJobSpec{
			ConcurrencyPolicy: batchv1beta1.ForbidConcurrent,
			Schedule:          jaeger.Spec.Storage.EsRollover.Schedule,
			JobTemplate: batchv1beta1.JobTemplateSpec{
				Spec: batchv1.JobSpec{
					Parallelism: &one,
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
									Args:  []string{"rollover", util.GetEsHostname(jaeger.Spec.Storage.Options.Map())},
									Env:   envs,
								},
							},
						},
					},
				},
			},
		},
	}
}

func lookback(jaeger *v1.Jaeger) batchv1beta1.CronJob {
	name := fmt.Sprintf("%s-es-lookback", jaeger.Name)
	envs := esScriptEnvVars(jaeger.Spec.Storage.Options)
	if jaeger.Spec.Storage.EsRollover.ReadTTL != "" {
		dur, err := time.ParseDuration(jaeger.Spec.Storage.EsRollover.ReadTTL)
		if err == nil {
			d := parseToUnits(dur)
			envs = append(envs, corev1.EnvVar{Name: "UNIT", Value: string(d.units)})
			envs = append(envs, corev1.EnvVar{Name: "UNIT_COUNT", Value: strconv.Itoa(d.count)})
		} else {
			jaeger.Logger().
				WithError(err).
				WithField("readTTL", jaeger.Spec.Storage.EsRollover.ReadTTL).
				Error("Failed to parse esRollover.readTTL to time.duration")
		}
	}
	return batchv1beta1.CronJob{
		ObjectMeta: metav1.ObjectMeta{
			Name:            name,
			Namespace:       jaeger.Namespace,
			Labels:          util.Labels(name, "cronjob-es-lookback", *jaeger),
			OwnerReferences: []metav1.OwnerReference{util.AsOwner(jaeger)},
		},
		Spec: batchv1beta1.CronJobSpec{
			ConcurrencyPolicy: batchv1beta1.ForbidConcurrent,
			Schedule:          jaeger.Spec.Storage.EsRollover.Schedule,
			JobTemplate: batchv1beta1.JobTemplateSpec{
				Spec: batchv1.JobSpec{
					TTLSecondsAfterFinished: jaeger.Spec.Storage.EsRollover.TTLSecondsAfterFinished,
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
									Args:  []string{"lookback", util.GetEsHostname(jaeger.Spec.Storage.Options.Map())},
									Env:   envs,
								},
							},
						},
					},
				},
			},
		},
	}
}

func esScriptEnvVars(opts v1.Options) []corev1.EnvVar {
	var envs []corev1.EnvVar
	if val, ok := opts.Map()["es.index-prefix"]; ok {
		envs = append(envs, corev1.EnvVar{Name: "INDEX_PREFIX", Value: val})
	}
	if val, ok := opts.Map()["es.username"]; ok {
		envs = append(envs, corev1.EnvVar{Name: "ES_USERNAME", Value: val})
	}
	if val, ok := opts.Map()["es.password"]; ok {
		envs = append(envs, corev1.EnvVar{Name: "ES_PASSWORD", Value: val})
	}
	return envs
}

type pythonUnits struct {
	units durationUnits
	count int
}

func parseToUnits(d time.Duration) pythonUnits {
	b := big.NewFloat(d.Hours())
	if big.NewFloat(d.Hours()).IsInt() {
		i, _ := b.Int64()
		return pythonUnits{units: hours, count: int(i)}
	}
	b = big.NewFloat(d.Minutes())
	if b.IsInt() {
		i, _ := b.Int64()
		return pythonUnits{units: minutes, count: int(i)}
	}
	b = big.NewFloat(d.Round(time.Second).Seconds())
	i, _ := b.Int64()
	return pythonUnits{units: seconds, count: int(i)}
}
