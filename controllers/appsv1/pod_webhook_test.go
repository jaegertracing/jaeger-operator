package appsv1

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"testing"

	"github.com/spf13/viper"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	admissionv1 "k8s.io/api/admission/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	"sigs.k8s.io/controller-runtime/pkg/webhook"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"

	v1 "github.com/jaegertracing/jaeger-operator/apis/v1"
	"github.com/jaegertracing/jaeger-operator/pkg/inject"
)

func TestNewPodInjectorWebhook(t *testing.T) {
	c := NewPodInjectorWebhook(&mockClient{})
	assert.IsType(t, &podInjector{}, c)
}

func TestNewPodInjectorWebhook_Handle(t *testing.T) {
	errTest := errors.New("test")
	tests := []struct {
		name            string
		handler         webhook.AdmissionHandler
		request         admission.Request
		response        admission.Response
		numberOfPatches int // considerd if >=0
	}{
		{
			name:            "no content",
			handler:         NewPodInjectorWebhook(&mockClient{}),
			response:        admission.Errored(http.StatusBadRequest, errors.New("there is no content to decode")),
			numberOfPatches: -1,
		},
		{
			name:     "failed getting namespace",
			handler:  NewPodInjectorWebhook(&mockClient{getErr: errTest}),
			response: admission.Errored(http.StatusNotFound, errTest),
			request: admission.Request{
				AdmissionRequest: admissionv1.AdmissionRequest{
					Object: runtime.RawExtension{
						Raw: []byte("{}"),
					},
				},
			},
			numberOfPatches: -1,
		},
		{
			name:     "no action needed",
			handler:  NewPodInjectorWebhook(&mockClient{}),
			response: admission.Allowed("no action necessary"),
			request: admission.Request{
				AdmissionRequest: admissionv1.AdmissionRequest{
					Object: runtime.RawExtension{
						Raw: []byte("{}"),
					},
				},
			},
			numberOfPatches: -1,
		},
		{
			name: "sidecar needed and jaeger present",
			handler: NewPodInjectorWebhook(&mockClient{jaegers: []v1.Jaeger{
				{
					Spec: v1.JaegerSpec{
						Strategy: "something",
					},
				},
			}}),
			request: admission.Request{
				AdmissionRequest: admissionv1.AdmissionRequest{
					Object: runtime.RawExtension{
						Raw: func() []byte {
							deploy := &corev1.Pod{
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
			numberOfPatches: 4,
		},
		{
			name:     "failed to get jaeger pods",
			handler:  NewPodInjectorWebhook(&mockClient{listErr: errTest}),
			response: admission.Errored(http.StatusInternalServerError, errTest),
			request: admission.Request{
				AdmissionRequest: admissionv1.AdmissionRequest{
					Object: runtime.RawExtension{
						Raw: []byte("{}"),
					},
				},
			},
			numberOfPatches: -1,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			d, getErr := admission.NewDecoder(runtime.NewScheme())
			require.NoError(t, getErr)
			_, getErr = admission.InjectDecoderInto(d, tc.handler)
			require.NoError(t, getErr)
			response := tc.handler.Handle(context.Background(), tc.request)
			if tc.numberOfPatches > -1 {
				assert.Equal(t, tc.numberOfPatches, len(response.Patches))
			} else {
				assert.Equal(t, tc.response, response)
			}
		})
	}
}

func TestReconcileConfigMapsj(t *testing.T) {
	testCases := []struct {
		desc     string
		existing []runtime.Object
	}{
		{
			desc: "all config maps missing",
		},
		{
			desc: "none missing",
			existing: []runtime.Object{
				&corev1.ConfigMap{
					ObjectMeta: metav1.ObjectMeta{
						Namespace: "ns1",
						Name:      "my-instance-trusted-ca",
					},
				},
				&corev1.ConfigMap{
					ObjectMeta: metav1.ObjectMeta{
						Namespace: "ns1",
						Name:      "my-instance-service-ca",
					},
				},
			},
		},
	}
	for _, tC := range testCases {
		t.Run(tC.desc, func(t *testing.T) {
			// prepare
			jaeger := v1.NewJaeger(types.NamespacedName{
				Namespace: "observability",
				Name:      "my-instance",
			})
			pod := corev1.Pod{
				ObjectMeta: metav1.ObjectMeta{
					Namespace: "ns1",
					Name:      "my-dep",
				},
			}

			cl := fake.NewFakeClient(tC.existing...)

			viper.Set("platform", v1.FlagPlatformOpenShift)
			defer viper.Reset()

			// test
			err := reconcileConfigMaps(context.Background(), cl, jaeger, &pod)

			// verify
			assert.NoError(t, err)

			cms := corev1.ConfigMapList{}
			err = cl.List(context.Background(), &cms)
			require.NoError(t, err)

			assert.Len(t, cms.Items, 2)
		})
	}
}
