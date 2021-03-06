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

package sidecarannotation

import (
	"context"
	"encoding/json"
	"net/http"

	"github.com/go-logr/logr"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"

	"github.com/jaegertracing/jaeger-operator/internal/config"
	"github.com/jaegertracing/jaeger-operator/pkg/sidecar"
)

// +kubebuilder:webhook:path=/mutate-v1-deployment,mutating=true,failurePolicy=ignore,groups="apps",resources=deployments,verbs=create;update,versions=v1,name=mdeploy.kb.io
// +kubebuilder:rbac:groups="",resources=namespaces,verbs=list;watch
// +kubebuilder:rbac:groups=jaegertracing.io,resources=jaegers,verbs=get;list;watch

var _ DeploySidecarAnnotation = (*deploySidecarAnnotation)(nil)

// DeploySidecarAnnotation is a webhoo that convert jaeger deployment annotations to opentelemetry sidecar annotations.
type DeploySidecarAnnotation interface {
	admission.Handler
	admission.DecoderInjector
}

// the implementation.
type deploySidecarAnnotation struct {
	config  config.Config
	logger  logr.Logger
	client  client.Client
	decoder *admission.Decoder
}

// NewDeploySidecarAnnotation creates a new DeploySidecarAnnotation.
func NewDeploySidecarAnnotation(cfg config.Config, logger logr.Logger, cl client.Client) DeploySidecarAnnotation {
	return &deploySidecarAnnotation{
		config: cfg,
		logger: logger,
		client: cl,
	}
}

func (p *deploySidecarAnnotation) Handle(ctx context.Context, req admission.Request) admission.Response {
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

func (p *deploySidecarAnnotation) mutate(deployment appsv1.Deployment) appsv1.Deployment {
	logger := p.logger.WithValues("namespace", deployment.Namespace, "name", deployment.Name)

	deployment, skipped := sidecar.TransformDeploymentAnnotation(deployment)
	if skipped {
		logger.V(1).Info("annotation not present in deployment, skipping sidecar injection")
	}

	return deployment
}

func (p *deploySidecarAnnotation) InjectDecoder(d *admission.Decoder) error {
	p.decoder = d
	return nil
}
