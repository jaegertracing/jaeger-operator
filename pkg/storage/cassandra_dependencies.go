package storage

import (
	"fmt"
	"time"

	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	v1 "github.com/jaegertracing/jaeger-operator/pkg/apis/jaegertracing/v1"
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

	host := jaeger.Spec.Storage.Options.Map()["cassandra.servers"]
	if host == "" {
		jaeger.Logger().Info("Cassandra hostname not specified. Using 'cassandra' for the cassandra-create-schema job.")
		host = "cassandra" // this is the default in the image
	}

	keyspace := jaeger.Spec.Storage.Options.Map()["cassandra.keyspace"]
	if keyspace == "" {
		jaeger.Logger().Info("Cassandra keyspace not specified. Using 'jaeger_v1_test' for the cassandra-create-schema job.")
		keyspace = "jaeger_v1_test" // this is default in the image
	}

	username := jaeger.Spec.Storage.Options.Map()["cassandra.username"]
	password := jaeger.Spec.Storage.Options.Map()["cassandra.password"]

	annotations := map[string]string{
		"prometheus.io/scrape":    "false",
		"sidecar.istio.io/inject": "false",
		"linkerd.io/inject":       "disabled",
	}

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

	if jaeger.Spec.Storage.CassandraCreateSchema.Timeout != "" {
		dur, err := time.ParseDuration(jaeger.Spec.Storage.CassandraCreateSchema.Timeout)
		if err == nil {
			seconds := int64(dur.Seconds())
			jobTimeout = &seconds
		} else {
			jaeger.Logger().
				WithError(err).
				WithField("timeout", jaeger.Spec.Storage.CassandraCreateSchema.Timeout).
				Error("Failed to parse cassandraCreateSchema.timeout to time.duration. Using the default.")
		}
	} else {
		jaeger.Logger().Debug("Timeout for cassandra-create-schema job not specified. Using default of 1 day.")
	}

	return []batchv1.Job{
		batchv1.Job{
			TypeMeta: metav1.TypeMeta{
				APIVersion: "batch/v1",
				Kind:       "Job",
			},
			ObjectMeta: metav1.ObjectMeta{
				Name:      fmt.Sprintf("%s-cassandra-schema-job", jaeger.Name),
				Namespace: jaeger.Namespace,
				Labels: map[string]string{
					"app":                          "jaeger",
					"app.kubernetes.io/name":       fmt.Sprintf("%s-cassandra-schema-job", jaeger.Name),
					"app.kubernetes.io/instance":   jaeger.Name,
					"app.kubernetes.io/component":  "cronjob-cassandra-schema",
					"app.kubernetes.io/part-of":    "jaeger",
					"app.kubernetes.io/managed-by": "jaeger-operator",
				},
				OwnerReferences: []metav1.OwnerReference{
					metav1.OwnerReference{
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
						Annotations: annotations,
					},
					Spec: corev1.PodSpec{
						ActiveDeadlineSeconds: podTimeout,
						Containers: []corev1.Container{{
							Image: util.ImageName(jaeger.Spec.Storage.CassandraCreateSchema.Image, "jaeger-cassandra-schema-image"),
							Name:  fmt.Sprintf("%s-cassandra-schema", jaeger.Name),
							Env: []corev1.EnvVar{{
								Name:  "CQLSH_HOST",
								Value: host,
							}, {
								Name:  "MODE",
								Value: jaeger.Spec.Storage.CassandraCreateSchema.Mode,
							}, {
								Name:  "DATACENTER",
								Value: jaeger.Spec.Storage.CassandraCreateSchema.Datacenter,
							}, {
								Name:  "KEYSPACE",
								Value: keyspace,
							}, {
								Name:  "CASSANDRA_USERNAME",
								Value: username,
							}, {
								Name:  "CASSANDRA_PASSWORD",
								Value: password,
							}},
						}},
						RestartPolicy: corev1.RestartPolicyOnFailure,
					},
				},
			},
		},
	}
}
