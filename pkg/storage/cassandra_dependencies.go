package storage

import (
	"fmt"
	"strings"
	"time"

	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	v1 "github.com/jaegertracing/jaeger-operator/apis/v1"
	"github.com/jaegertracing/jaeger-operator/pkg/util"
)

func cassandraDeps(jaeger *v1.Jaeger) []batchv1.Job {
	trueVar := true

	// TODO should be moved to normalize
	if jaeger.Spec.Storage.CassandraCreateSchema.Enabled == nil {
		jaeger.Spec.Storage.CassandraCreateSchema.Enabled = &trueVar
	}

	// if the create-schema job is disabled, return an empty list
	if !*jaeger.Spec.Storage.CassandraCreateSchema.Enabled {
		return []batchv1.Job{}
	}

	if jaeger.Spec.Storage.CassandraCreateSchema.Datacenter == "" {
		// the default in the create-schema is "dc1", but the default in Jaeger is "test"! We align with Jaeger here
		jaeger.Logger().Info("Datacenter not specified. Using 'test' for the cassandra-create-schema job.")
		jaeger.Spec.Storage.CassandraCreateSchema.Datacenter = "test"
	}

	if jaeger.Spec.Storage.CassandraCreateSchema.Mode == "" {
		jaeger.Logger().Info("Mode not specified. Using 'prod' for the cassandra-create-schema job.")
		jaeger.Spec.Storage.CassandraCreateSchema.Mode = "prod"
	}

	envVars := []corev1.EnvVar{{
		Name:  "MODE",
		Value: jaeger.Spec.Storage.CassandraCreateSchema.Mode,
	}, {
		Name:  "DATACENTER",
		Value: jaeger.Spec.Storage.CassandraCreateSchema.Datacenter,
	}}

	servers := jaeger.Spec.Storage.Options.StringMap()["cassandra.servers"]
	host := strings.Split(servers, ",")[0] //  <- choose first server address
	if host == "" {
		jaeger.Logger().Info("Cassandra hostname not specified. Using 'cassandra' for the cassandra-create-schema job.")
		host = "cassandra" // this is the default in the image
	}
	envVars = append(envVars, corev1.EnvVar{
		Name:  "CQLSH_HOST",
		Value: host,
	})

	port := jaeger.Spec.Storage.Options.StringMap()["cassandra.port"]
	if port == "" {
		jaeger.Logger().Info("Cassandra port not specified. Using '9042' for the cassandra-create-schema job.")
		port = "9042" // this is the default in the image
	}
	envVars = append(envVars, corev1.EnvVar{
		Name:  "CQLSH_PORT",
		Value: port,
	})

	keyspace := jaeger.Spec.Storage.Options.StringMap()["cassandra.keyspace"]
	if keyspace == "" {
		jaeger.Logger().Info("Cassandra keyspace not specified. Using 'jaeger_v1_test' for the cassandra-create-schema job.")
		keyspace = "jaeger_v1_test" // this is default in the image
	}

	envVars = append(envVars, corev1.EnvVar{
		Name:  "KEYSPACE",
		Value: keyspace,
	})
	username := jaeger.Spec.Storage.Options.StringMap()["cassandra.username"]
	password := jaeger.Spec.Storage.Options.StringMap()["cassandra.password"]

	envFromSource := util.CreateEnvsFromSecret(jaeger.Spec.Storage.SecretName)
	if len(envFromSource) == 0 {
		envVars = append(envVars, corev1.EnvVar{
			Name:  "CASSANDRA_USERNAME",
			Value: username,
		})
		envVars = append(envVars, corev1.EnvVar{
			Name:  "CASSANDRA_PASSWORD",
			Value: password,
		})
	}

	truncatedName := util.Truncate("%s-cassandra-schema-job", 63, jaeger.Name)

	commonSpec := &v1.JaegerCommonSpec{
		Annotations: map[string]string{
			"prometheus.io/scrape":    "false",
			"sidecar.istio.io/inject": "false",
			"linkerd.io/inject":       "disabled",
		},
		Labels: util.Labels(truncatedName, "cronjob-cassandra-schema", *jaeger),
	}
	commonSpec = util.Merge([]v1.JaegerCommonSpec{jaeger.Spec.Collector.JaegerCommonSpec, jaeger.Spec.JaegerCommonSpec, *commonSpec})

	// Set job deadline to 1 day by default. If the job does not succeed within
	// that duration it transitions into a permanent error state.
	// See https://github.com/jaegertracing/jaeger-kubernetes/issues/32 and
	// https://github.com/jaegertracing/jaeger-kubernetes/pull/125
	oneDaySeconds := int64(86400)
	jobTimeout := &oneDaySeconds

	// The schema creation code running in a container in this pod has a retry
	// loop which is supposed to retry forever. However, if that retry loop
	// does not yield success within ~5 minutes then restart the container in
	// the pod, effectively restarting the inner retry loop. This guards
	// against the unlikely case of the code running in the container being
	// dead-locked for whichever reason. See jaeger-kubernetes/issues/32.
	podTimeoutSeconds := int64(320)
	podTimeout := &podTimeoutSeconds

	// TTL for trace data, in seconds (default: 172800, 2 days)
	// see: https://github.com/jaegertracing/jaeger/blob/main/plugin/storage/cassandra/schema/create.sh
	traceTTLSeconds := "172800"
	if jaeger.Spec.Storage.CassandraCreateSchema.TraceTTL != "" {
		dur, err := time.ParseDuration(jaeger.Spec.Storage.CassandraCreateSchema.TraceTTL)
		if err != nil {
			jaeger.Logger().Error(
				err,
				"Failed to parse cassandraCreateSchema.traceTTL to time.duration. Using the default.",
				"timeout", jaeger.Spec.Storage.CassandraCreateSchema.TraceTTL,
			)
		} else {
			traceTTLSeconds = fmt.Sprintf("%.0f", dur.Seconds())
		}
	}
	envVars = append(envVars, corev1.EnvVar{
		Name:  "TRACE_TTL",
		Value: traceTTLSeconds,
	})

	if jaeger.Spec.Storage.CassandraCreateSchema.Timeout != "" {
		dur, err := time.ParseDuration(jaeger.Spec.Storage.CassandraCreateSchema.Timeout)
		if err == nil {
			seconds := int64(dur.Seconds())
			jobTimeout = &seconds
		} else {
			jaeger.Logger().Error(
				err,
				"Failed to parse cassandraCreateSchema.timeout to time.duration. Using the default.",
				"timeout", jaeger.Spec.Storage.CassandraCreateSchema.Timeout,
			)
		}
	} else {
		jaeger.Logger().V(-1).Info(
			"Timeout for cassandra-create-schema job not specified. Using default of 1 day.",
		)
	}

	return []batchv1.Job{
		{
			TypeMeta: metav1.TypeMeta{
				APIVersion: "batch/v1",
				Kind:       "Job",
			},
			ObjectMeta: metav1.ObjectMeta{
				// while the name itself isn't a problem, Kubernetes will create a job with a label "job-name" with this value
				// so, this value has to be restricted to 63 chars
				Name:      truncatedName,
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
			Spec: batchv1.JobSpec{
				ActiveDeadlineSeconds: jobTimeout,
				Template: corev1.PodTemplateSpec{
					ObjectMeta: metav1.ObjectMeta{
						Annotations: commonSpec.Annotations,
					},
					Spec: corev1.PodSpec{
						ImagePullSecrets:      commonSpec.ImagePullSecrets,
						ActiveDeadlineSeconds: podTimeout,
						Affinity:              jaeger.Spec.Storage.CassandraCreateSchema.Affinity,
						Tolerations:           commonSpec.Tolerations,
						SecurityContext:       commonSpec.SecurityContext,
						Volumes:               commonSpec.Volumes,
						Containers: []corev1.Container{{
							Image:        util.ImageName(jaeger.Spec.Storage.CassandraCreateSchema.Image, "jaeger-cassandra-schema-image"),
							Name:         truncatedName,
							Env:          envVars,
							EnvFrom:      envFromSource,
							Resources:    commonSpec.Resources,
							VolumeMounts: commonSpec.VolumeMounts,
						}},
						RestartPolicy: corev1.RestartPolicyOnFailure,
					},
				},
			},
		},
	}
}
