// Copyright The Jaeger Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package strategy

import (
	otelv1alpha1 "github.com/open-telemetry/opentelemetry-operator/api/v1alpha1"
	autoscalingv2beta2 "k8s.io/api/autoscaling/v2beta2"
	corev1 "k8s.io/api/core/v1"
	rbac "k8s.io/api/rbac/v1"

	jaegertracingv2 "github.com/jaegertracing/jaeger-operator/apis/jaegertracing/v2"
)

// S knows what type of deployments to build based on a given spec.
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
