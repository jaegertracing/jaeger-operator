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

package deployinjector

import (
	"context"
	"encoding/json"
	"strings"

	"github.com/go-logr/logr"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"

	v2 "github.com/jaegertracing/jaeger-operator/apis/jaegertracing/v2"
	"github.com/jaegertracing/jaeger-operator/internal/config"
	"github.com/jaegertracing/jaeger-operator/pkg/sidecar"

	"net/http"

	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"
)

// +kubebuilder:webhook:path=/mutate-v1-deployment,mutating=true,failurePolicy=ignore,groups="apps",resources=deployments,verbs=create;update,versions=v1,name=mdeploy.kb.io
// +kubebuilder:rbac:groups="",resources=namespaces,verbs=list;watch
// +kubebuilder:rbac:groups=jaegertracing.io,resources=jaegers,verbs=get;list;watch

var _ DeploySidecarInjector = (*deploySidecarInjector)(nil)

// DeploySidecarInjector is a webhook handler that analyzes new pods and injects appropriate sidecars into it.
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

// NewPodSidecarInjector creates a new DeploySidecarInjector.
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

	deployment, err = p.mutate(ctx, ns, deployment)
	if err != nil {
		return admission.Errored(http.StatusInternalServerError, err)
	}

	marshaledDeployment, err := json.Marshal(deployment)
	if err != nil {
		return admission.Errored(http.StatusInternalServerError, err)
	}
	return admission.PatchResponseFromRaw(req.Object.Raw, marshaledDeployment)
}

func (p *deploySidecarInjector) mutate(ctx context.Context, ns corev1.Namespace, deployment appsv1.Deployment) (appsv1.Deployment, error) {
	logger := p.logger.WithValues("namespace", deployment.Namespace, "name", deployment.Name)

	// if no annotations are found at all, just return the same deployment
	annValue := sidecar.AnnotationValue(deployment, ns)

	if len(annValue) == 0 {
		logger.V(1).Info("annotation not present in deployment, skipping sidecar injection")
		return deployment, nil
	}

	// is the annotation value 'false'? if so, we need a pod without the sidecar (ie, remove if exists)
	if strings.EqualFold(annValue, "false") {
		logger.V(1).Info("deployment explicitly refuses sidecar injection, attempting to remove sidecar if it exists")
		return sidecar.Remove(deployment)
	}

	// from this point and on, a sidecar is wanted

	// which instance should it talk to?
	candidates, err := p.getCandidates(ctx, ns, annValue)
	if err != nil {
		// we still allow the pod to be created, but we log a message to the operator's logs
		logger.Error(err, "failed to get a Jaeger instances list for this deployment's sidecar")
		return deployment, err
	}

	jaeger, err := sidecar.Select(annValue, ns, candidates)
	if err != nil {
		if err == sidecar.ErrMultipleInstancesPossible || err == sidecar.ErrNoInstancesAvailable {
			// we still allow the deployment to be created, but we log a message to the operator's logs
			logger.Error(err, "failed to select a Jaeger instance for this deployment's sidecar")
			return deployment, nil
		}

		// something else happened, better fail here
		return deployment, err
	}

	// once it's been determined that a sidecar is desired, none exists yet, and we know which instance it should talk to,
	// we should add the sidecar.
	logger.V(1).Info("injecting sidecar into pod", "jaeger-namespace", jaeger.Namespace, "jaeger-name", jaeger.Name)
	return sidecar.Add(p.logger, jaeger, deployment)
}

func (p *deploySidecarInjector) InjectDecoder(d *admission.Decoder) error {
	p.decoder = d
	return nil
}

func (p *deploySidecarInjector) getCandidates(ctx context.Context, ns corev1.Namespace, ann string) ([]v2.Jaeger, error) {
	if strings.EqualFold(ann, "true") {
		jaegers := v2.JaegerList{}
		if err := p.client.List(ctx, &jaegers, client.InNamespace(ns.Name)); err != nil {
			return []v2.Jaeger{}, err
		}
		return jaegers.Items, nil
	}
	// TODO: For now we only return Jaeger instances in the same namespace
	//		 we need to change this as soon as opentelemetry-operator support cross namespace sidecar injection.
	jaeger := v2.Jaeger{}
	err := p.client.Get(ctx, types.NamespacedName{Name: ann, Namespace: ns.Name}, &jaeger)
	if err != nil {
		return []v2.Jaeger{}, err
	}
	return []v2.Jaeger{jaeger}, err

}
