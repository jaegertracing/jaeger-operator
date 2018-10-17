package storage

import (
	"strings"

	batchv1 "k8s.io/api/batch/v1"

	"github.com/jaegertracing/jaeger-operator/pkg/apis/io/v1alpha1"
)

// Dependencies return a list of Jobs that have to be finished before the other components are deployed
func Dependencies(jaeger *v1alpha1.Jaeger) []batchv1.Job {
	if strings.ToLower(jaeger.Spec.Storage.Type) == "cassandra" {
		return cassandraDeps(jaeger)
	}

	return []batchv1.Job{}
}
