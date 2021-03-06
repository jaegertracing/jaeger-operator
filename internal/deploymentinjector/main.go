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

package deploymentinjector

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"

	"github.com/go-logr/logr"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	otelsidecar "github.com/open-telemetry/opentelemetry-operator/pkg/sidecar"

	"github.com/jaegertracing/jaeger-operator/apis/jaegertracing/v2"
	"github.com/jaegertracing/jaeger-operator/internal/config"
	"github.com/jaegertracing/jaeger-operator/pkg/naming"
	"github.com/jaegertracing/jaeger-operator/pkg/sidecar"
)

// +kubebuilder:webhook:path=/mutate-v1-deployment,mutating=true,failurePolicy=ignore,groups="apps",resources=deployments,verbs=create;update,versions=v1,name=mdeploy.kb.io
// +kubebuilder:rbac:groups="",resources=namespaces,verbs=list;watch
// +kubebuilder:rbac:groups=jaegertracing.io,resources=jaegers,verbs=get;list;watch

var _ DeploySidecarInjector = (*deploySidecarInjector)(nil)

// DeploySidecarInjector is a webhoo that convert jaeger deployment annotations to opentelemetry sidecar annotations.
type DeploySidecarInjector interface {
	admission.Handler
	admission.DecoderInjector
}

// the implementation.
type deploySidecarInjector struct {
	config  config.Config
	logger  logr.Logger
	client  client.Client
	decoder *admission.Decoder
}

// NewDeploySidecarInjector creates a new DeploySidecarInjector.
func NewDeploySidecarInjector(cfg config.Config, logger logr.Logger, cl client.Client) DeploySidecarInjector {
	return &deploySidecarInjector{
		config: cfg,
		logger: logger,
		client: cl,
	}
}

func (p *deploySidecarInjector) Handle(ctx context.Context, req admission.Request) admission.Response {
	deployment := appsv1.Deployment{}
	err := p.decoder.Decode(req, &deployment)
	if err != nil {
		return admission.Errored(http.StatusBadRequest, err)
	}

	// we use the req.Namespace here because the pod might have not been created yet
	ns := corev1.Namespace{}
	err = p.client.Get(ctx, types.NamespacedName{Name: req.Namespace, Namespace: ""}, &ns)
	if err != nil {
		return admission.Errored(http.StatusInternalServerError, err)
	}

	deployment = p.mutate(deployment)
	marshaledDeployment, err := json.Marshal(deployment)
	if err != nil {
		return admission.Errored(http.StatusInternalServerError, err)
	}
	return admission.PatchResponseFromRaw(req.Object.Raw, marshaledDeployment)
}

func (p *deploySidecarInjector) mutate(deployment appsv1.Deployment) appsv1.Deployment {
	logger := p.logger.WithValues("namespace", deployment.Namespace, "name", deployment.Name)

	// if no annotations are found at all, just return the same deployment
	depAnnValue, hasAnnotation := deployment.Annotations[sidecar.Annotation]

	if !hasAnnotation {
		if _, hasOtelAnnotation := deployment.Spec.Template.Annotations[otelsidecar.Annotation]; hasOtelAnnotation {
			return removeOpentelemetryAnnotation(deployment)
		}

		logger.V(1).Info("annotation not present in deployment, skipping sidecar injection")
		return deployment
	}

	logger.Info("annotation " + depAnnValue)

	// is the annotation value 'false'? if so, we need a pod without the sidecar (ie, remove if exists)
	if strings.EqualFold(depAnnValue, "false") {
		logger.V(1).Info("deployment explicitly refuses sidecar injection, attempting to remove sidecar if it exists")
		return removeOpentelemetryAnnotation(deployment)
	}

	if strings.EqualFold(depAnnValue, "true") || strings.EqualFold(depAnnValue, "false") {
		return addOpentelemetryAnnotation(depAnnValue, deployment)
	}

	otelCollectorName := naming.Agent(v2.Jaeger{
		ObjectMeta: metav1.ObjectMeta{
			Name: depAnnValue,
		},
	})

	return addOpentelemetryAnnotation(otelCollectorName, deployment)
}

func (p *deploySidecarInjector) InjectDecoder(d *admission.Decoder) error {
	p.decoder = d
	return nil
}

func addOpentelemetryAnnotation(annotationValue string, deployment appsv1.Deployment) appsv1.Deployment {
	// add opentelemetry annotation to template
	if deployment.Spec.Template.Annotations == nil {
		deployment.Spec.Template.Annotations = map[string]string{}
	}

	deployment.Spec.Template.Annotations[otelsidecar.Annotation] = annotationValue
	return deployment
}

// Remove the sidecar container from the given deployment.
func removeOpentelemetryAnnotation(deployment appsv1.Deployment) appsv1.Deployment {
	delete(deployment.Spec.Template.Annotations, otelsidecar.Annotation)
	return deployment
}
