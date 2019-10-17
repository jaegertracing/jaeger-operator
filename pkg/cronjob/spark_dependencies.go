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
	v1 "github.com/jaegertracing/jaeger-operator/pkg/apis/jaegertracing/v1"
	"github.com/jaegertracing/jaeger-operator/pkg/storage"
	"github.com/jaegertracing/jaeger-operator/pkg/util"
)

var supportedStorageTypes = map[string]bool{"elasticsearch": true, "cassandra": true}

// SupportedStorage returns whether the given storage is supported
func SupportedStorage(storage string) bool {
	return supportedStorageTypes[strings.ToLower(storage)]
}

// CreateSparkDependencies creates a new cronjob for the Spark Dependencies task
func CreateSparkDependencies(jaeger *v1.Jaeger) *batchv1beta1.CronJob {
	envVars := []corev1.EnvVar{
		{Name: "STORAGE", Value: jaeger.Spec.Storage.Type},
		{Name: "SPARK_MASTER", Value: jaeger.Spec.Storage.Dependencies.SparkMaster},
		{Name: "JAVA_OPTS", Value: jaeger.Spec.Storage.Dependencies.JavaOpts},
	}
	envVars = append(envVars, getStorageEnvs(jaeger.Spec.Storage)...)

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

	trueVar := true
	one := int32(1)
	name := fmt.Sprintf("%s-spark-dependencies", jaeger.Name)

	baseCommonSpec := v1.JaegerCommonSpec{
		Annotations: map[string]string{
			"prometheus.io/scrape":    "false",
			"sidecar.istio.io/inject": "false",
			"linkerd.io/inject":       "disabled",
		},
	}

	commonSpec := util.Merge([]v1.JaegerCommonSpec{jaeger.Spec.Storage.Dependencies.JaegerCommonSpec, jaeger.Spec.JaegerCommonSpec, baseCommonSpec})

	return &batchv1beta1.CronJob{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: jaeger.Namespace,
			Labels: map[string]string{
				"app":                          "jaeger",
				"app.kubernetes.io/name":       name,
				"app.kubernetes.io/instance":   jaeger.Name,
				"app.kubernetes.io/component":  "cronjob-spark-dependencies",
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
			ConcurrencyPolicy:          batchv1beta1.ForbidConcurrent,
			Schedule:                   jaeger.Spec.Storage.Dependencies.Schedule,
			SuccessfulJobsHistoryLimit: jaeger.Spec.Storage.Dependencies.SuccessfulJobsHistoryLimit,
			JobTemplate: batchv1beta1.JobTemplateSpec{
				Spec: batchv1.JobSpec{
					Parallelism: &one,
					Template: corev1.PodTemplateSpec{
						Spec: corev1.PodSpec{
							Containers: []corev1.Container{
								{
									Image: jaeger.Spec.Storage.Dependencies.Image,
									Name:  name,
									// let spark job use its default values
									Env:       removeEmptyVars(envVars),
									EnvFrom:   envFromSource,
									Resources: commonSpec.Resources,
								},
							},
							RestartPolicy:      corev1.RestartPolicyNever,
							Affinity:           commonSpec.Affinity,
							Tolerations:        commonSpec.Tolerations,
							SecurityContext:    commonSpec.SecurityContext,
							ServiceAccountName: account.JaegerServiceAccountFor(jaeger, account.DependenciesComponent),
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

func getStorageEnvs(s v1.JaegerStorageSpec) []corev1.EnvVar {
	sFlags := s.Options.Filter(storage.OptionsPrefix(s.Type))
	sFlagsMap := sFlags.Map()
	keyspace := sFlagsMap["cassandra.keyspace"]
	if keyspace == "" {
		keyspace = "jaeger_v1_test"
	}

	switch s.Type {
	case "cassandra":
		return []corev1.EnvVar{
			{Name: "CASSANDRA_CONTACT_POINTS", Value: sFlagsMap["cassandra.servers"]},
			{Name: "CASSANDRA_KEYSPACE", Value: keyspace},
			{Name: "CASSANDRA_USERNAME", Value: sFlagsMap["cassandra.username"]},
			{Name: "CASSANDRA_PASSWORD", Value: sFlagsMap["cassandra.password"]},
			{Name: "CASSANDRA_USE_SSL", Value: sFlagsMap["cassandra.tls"]},
			{Name: "CASSANDRA_LOCAL_DC", Value: sFlagsMap["cassandra.local-dc"]},
			{Name: "CASSANDRA_CLIENT_AUTH_ENABLED", Value: strconv.FormatBool(s.Dependencies.CassandraClientAuthEnabled)},
		}

	case "elasticsearch":
		env := []corev1.EnvVar{
			{Name: "ES_NODES", Value: sFlagsMap["es.server-urls"]},
			{Name: "ES_INDEX_PREFIX", Value: sFlagsMap["es.index-prefix"]},
			{Name: "ES_CLIENT_NODE_ONLY", Value: strconv.FormatBool(s.Dependencies.ElasticsearchClientNodeOnly)},
			{Name: "ES_NODES_WAN_ONLY", Value: strconv.FormatBool(s.Dependencies.ElasticsearchNodesWanOnly)},
		}

		// we add ES_USERNAME and ES_PASSWORD (as empty strings) if the secret is empty or if we know that we have both
		// username and passwords in the options Map (if also there's a secret, it's up to the user to make sure it is
		// right at runtime, as the secret can be used for more stuff than ES_USERNAME/ES_PASSWORD)
		username, hasUsername := sFlagsMap["es.username"]
		password, hasPassword := sFlagsMap["es.password"]
		if len(s.SecretName) == 0 || (hasUsername && hasPassword) {
			env = append(env,
				corev1.EnvVar{Name: "ES_USERNAME", Value: username},
				corev1.EnvVar{Name: "ES_PASSWORD", Value: password})
		}
		return env

	default:
		return nil
	}
}

func removeEmptyVars(envVars []corev1.EnvVar) []corev1.EnvVar {
	var notEmpty []corev1.EnvVar
	for _, v := range envVars {
		if v.Value != "" || v.ValueFrom != nil {
			notEmpty = append(notEmpty, v)
		}
	}
	return notEmpty
}
