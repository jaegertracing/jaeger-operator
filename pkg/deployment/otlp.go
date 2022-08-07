package deployment

import (
	corev1 "k8s.io/api/core/v1"

	"github.com/jaegertracing/jaeger-operator/pkg/util"
)

const collectorOTLPEnvVarName = "COLLECTOR_OTLP_ENABLED"

func getOTLPEnvVars(options []string) []corev1.EnvVar {
	if !util.IsOTLPExplcitSet(options) {
		return []corev1.EnvVar{
			{
				Name:  collectorOTLPEnvVarName,
				Value: "true",
			},
		}
	}
	return []corev1.EnvVar{}
}

func getOTLPContainePorts(options []string) []corev1.ContainerPort {
	if util.IsOTLPEnable(options) {
		return []corev1.ContainerPort{
			{
				ContainerPort: 4317,
				Name:          "grpc-otlp",
			},
			{
				ContainerPort: 4318,
				Name:          "http-otlp",
			},
		}
	}
	return []corev1.ContainerPort{}
}
