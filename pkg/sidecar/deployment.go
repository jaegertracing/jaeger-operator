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
	"github.com/go-logr/logr"
	"github.com/open-telemetry/opentelemetry-operator/pkg/sidecar"
	appsv1 "k8s.io/api/apps/v1"

	v2 "github.com/jaegertracing/jaeger-operator/apis/jaegertracing/v2"
	"github.com/jaegertracing/jaeger-operator/pkg/naming"
)

// Add a new sidecar container to the given deployment, based on the given Jaeger instance.
func Add(logger logr.Logger, jaeger v2.Jaeger, deployment appsv1.Deployment) appsv1.Deployment {
	// add opentelemetry annotation to template
	if deployment.Spec.Template.Annotations == nil {
		deployment.Spec.Template.Annotations = map[string]string{}
	}
	deployment.Spec.Template.Annotations[sidecar.Annotation] = naming.Agent(jaeger)
	return deployment
}

// Remove the sidecar container from the given deployment.
func Remove(deployment appsv1.Deployment) appsv1.Deployment {
	delete(deployment.Spec.Template.Annotations, sidecar.Annotation)
	return deployment
}
