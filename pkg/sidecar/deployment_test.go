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
	"testing"

	"github.com/open-telemetry/opentelemetry-operator/pkg/sidecar"

	v2 "github.com/jaegertracing/jaeger-operator/apis/jaegertracing/v2"

	"github.com/stretchr/testify/assert"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
)

var logger = logf.Log.WithName("unit-tests")

func TestAddSidecarWhenNoSidecarExists(t *testing.T) {
	// prepare
	deployment := appsv1.Deployment{
		Spec: appsv1.DeploymentSpec{
			Template: corev1.PodTemplateSpec{
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{
						{Name: "my-app"},
					},
				},
			},
		},
	}
	jaeger := v2.Jaeger{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "jaeger-sample",
			Namespace: "some-app",
		},
	}

	// test
	changed := Add(logger, jaeger, deployment)

	// verify
	assert.Equal(t, "jaeger-sample-agent", changed.Spec.Template.Annotations[sidecar.Annotation])
}

func TestAddSidecarWhenOneExistsAlready(t *testing.T) {
	// prepare
	deployment := appsv1.Deployment{
		Spec: appsv1.DeploymentSpec{
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Annotations: map[string]string{
						sidecar.Annotation: "jaeger-sample-agent",
						"other-annotation": "other-value",
					},
				},
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{
						{Name: "my-app"},
					},
				},
			},
		},
	}
	jaeger := v2.Jaeger{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "jaeger-sample",
			Namespace: "some-app",
		},
	}

	// test
	changed := Add(logger, jaeger, deployment)

	// verify
	assert.Len(t, changed.Spec.Template.Annotations, 2)
	assert.Equal(t, "jaeger-sample-agent", changed.Spec.Template.Annotations[sidecar.Annotation])

}

func TestRemoveSidecar(t *testing.T) {
	// prepare
	deployment := appsv1.Deployment{
		Spec: appsv1.DeploymentSpec{
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Annotations: map[string]string{
						sidecar.Annotation: "jaeger-sample-agent",
						"other-annotation": "other-value",
					},
				},
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{
						{Name: "my-app"},
					},
				},
			},
		},
	}

	// test
	changed := Remove(deployment)

	// verify
	assert.Len(t, changed.Spec.Template.Annotations, 1)
	_, hasAnnotation := changed.Spec.Template.Annotations[sidecar.Annotation]
	assert.False(t, hasAnnotation)
}

func TestRemoveNonExistingSidecar(t *testing.T) {
	// prepare
	deployment := appsv1.Deployment{
		Spec: appsv1.DeploymentSpec{
			Template: corev1.PodTemplateSpec{
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{
						{Name: "my-app"},
					},
				},
			},
		},
	}

	// test
	changed := Remove(deployment)

	// verify
	assert.Len(t, changed.Spec.Template.Annotations, 0)
}
