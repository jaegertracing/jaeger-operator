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

	"k8s.io/apimachinery/pkg/types"

	v2 "github.com/jaegertracing/jaeger-operator/apis/jaegertracing/v2"

	"github.com/stretchr/testify/assert"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestEffectiveAnnotationValue(t *testing.T) {
	for _, tt := range []struct {
		desc       string
		expected   string
		deployment appsv1.Deployment
		ns         corev1.Namespace
	}{
		{
			"pod-true-overrides-ns",
			"true",
			appsv1.Deployment{
				ObjectMeta: metav1.ObjectMeta{
					Annotations: map[string]string{
						Annotation: "true",
					},
				},
			},
			corev1.Namespace{
				ObjectMeta: metav1.ObjectMeta{
					Annotations: map[string]string{
						Annotation: "false",
					},
				},
			},
		},

		{
			"ns-has-concrete-instance",
			"some-instance",
			appsv1.Deployment{
				ObjectMeta: metav1.ObjectMeta{
					Annotations: map[string]string{
						Annotation: "true",
					},
				},
			},
			corev1.Namespace{
				ObjectMeta: metav1.ObjectMeta{
					Annotations: map[string]string{
						Annotation: "some-instance",
					},
				},
			},
		},

		{
			"pod-has-concrete-instance",
			"some-instance-from-pod",
			appsv1.Deployment{
				ObjectMeta: metav1.ObjectMeta{
					Annotations: map[string]string{
						Annotation: "some-instance-from-pod",
					},
				},
			},
			corev1.Namespace{
				ObjectMeta: metav1.ObjectMeta{
					Annotations: map[string]string{
						Annotation: "some-instance",
					},
				},
			},
		},

		{
			"pod-has-explicit-false",
			"false",
			appsv1.Deployment{
				ObjectMeta: metav1.ObjectMeta{
					Annotations: map[string]string{
						Annotation: "false",
					},
				},
			},
			corev1.Namespace{
				ObjectMeta: metav1.ObjectMeta{
					Annotations: map[string]string{
						Annotation: "some-instance",
					},
				},
			},
		},

		{
			"pod-has-no-annotations",
			"some-instance",
			appsv1.Deployment{},
			corev1.Namespace{
				ObjectMeta: metav1.ObjectMeta{
					Annotations: map[string]string{
						Annotation: "some-instance",
					},
				},
			},
		},

		{
			"ns-has-no-annotations",
			"true",
			appsv1.Deployment{
				ObjectMeta: metav1.ObjectMeta{
					Annotations: map[string]string{
						Annotation: "true",
					},
				},
			},
			corev1.Namespace{},
		},
	} {
		t.Run(tt.desc, func(t *testing.T) {
			// test
			annValue := AnnotationValue(tt.deployment, tt.ns)

			// verify
			assert.Equal(t, tt.expected, annValue)
		})
	}
}

func TestSelect(t *testing.T) {
	for _, tt := range []struct {
		desc              string
		annotation        string
		namespace         corev1.Namespace
		candidates        []v2.Jaeger
		expectedCandidate v2.Jaeger
		expectedError     error
	}{
		{

			"Annotation true and a single instance",
			"true",
			corev1.Namespace{
				ObjectMeta: metav1.ObjectMeta{
					Name: "ns-test",
				},
			},
			[]v2.Jaeger{
				*v2.NewJaeger(types.NamespacedName{Namespace: "ns-test", Name: "selected-candidate"}),
			},
			*v2.NewJaeger(
				types.NamespacedName{Namespace: "ns-test", Name: "selected-candidate"},
			),
			nil,
		},
		{

			"Annotation true and a multiple instances, one on the same namespace",
			"true",
			corev1.Namespace{
				ObjectMeta: metav1.ObjectMeta{
					Name: "ns-test",
				},
			},
			[]v2.Jaeger{
				*v2.NewJaeger(types.NamespacedName{Namespace: "ns-test", Name: "selected-candidate"}),
				*v2.NewJaeger(types.NamespacedName{Namespace: "ns-test-other", Name: "other-candidate"}),
			},
			*v2.NewJaeger(
				types.NamespacedName{Namespace: "ns-test", Name: "selected-candidate"},
			),
			nil,
		},
		{
			"Annotation true and a multiple instances, both on the same namespace",
			"true",
			corev1.Namespace{
				ObjectMeta: metav1.ObjectMeta{
					Name: "ns-test",
				},
			},
			[]v2.Jaeger{
				*v2.NewJaeger(types.NamespacedName{Namespace: "ns-test", Name: "selected-candidate"}),
				*v2.NewJaeger(types.NamespacedName{Namespace: "ns-test", Name: "other-candidate"}),
			},
			v2.Jaeger{},
			ErrMultipleInstancesPossible,
		},
		{
			"Annotation name of an instance",
			"selected-candidate",
			corev1.Namespace{
				ObjectMeta: metav1.ObjectMeta{
					Name: "ns-test",
				},
			},
			[]v2.Jaeger{
				*v2.NewJaeger(types.NamespacedName{Namespace: "ns-test", Name: "selected-candidate"}),
				*v2.NewJaeger(types.NamespacedName{Namespace: "ns-test", Name: "other-candidate"}),
			},
			*v2.NewJaeger(types.NamespacedName{Namespace: "ns-test", Name: "selected-candidate"}),
			nil,
		},
		{
			"Annotation name of an instance but two instances with same name on different namespaces",
			"selected-candidate",
			corev1.Namespace{
				ObjectMeta: metav1.ObjectMeta{
					Name: "ns-test",
				},
			},
			[]v2.Jaeger{
				*v2.NewJaeger(types.NamespacedName{Namespace: "ns-test", Name: "selected-candidate"}),
				*v2.NewJaeger(types.NamespacedName{Namespace: "ns-test-other", Name: "selected-candidate"}),
			},
			*v2.NewJaeger(types.NamespacedName{Namespace: "ns-test", Name: "selected-candidate"}),
			nil,
		},
		{
			"Annotation name of a non existing instance.",
			"non-existing-jaeger",
			corev1.Namespace{
				ObjectMeta: metav1.ObjectMeta{
					Name: "ns-test",
				},
			},
			[]v2.Jaeger{
				*v2.NewJaeger(types.NamespacedName{Namespace: "ns-test", Name: "selected-candidate"}),
				*v2.NewJaeger(types.NamespacedName{Namespace: "ns-test-other", Name: "selected-candidate"}),
			},
			v2.Jaeger{},
			ErrNoInstancesAvailable,
		},
	} {
		t.Run(tt.desc, func(t *testing.T) {
			candidate, err := Select(tt.annotation, tt.namespace, tt.candidates)
			assert.Equal(t, tt.expectedError, err)
			assert.Equal(t, tt.expectedCandidate, candidate)
		})

	}
}
