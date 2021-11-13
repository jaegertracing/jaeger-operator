package storage

import (
	batchv1 "k8s.io/api/batch/v1"

	v1 "github.com/jaegertracing/jaeger-operator/apis/v1"
)

// Dependencies return a list of Jobs that have to be finished before the other components are deployed
func Dependencies(jaeger *v1.Jaeger) []batchv1.Job {
	if jaeger.Spec.Storage.Type == v1.JaegerCassandraStorage {
		return cassandraDeps(jaeger)
	}
	if EnableRollover(jaeger.Spec.Storage) {
		return elasticsearchDependencies(jaeger)
	}
	return nil
}
