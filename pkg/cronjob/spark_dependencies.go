package cronjob

import (
	"strconv"
	"strings"

	"k8s.io/apimachinery/pkg/runtime"

	batchv1 "k8s.io/api/batch/v1"
	batchv1beta1 "k8s.io/api/batch/v1beta1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/spf13/viper"

	v1 "github.com/jaegertracing/jaeger-operator/apis/v1"
	"github.com/jaegertracing/jaeger-operator/pkg/account"
	"github.com/jaegertracing/jaeger-operator/pkg/config/ca"
	"github.com/jaegertracing/jaeger-operator/pkg/util"
)

var supportedStorageTypes = map[v1.JaegerStorageType]bool{v1.JaegerESStorage: true, v1.JaegerCassandraStorage: true}

// SupportedStorage returns whether the given storage is supported
func SupportedStorage(storage v1.JaegerStorageType) bool {
	return supportedStorageTypes[storage]
}

// CreateSparkDependencies creates a new cronjob for the Spark Dependencies task
func CreateSparkDependencies(jaeger *v1.Jaeger) runtime.Object {
	logTLSNotSupported(jaeger)
	envVars := []corev1.EnvVar{
		{Name: "STORAGE", Value: string(jaeger.Spec.Storage.Type)},
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

	objectMeta := metav1.ObjectMeta{
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
	}

	jobSpec := batchv1.JobSpec{
		Parallelism:  &one,
		BackoffLimit: jaeger.Spec.Storage.Dependencies.BackoffLimit,
		Template: corev1.PodTemplateSpec{
			Spec: corev1.PodSpec{
				Containers: []corev1.Container{
					{
						Image: image,
						Name:  name,
						// let spark job use its default values
						Env:          util.RemoveEmptyVars(envVars),
						EnvFrom:      envFromSource,
						Resources:    commonSpec.Resources,
						VolumeMounts: jaeger.Spec.Storage.Dependencies.JaegerCommonSpec.VolumeMounts,
					},
				},
				ImagePullSecrets:   commonSpec.ImagePullSecrets,
				RestartPolicy:      corev1.RestartPolicyNever,
				Affinity:           commonSpec.Affinity,
				Tolerations:        commonSpec.Tolerations,
				SecurityContext:    commonSpec.SecurityContext,
				ServiceAccountName: account.JaegerServiceAccountFor(jaeger, account.DependenciesComponent),
				Volumes:            jaeger.Spec.Storage.Dependencies.JaegerCommonSpec.Volumes,
			},
			ObjectMeta: metav1.ObjectMeta{
				Labels:      commonSpec.Labels,
				Annotations: commonSpec.Annotations,
			},
		},
	}

	var o runtime.Object
	cronjobsVersion := viper.GetString(v1.FlagCronJobsVersion)
	if cronjobsVersion == v1.FlagCronJobsVersionBatchV1Beta1 {
		cj := &batchv1beta1.CronJob{
			ObjectMeta: objectMeta,
			Spec: batchv1beta1.CronJobSpec{
				ConcurrencyPolicy:          batchv1beta1.ForbidConcurrent,
				Schedule:                   jaeger.Spec.Storage.Dependencies.Schedule,
				SuccessfulJobsHistoryLimit: jaeger.Spec.Storage.Dependencies.SuccessfulJobsHistoryLimit,
				JobTemplate: batchv1beta1.JobTemplateSpec{
					Spec: jobSpec,
				},
			},
		}

		o = cj
	} else {
		cj := &batchv1.CronJob{
			ObjectMeta: objectMeta,
			Spec: batchv1.CronJobSpec{
				ConcurrencyPolicy:          batchv1.ForbidConcurrent,
				Schedule:                   jaeger.Spec.Storage.Dependencies.Schedule,
				SuccessfulJobsHistoryLimit: jaeger.Spec.Storage.Dependencies.SuccessfulJobsHistoryLimit,
				JobTemplate: batchv1.JobTemplateSpec{
					Spec: jobSpec,
				},
			},
		}

		o = cj
	}

	return o
}

func getStorageEnvs(s v1.JaegerStorageSpec) []corev1.EnvVar {
	sFlagsMap := s.Options.StringMap()
	switch s.Type {
	case v1.JaegerCassandraStorage:
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
	case v1.JaegerESStorage:
		vars := []corev1.EnvVar{
			{Name: "ES_NODES", Value: sFlagsMap["es.server-urls"]},
			{Name: "ES_INDEX_PREFIX", Value: sFlagsMap["es.index-prefix"]},
			{Name: "ES_INDEX_DATE_SEPARATOR", Value: sFlagsMap["es.index-date-separator"]},
			{Name: "ES_USERNAME", Value: sFlagsMap["es.username"]},
			{Name: "ES_PASSWORD", Value: sFlagsMap["es.password"]},
			{Name: "ES_TIME_RANGE", Value: s.Dependencies.ElasticsearchTimeRange},
			{Name: "ES_USE_ALIASES", Value: sFlagsMap["es.use-aliases"]},
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
	sFlagsMap := j.Spec.Storage.Options.StringMap()
	if strings.EqualFold(sFlagsMap["es.tls.enabled"], "true") || strings.EqualFold(sFlagsMap["es.tls"], "true") {
		j.Logger().V(1).Info(
			"Spark dependencies does not support TLS with Elasticsearch, consider disabling dependencies",
		)
	}
	if strings.EqualFold(sFlagsMap["es.tls.skip-host-verify"], "true") || sFlagsMap["es.tls.ca"] != "" {
		j.Logger().V(1).Info(
			"Spark dependencies does not support insecure TLS nor specifying only CA cert, consider disabling dependencies",
		)
	}
}
