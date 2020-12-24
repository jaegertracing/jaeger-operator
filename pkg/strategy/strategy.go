package strategy

import (
	jaegertracingv2 "github.com/jaegertracing/jaeger-operator/apis/jaegertracing/v2"
	otelv1alpha1 "github.com/open-telemetry/opentelemetry-operator/api/v1alpha1"
	autoscalingv2beta2 "k8s.io/api/autoscaling/v2beta2"
	corev1 "k8s.io/api/core/v1"
	rbac "k8s.io/api/rbac/v1"
)

// S knows what type of deployments to build based on a given spec
type Strategy struct {
	Type                     jaegertracingv2.DeploymentStrategy
	Accounts                 []corev1.ServiceAccount
	ClusterRoleBindings      []rbac.ClusterRoleBinding
	ConfigMaps               []corev1.ConfigMap
	HorizontalPodAutoscalers []autoscalingv2beta2.HorizontalPodAutoscaler
	Services                 []corev1.Service
	Secrets                  []corev1.Secret
	Collector                otelv1alpha1.OpenTelemetryCollector
}
