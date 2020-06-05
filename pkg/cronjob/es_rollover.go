package cronjob

import (
	"math/big"
	"strconv"
	"time"

	batchv1 "k8s.io/api/batch/v1"
	batchv1beta1 "k8s.io/api/batch/v1beta1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/jaegertracing/jaeger-operator/pkg/account"
	v1 "github.com/jaegertracing/jaeger-operator/pkg/apis/jaegertracing/v1"
	"github.com/jaegertracing/jaeger-operator/pkg/config/ca"
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
	// CronJob names are restricted to 52 chars
	name := util.Truncate("%s-es-rollover", 52, jaeger.Name)
	envs := EsScriptEnvVars(jaeger.Spec.Storage.Options)
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
			ConcurrencyPolicy:          batchv1beta1.ForbidConcurrent,
			Schedule:                   jaeger.Spec.Storage.EsRollover.Schedule,
			SuccessfulJobsHistoryLimit: jaeger.Spec.Storage.EsRollover.SuccessfulJobsHistoryLimit,
			JobTemplate: batchv1beta1.JobTemplateSpec{
				Spec: batchv1.JobSpec{
					Parallelism: &one,
					Template:    *createTemplate(name, "rollover", jaeger, envs),
				},
			},
		},
	}
}

func createTemplate(name, action string, jaeger *v1.Jaeger, envs []corev1.EnvVar) *corev1.PodTemplateSpec {
	envFromSource := util.CreateEnvsFromSecret(jaeger.Spec.Storage.SecretName)
	baseCommonSpec := v1.JaegerCommonSpec{
		Annotations: map[string]string{
			"prometheus.io/scrape":    "false",
			"sidecar.istio.io/inject": "false",
			"linkerd.io/inject":       "disabled",
		},
	}

	commonSpec := util.Merge([]v1.JaegerCommonSpec{jaeger.Spec.Storage.EsRollover.JaegerCommonSpec, jaeger.Spec.JaegerCommonSpec, baseCommonSpec})

	ca.Update(jaeger, commonSpec)

	return &corev1.PodTemplateSpec{
		ObjectMeta: metav1.ObjectMeta{
			Labels:      commonSpec.Labels,
			Annotations: commonSpec.Annotations,
		},
		Spec: corev1.PodSpec{
			RestartPolicy:      corev1.RestartPolicyOnFailure,
			Affinity:           commonSpec.Affinity,
			Tolerations:        commonSpec.Tolerations,
			SecurityContext:    commonSpec.SecurityContext,
			ServiceAccountName: account.JaegerServiceAccountFor(jaeger, account.EsRolloverComponent),
			Volumes:            commonSpec.Volumes,
			Containers: []corev1.Container{
				{
					Name:         name,
					Image:        util.ImageName(jaeger.Spec.Storage.EsRollover.Image, "jaeger-es-rollover-image"),
					Args:         []string{action, util.GetEsHostname(jaeger.Spec.Storage.Options.Map())},
					Env:          util.RemoveEmptyVars(envs),
					EnvFrom:      envFromSource,
					Resources:    commonSpec.Resources,
					VolumeMounts: commonSpec.VolumeMounts,
				},
			},
		},
	}
}

func lookback(jaeger *v1.Jaeger) batchv1beta1.CronJob {
	// CronJob names are restricted to 52 chars
	name := util.Truncate("%s-es-lookback", 52, jaeger.Name)
	envs := EsScriptEnvVars(jaeger.Spec.Storage.Options)
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
			ConcurrencyPolicy:          batchv1beta1.ForbidConcurrent,
			Schedule:                   jaeger.Spec.Storage.EsRollover.Schedule,
			SuccessfulJobsHistoryLimit: jaeger.Spec.Storage.EsRollover.SuccessfulJobsHistoryLimit,
			JobTemplate: batchv1beta1.JobTemplateSpec{
				Spec: batchv1.JobSpec{
					TTLSecondsAfterFinished: jaeger.Spec.Storage.EsRollover.TTLSecondsAfterFinished,
					Template:                *createTemplate(name, "lookback", jaeger, envs),
				},
			},
		},
	}
}

// EsScriptEnvVars returns environmental variables for ES cron jobs.
func EsScriptEnvVars(opts v1.Options) []corev1.EnvVar {
	scriptEnvVars := []struct {
		flag   string
		envVar string
	}{
		{flag: "es.index-prefix", envVar: "INDEX_PREFIX"},
		{flag: "es.username", envVar: "ES_USERNAME"},
		{flag: "es.password", envVar: "ES_PASSWORD"},
		{flag: "es.tls", envVar: "ES_TLS"},
		{flag: "es.tls.ca", envVar: "ES_TLS_CA"},
		{flag: "es.tls.cert", envVar: "ES_TLS_CERT"},
		{flag: "es.tls.key", envVar: "ES_TLS_KEY"},
		{flag: "es.tls.skip-host-verify", envVar: "ES_TLS_SKIP_HOST_VERIFY"},
	}
	options := opts.Map()
	var envs []corev1.EnvVar
	for _, x := range scriptEnvVars {
		if val, ok := options[x.flag]; ok {
			envs = append(envs, corev1.EnvVar{Name: x.envVar, Value: val})
		}
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
