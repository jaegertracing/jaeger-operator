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

package sidecar

import (
	"strings"

	otelsidecar "github.com/open-telemetry/opentelemetry-operator/pkg/sidecar"
	appsv1 "k8s.io/api/apps/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	v2 "github.com/jaegertracing/jaeger-operator/apis/jaegertracing/v2"
	"github.com/jaegertracing/jaeger-operator/pkg/naming"
)

func TransformDeploymentAnnotation(deployment appsv1.Deployment) (appsv1.Deployment, bool) {
	// if no annotations are found at all, just return the same deployment
	depAnnValue, hasAnnotation := deployment.Annotations[Annotation]

	if !hasAnnotation {
		if _, hasOtelAnnotation := deployment.Spec.Template.Annotations[otelsidecar.Annotation]; hasOtelAnnotation {
			return removeOtelAnnotationFromDeployment(deployment), true
		}

		return deployment, false
	}

	if strings.EqualFold(depAnnValue, "true") || strings.EqualFold(depAnnValue, "false") {
		return addOtelAnnotationToDeployment(depAnnValue, deployment), true
	}

	otelCollectorName := naming.Agent(v2.Jaeger{
		ObjectMeta: metav1.ObjectMeta{
			Name: depAnnValue,
		},
	})

	return addOtelAnnotationToDeployment(otelCollectorName, deployment), true
}

// Adfd opentelemetry annotation to the deployment podSpec.
func addOtelAnnotationToDeployment(annotationValue string, deployment appsv1.Deployment) appsv1.Deployment {
	if deployment.Spec.Template.Annotations == nil {
		deployment.Spec.Template.Annotations = map[string]string{}
	}

	deployment.Spec.Template.Annotations[otelsidecar.Annotation] = annotationValue
	return deployment
}

// Remove opentelemetry annotation from the deployment podSpec.
func removeOtelAnnotationFromDeployment(deployment appsv1.Deployment) appsv1.Deployment {
	delete(deployment.Spec.Template.Annotations, otelsidecar.Annotation)
	return deployment
}
