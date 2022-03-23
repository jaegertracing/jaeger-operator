package appsv1

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	jsonpatch "gomodules.xyz/jsonpatch/v2"
	admissionv1 "k8s.io/api/admission/v1"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/webhook"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"

	"github.com/jaegertracing/jaeger-operator/pkg/inject"

	v1 "github.com/jaegertracing/jaeger-operator/apis/v1"
)

var _ client.Client = (*mockClient)(nil)

type mockClient struct {
	client.Client
	getErr  error
	listErr error
	jaegers []v1.Jaeger
}

func (m *mockClient) Get(ctx context.Context, key client.ObjectKey, obj client.Object) error {
	return m.getErr
}

func (m *mockClient) List(ctx context.Context, list client.ObjectList, opts ...client.ListOption) error {
	jl, ok := list.(*v1.JaegerList)
	if ok {
		jl.Items = append(jl.Items, m.jaegers...)
	}
	return m.listErr
}

func TestNewDeploymentInterceptorWebhook(t *testing.T) {
	c := NewDeploymentInterceptorWebhook(&mockClient{})
	assert.IsType(t, &deploymentInterceptor{}, c)
}

func TestDeploymentInterceptor_Handle(t *testing.T) {
	errTest := errors.New("test")
	tests := []struct {
		name     string
		handler  webhook.AdmissionHandler
		request  admission.Request
		response admission.Response
	}{
		{
			name:     "no content",
			handler:  NewDeploymentInterceptorWebhook(&mockClient{}),
			response: admission.Errored(http.StatusBadRequest, errors.New("there is no content to decode")),
		},
		{
			name:     "failed getting namespace",
			handler:  NewDeploymentInterceptorWebhook(&mockClient{getErr: errTest}),
			response: admission.Errored(http.StatusNotFound, errTest),
			request: admission.Request{
				AdmissionRequest: admissionv1.AdmissionRequest{
					Object: runtime.RawExtension{
						Raw: []byte("{}"),
					},
				},
			},
		},
		{
			name:     "no action needed",
			handler:  NewDeploymentInterceptorWebhook(&mockClient{}),
			response: admission.Allowed("no need to update PodTemplateSpec"),
			request: admission.Request{
				AdmissionRequest: admissionv1.AdmissionRequest{
					Object: runtime.RawExtension{
						Raw: []byte("{}"),
					},
				},
			},
		},
		{
			name:    "deployment needed",
			handler: NewDeploymentInterceptorWebhook(&mockClient{}),
			response: admission.Response{
				Patches: []jsonpatch.JsonPatchOperation{
					{
						Operation: "add",
						Path:      "/spec/template/metadata/annotations",
						Value: map[string]interface{}{
							"sidecar.jaegertracing.io/inject": "true",
						},
					},
				},
				AdmissionResponse: admissionv1.AdmissionResponse{
					Allowed: true,
					PatchType: func() *admissionv1.PatchType {
						pt := admissionv1.PatchTypeJSONPatch
						return &pt
					}(),
				},
			},
			request: admission.Request{
				AdmissionRequest: admissionv1.AdmissionRequest{
					Object: runtime.RawExtension{
						Raw: func() []byte {
							deploy := &appsv1.Deployment{
								ObjectMeta: metav1.ObjectMeta{
									Annotations: map[string]string{
										inject.Annotation: "true",
									},
								},
							}
							var buf bytes.Buffer
							if getErr := json.NewEncoder(&buf).Encode(deploy); getErr != nil {
								t.Fatal(getErr)
							}
							return buf.Bytes()
						}(),
					},
				},
			},
		},
		{
			name:    "pod has inject annotation too",
			handler: NewDeploymentInterceptorWebhook(&mockClient{}),
			response: admission.Response{
				Patches: []jsonpatch.JsonPatchOperation{},
				AdmissionResponse: admissionv1.AdmissionResponse{
					Allowed: true,
				},
			},
			request: admission.Request{
				AdmissionRequest: admissionv1.AdmissionRequest{
					Object: runtime.RawExtension{
						Raw: func() []byte {
							deploy := &appsv1.Deployment{
								ObjectMeta: metav1.ObjectMeta{
									Annotations: map[string]string{
										inject.Annotation: "true",
									},
								},
								Spec: appsv1.DeploymentSpec{
									Template: corev1.PodTemplateSpec{
										ObjectMeta: metav1.ObjectMeta{
											Annotations: map[string]string{
												inject.Annotation: "true",
											},
										},
									},
								},
							}
							var buf bytes.Buffer
							if getErr := json.NewEncoder(&buf).Encode(deploy); getErr != nil {
								t.Fatal(getErr)
							}
							return buf.Bytes()
						}(),
					},
				},
			},
		},
		{
			name:     "deployment annotation false",
			handler:  NewDeploymentInterceptorWebhook(&mockClient{}),
			response: admission.Allowed("no need to update PodTemplateSpec"),
			request: admission.Request{
				AdmissionRequest: admissionv1.AdmissionRequest{
					Object: runtime.RawExtension{
						Raw: func() []byte {
							deploy := &appsv1.Deployment{
								ObjectMeta: metav1.ObjectMeta{
									Annotations: map[string]string{
										inject.Annotation: "false",
									},
								},
							}
							var buf bytes.Buffer
							if getErr := json.NewEncoder(&buf).Encode(deploy); getErr != nil {
								t.Fatal(getErr)
							}
							return buf.Bytes()
						}(),
					},
				},
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			d, getErr := admission.NewDecoder(runtime.NewScheme())
			require.NoError(t, getErr)
			_, getErr = admission.InjectDecoderInto(d, tc.handler)
			require.NoError(t, getErr)
			response := tc.handler.Handle(context.Background(), tc.request)
			assert.Equal(t, tc.response, response)
		})
	}
}
