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
	"errors"
	"strings"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"

	v2 "github.com/jaegertracing/jaeger-operator/apis/jaegertracing/v2"
)

var (
	// Annotation is the annotation name to look for when deciding whether or not to inject.
	Annotation = "sidecar.jaegertracing.io/inject"
	// Label is the label name the operator put on injected deployments.
	Label                        = "sidecar.jaegertracing.io/injected"
	ErrMultipleInstancesPossible = errors.New("multiple Jaeger instances available, cannot determine which one to select")
	ErrNoInstancesAvailable      = errors.New("no Jaeger instances available")
)

// Needed determines whether a pod needs to get a sidecar injected or not.
func AnnotationValue(dep appsv1.Deployment, ns corev1.Namespace) string {
	// is the pod annotated with instructions to inject sidecars? is the namespace annotated?
	// if any of those is true, a sidecar might be desired.
	depAnnValue := dep.Annotations[Annotation]
	nsAnnValue := ns.Annotations[Annotation]

	// if the namespace value is empty, the pod annotation should be used, whatever it is
	if len(nsAnnValue) == 0 {
		return depAnnValue
	}

	// if the pod value is empty, the annotation annotation should be used (true, false, instance)
	if len(depAnnValue) == 0 {
		return nsAnnValue
	}

	// the pod annotation isn't empty -- if it's an instance name, or false, that's the decision
	if !strings.EqualFold(depAnnValue, "true") {
		return depAnnValue
	}

	// pod annotation is 'true', and if the namespace annotation is false, we just return 'true'
	if strings.EqualFold(nsAnnValue, "false") {
		return depAnnValue
	}

	// by now, the pod annotation is 'true', and the namespace annotation is either true or an instance name
	// so, the namespace annotation can be used
	return nsAnnValue
}

// Select a suitable Jaeger from the JaegerList for the given Pod, or nil of none is suitable.
func Select(annotationValue string, namespace corev1.Namespace, candidates []v2.Jaeger) (v2.Jaeger, error) {

	if strings.EqualFold(annotationValue, "true") {
		if len(candidates) == 1 {
			jaeger := candidates[0]
			return jaeger, nil
		}
		instancesInNamespace := filterJaegerByNamespace(namespace.Name, candidates)
		if len(instancesInNamespace) == 1 {
			jaeger := instancesInNamespace[0]
			return jaeger, nil
		}
		// At this point, we have more than one instance that could be used to inject
		// we should just not inject, as it's not clear which one should be used.
		return v2.Jaeger{}, ErrMultipleInstancesPossible
	}

	if jaeger := filterJaegerByName(annotationValue, candidates); jaeger != nil {
		return *jaeger, nil
	}

	return v2.Jaeger{}, ErrNoInstancesAvailable
}

func filterJaegerByName(name string, jaegers []v2.Jaeger) *v2.Jaeger {
	for _, p := range jaegers {
		if p.Name == name {
			// matched the name!
			return &p
		}
	}
	return nil
}

func filterJaegerByNamespace(namespace string, jaegers []v2.Jaeger) []v2.Jaeger {
	var instances []v2.Jaeger
	for _, p := range jaegers {
		if p.Namespace == namespace {
			// matched the namespace!
			instances = append(instances, p)
		}
	}
	return instances
}
