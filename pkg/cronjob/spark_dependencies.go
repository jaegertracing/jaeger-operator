package cronjob

import (
	"strconv"
	"strings"

	batchv1 "k8s.io/api/batch/v1"
	batchv1beta1 "k8s.io/api/batch/v1beta1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/spf13/viper"

	"github.com/jaegertracing/jaeger-operator/pkg/account"
	v1 "github.com/jaegertracing/jaeger-operator/pkg/apis/jaegertracing/v1"
	"github.com/jaegertracing/jaeger-operator/pkg/config/ca"
	"github.com/jaegertracing/jaeger-operator/pkg/util"
)

var supportedStorageTypes = map[string]bool{"elasticsearch": true, "cassandra": true}

// SupportedStorage returns whether the given storage is supported
func SupportedStorage(storage string) bool {
	return supportedStorageTypes[strings.ToLower(storage)]
}

// CreateSparkDependencies creates a new cronjob for the Spark Dependencies task
func CreateSparkDependencies(jaeger *v1.Jaeger) *batchv1beta1.CronJob {
	logTLSNotSupported(jaeger)
	envVars := []corev1.EnvVar{
		{Name: "STORAGE", Value: jaeger.Spec.Storage.Type},
		{Name: "SPARK_MASTER", Value: jaeger.Spec.Storage.Dependencies.SparkMaster},
		{Name: "JAVA_OPTS", Value: jaeger.Spec.Storage.Dependencies.JavaOpts},
	}
	envVars = append(envVars, getStorageEnvs(jaeger.Spec.Storage)...)

	envFromSource := util.CreateEnvsFromSecret(jaeger.Spec.Storage.SecretName)

	trueVar := true
	one := int32(1)

	// cron job names are restricted to 52 chars
	name := util.Truncate("%s-spark-dependencies", 52, jaeger.Name)

	baseCommonSpec := v1.JaegerCommonSpec{
		Annotations: map[string]string{
			"prometheus.io/scrape":    "false",
			"sidecar.istio.io/inject": "false",
			"linkerd.io/inject":       "disabled",
		},
		Labels: util.Labels(name, "spark-dependencies", *jaeger),
	}

	commonSpec := util.Merge([]v1.JaegerCommonSpec{jaeger.Spec.Storage.Dependencies.JaegerCommonSpec, jaeger.Spec.JaegerCommonSpec, baseCommonSpec})

	ca.Update(jaeger, commonSpec)

	// Cannot use util.ImageName to obtain the correct image, as the spark-dependencies
	// image does not get tagged with the jaeger version, so the latest image must
	// be used instead.
	image := jaeger.Spec.Storage.Dependencies.Image
	if image == "" {
		// the version is not included, there is only one version - latest
		image = viper.GetString("jaeger-spark-dependencies-image")
	}

	return &batchv1beta1.CronJob{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: jaeger.Namespace,
			Labels:    commonSpec.Labels,
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
									Image: image,
									Name:  name,
									// let spark job use its default values
									Env:       util.RemoveEmptyVars(envVars),
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
	sFlagsMap := s.Options.Map()
	switch s.Type {
	case "cassandra":
		keyspace := sFlagsMap["cassandra.keyspace"]
		if keyspace == "" {
			keyspace = "jaeger_v1_test"
		}
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
		vars := []corev1.EnvVar{
			{Name: "ES_NODES", Value: sFlagsMap["es.server-urls"]},
			{Name: "ES_INDEX_PREFIX", Value: sFlagsMap["es.index-prefix"]},
			{Name: "ES_USERNAME", Value: sFlagsMap["es.username"]},
			{Name: "ES_PASSWORD", Value: sFlagsMap["es.password"]},
		}
		if s.Dependencies.ElasticsearchNodesWanOnly != nil {
			vars = append(vars, corev1.EnvVar{Name: "ES_NODES_WAN_ONLY", Value: strconv.FormatBool(*s.Dependencies.ElasticsearchNodesWanOnly)})
		}
		if s.Dependencies.ElasticsearchClientNodeOnly != nil {
			vars = append(vars, corev1.EnvVar{Name: "ES_CLIENT_NODE_ONLY", Value: strconv.FormatBool(*s.Dependencies.ElasticsearchClientNodeOnly)})
		}
		return vars
	default:
		return nil
	}
}

func logTLSNotSupported(j *v1.Jaeger) {
	sFlagsMap := j.Spec.Storage.Options.Map()
	if strings.EqualFold(sFlagsMap["es.tls.enabled"], "true") || strings.EqualFold(sFlagsMap["es.tls"], "true") {
		j.Logger().Warn("Spark dependencies does not support TLS with Elasticsearch, consider disabling dependencies")
	}
	if strings.EqualFold(sFlagsMap["es.tls.skip-host-verify"], "true") || sFlagsMap["es.tls.ca"] != "" {
		j.Logger().Warn("Spark dependencies does not support insecure TLS nor specifying only CA cert, consider disabling dependencies")
	}
}
