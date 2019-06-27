package cronjob

import (
	"fmt"
	"strconv"
	"strings"

	batchv1 "k8s.io/api/batch/v1"
	batchv1beta1 "k8s.io/api/batch/v1beta1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	v1 "github.com/jaegertracing/jaeger-operator/pkg/apis/jaegertracing/v1"
	"github.com/jaegertracing/jaeger-operator/pkg/storage"
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

	trueVar := true
	one := int32(1)
	name := fmt.Sprintf("%s-spark-dependencies", jaeger.Name)
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
			ConcurrencyPolicy: batchv1beta1.ForbidConcurrent,
			Schedule:          jaeger.Spec.Storage.Dependencies.Schedule,
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
									Env: removeEmptyVars(envVars),
								},
							},
							RestartPolicy: corev1.RestartPolicyNever,
						},
						ObjectMeta: metav1.ObjectMeta{
							Annotations: map[string]string{
								"prometheus.io/scrape":    "false",
								"sidecar.istio.io/inject": "false",
								"linkerd.io/inject":       "disabled",
							},
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
		return []corev1.EnvVar{
			{Name: "ES_NODES", Value: sFlagsMap["es.server-urls"]},
			{Name: "ES_INDEX_PREFIX", Value: sFlagsMap["es.index-prefix"]},
			{Name: "ES_USERNAME", Value: sFlagsMap["es.username"]},
			{Name: "ES_PASSWORD", Value: sFlagsMap["es.password"]},
			{Name: "ES_CLIENT_NODE_ONLY", Value: strconv.FormatBool(s.Dependencies.ElasticsearchClientNodeOnly)},
			{Name: "ES_NODES_WAN_ONLY", Value: strconv.FormatBool(s.Dependencies.ElasticsearchNodesWanOnly)},
		}
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
