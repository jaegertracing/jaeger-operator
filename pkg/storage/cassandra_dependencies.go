package storage

import (
	"fmt"

	"github.com/sirupsen/logrus"
	"github.com/spf13/viper"
	batchv1 "k8s.io/api/batch/v1"
	"k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/jaegertracing/jaeger-operator/pkg/apis/io/v1alpha1"
)

func cassandraDeps(jaeger *v1alpha1.Jaeger) []batchv1.Job {
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
		logrus.WithField("instance", jaeger.Name).Info("Datacenter not specified. Using 'test' for the cassandra-create-schema job.")
		jaeger.Spec.Storage.CassandraCreateSchema.Datacenter = "test"
	}

	if jaeger.Spec.Storage.CassandraCreateSchema.Mode == "" {
		logrus.WithField("instance", jaeger.Name).Info("Mode not specified. Using 'prod' for the cassandra-create-schema job.")
		jaeger.Spec.Storage.CassandraCreateSchema.Mode = "prod"
	}

	if jaeger.Spec.Storage.CassandraCreateSchema.Image == "" {
		jaeger.Spec.Storage.CassandraCreateSchema.Image = fmt.Sprintf("%s:%s", viper.GetString("jaeger-cassandra-schema-image"), jaeger.Spec.Version)
	}

	host := jaeger.Spec.Storage.Options.Map()["cassandra.servers"]
	if host == "" {
		logrus.WithField("instance", jaeger.Name).Info("Cassandra hostname not specified. Using 'cassandra' for the cassandra-create-schema job.")
		host = "cassandra" // this is the default in the image
	}

	annotations := map[string]string{
		"prometheus.io/scrape":    "false",
		"sidecar.istio.io/inject": "false",
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
				Template: v1.PodTemplateSpec{
					ObjectMeta: metav1.ObjectMeta{
						Annotations: annotations,
					},
					Spec: v1.PodSpec{
						Containers: []v1.Container{
							v1.Container{
								Image: jaeger.Spec.Storage.CassandraCreateSchema.Image,
								Name:  fmt.Sprintf("%s-cassandra-schema", jaeger.Name),
								Env: []v1.EnvVar{
									v1.EnvVar{
										Name:  "CQLSH_HOST",
										Value: host,
									},
									v1.EnvVar{
										Name:  "MODE",
										Value: jaeger.Spec.Storage.CassandraCreateSchema.Mode,
									},
									v1.EnvVar{
										Name:  "DATACENTER",
										Value: jaeger.Spec.Storage.CassandraCreateSchema.Datacenter,
									},
								},
							},
						},
						RestartPolicy: v1.RestartPolicyOnFailure,
					},
				},
			},
		},
	}
}
