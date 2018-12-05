package cronjob

import (
	"fmt"
	"strconv"

	batchv1 "k8s.io/api/batch/v1"
	batchv1beta1 "k8s.io/api/batch/v1beta1"
	"k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/jaegertracing/jaeger-operator/pkg/apis/io/v1alpha1"
	"github.com/jaegertracing/jaeger-operator/pkg/storage"
)

var supportedStorageTypes = map[string]bool{"elasticsearch": true, "cassandra": true}

func SupportedStorage(storage string) bool {
	return supportedStorageTypes[storage]
}

func CreateSparkDependencies(jaeger *v1alpha1.Jaeger) *batchv1beta1.CronJob {
	envVars := []v1.EnvVar{
		{Name: "STORAGE", Value: jaeger.Spec.Storage.Type},
		{Name: "SPARK_MASTER", Value: jaeger.Spec.Storage.SparkDependencies.SparkMaster},
		{Name: "JAVA_OPTS", Value: jaeger.Spec.Storage.SparkDependencies.JavaOpts},
	}
	envVars = append(envVars, getStorageEnvs(jaeger.Spec.Storage)...)

	trueVar := true
	name := fmt.Sprintf("%s-spark-dependencies", jaeger.Name)
	return &batchv1beta1.CronJob{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: jaeger.Namespace,
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
			Schedule: jaeger.Spec.Storage.SparkDependencies.Schedule,
			JobTemplate: batchv1beta1.JobTemplateSpec{
				Spec: batchv1.JobSpec{
					Template: v1.PodTemplateSpec{
						Spec: v1.PodSpec{
							Containers: []v1.Container{
								{
									Image: jaeger.Spec.Storage.SparkDependencies.Image,
									Name:  name,
									// let spark job use its default values
									Env: removeEmptyVars(envVars),
								},
							},
							RestartPolicy: v1.RestartPolicyNever,
						},
						ObjectMeta: metav1.ObjectMeta{
							Annotations: map[string]string{
								"prometheus.io/scrape":    "false",
								"sidecar.istio.io/inject": "false",
							},
						},
					},
				},
			},
		},
	}
}

func getStorageEnvs(s v1alpha1.JaegerStorageSpec) []v1.EnvVar {
	sFlags := s.Options.Filter(storage.OptionsPrefix(s.Type))
	sFlagsMap := sFlags.Map()
	keyspace := sFlagsMap["cassandra.keyspace"]
	if keyspace == "" {
		keyspace = "jaeger_v1_test"
	}
	switch s.Type {
	case "cassandra":
		return []v1.EnvVar{
			{Name: "CASSANDRA_CONTACT_POINTS", Value: sFlagsMap["cassandra.servers"]},
			{Name: "CASSANDRA_KEYSPACE", Value: keyspace},
			{Name: "CASSANDRA_USERNAME", Value: sFlagsMap["cassandra.username"]},
			{Name: "CASSANDRA_PASSWORD", Value: sFlagsMap["cassandra.password"]},
			{Name: "CASSANDRA_USE_SSL", Value: strconv.FormatBool(s.SparkDependencies.CassandraUseSsl)},
			{Name: "CASSANDRA_LOCAL_DC", Value: s.SparkDependencies.CassandraLocalDc},
			{Name: "CASSANDRA_CLIENT_AUTH_ENABLED", Value: strconv.FormatBool(s.SparkDependencies.CassandraClientAuthEnabled)},
		}
	case "elasticsearch":
		return []v1.EnvVar{
			{Name: "ES_NODES", Value: sFlagsMap["es.server-urls"]},
			{Name: "ES_INDEX_PREFIX", Value: sFlagsMap["es.index-prefix"]},
			{Name: "ES_USERNAME", Value: sFlagsMap["es.username"]},
			{Name: "ES_PASSWORD", Value: sFlagsMap["es.password"]},
			{Name: "ES_CLIENT_NODE_ONLY", Value: strconv.FormatBool(s.SparkDependencies.ElasticsearchClientNodeOnly)},
			{Name: "ES_NODES_WAN_ONLY", Value: strconv.FormatBool(s.SparkDependencies.ElasticsearchNodesWanOnly)},
		}
	default:
		return nil
	}
}

func removeEmptyVars(envVars []v1.EnvVar) []v1.EnvVar {
	var notEmpty []v1.EnvVar
	for _, v := range envVars {
		if v.Value != "" || v.ValueFrom != nil {
			notEmpty = append(notEmpty, v)
		}
	}
	return notEmpty
}
