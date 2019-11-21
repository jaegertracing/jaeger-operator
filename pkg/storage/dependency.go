package storage

import (
	"strings"

	batchv1 "k8s.io/api/batch/v1"

	v1 "github.com/jaegertracing/jaeger-operator/pkg/apis/jaegertracing/v1"
)

// Dependencies return a list of Jobs that have to be finished before the other components are deployed
func Dependencies(jaeger *v1.Jaeger) []batchv1.Job {
	if strings.EqualFold(jaeger.Spec.Storage.Type, "cassandra") {
		return cassandraDeps(jaeger)
	}
	if EnableRollover(jaeger.Spec.Storage) {
		return elasticsearchDependencies(jaeger)
	}
	return nil
}
