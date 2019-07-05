package storage

import (
	"fmt"

	"github.com/spf13/viper"
	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	v1 "github.com/jaegertracing/jaeger-operator/pkg/apis/jaegertracing/v1"
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

	if jaeger.Spec.Storage.CassandraCreateSchema.Image == "" {
		jaeger.Spec.Storage.CassandraCreateSchema.Image = fmt.Sprintf("%s:%s", viper.GetString("jaeger-cassandra-schema-image"), viper.GetString("jaeger-version"))
	}

	host := jaeger.Spec.Storage.Options.Map()["cassandra.servers"]
	if host == "" {
		jaeger.Logger().Info("Cassandra hostname not specified. Using 'cassandra' for the cassandra-create-schema job.")
		host = "cassandra" // this is the default in the image
	}

	annotations := map[string]string{
		"prometheus.io/scrape":    "false",
		"sidecar.istio.io/inject": "false",
		"linkerd.io/inject":       "disabled",
	}

	// TODO: should this be configurable? Would we ever think that 2 minutes is OK for this job to complete?
	deadline := int64(120)

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
				ActiveDeadlineSeconds: &deadline,
				Template: corev1.PodTemplateSpec{
					ObjectMeta: metav1.ObjectMeta{
						Annotations: annotations,
					},
					Spec: corev1.PodSpec{
						Containers: []corev1.Container{{
							Image: jaeger.Spec.Storage.CassandraCreateSchema.Image,
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
							}},
						}},
						RestartPolicy: corev1.RestartPolicyOnFailure,
					},
				},
			},
		},
	}
}
