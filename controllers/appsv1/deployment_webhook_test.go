package appsv1

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"sort"
	"testing"

	"github.com/spf13/viper"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	jsonpatch "gomodules.xyz/jsonpatch/v2"
	admissionv1 "k8s.io/api/admission/v1"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes/scheme"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"

	v1 "github.com/jaegertracing/jaeger-operator/apis/v1"
	"github.com/jaegertracing/jaeger-operator/pkg/autodetect"
	"github.com/jaegertracing/jaeger-operator/pkg/inject"
)

func TestReconcileConfigMaps(t *testing.T) {
	testCases := []struct {
		desc     string
		existing []runtime.Object
		errors   errorGroup
		expect   error
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
		{
			desc:   "can not create",
			errors: errorGroup{createErr: fmt.Errorf("ups, cant create things")},
			expect: fmt.Errorf("ups, cant create things"),
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
			dep := appsv1.Deployment{
				ObjectMeta: metav1.ObjectMeta{
					Namespace: "ns1",
					Name:      "my-dep",
				},
			}

			cl := &failingClient{
				WithWatch: fake.NewClientBuilder().WithRuntimeObjects(tC.existing...).Build(),
				errors:    tC.errors,
			}

			autodetect.OperatorConfiguration.SetPlatform(autodetect.OpenShiftPlatform)

			// test
			err := reconcileConfigMaps(context.Background(), cl, jaeger, &dep)

			// verify
			assert.Equal(t, tC.expect, err)

			cms := corev1.ConfigMapList{}
			err = cl.List(context.Background(), &cms)
			require.NoError(t, err)

			assert.Len(t, cms.Items, 2)
		})
	}
}

type failingClient struct {
	client.WithWatch

	errors errorGroup
}

type errorGroup struct {
	listErr   error
	getErr    error
	createErr error
}

func (u *failingClient) List(ctx context.Context, list client.ObjectList, opts ...client.ListOption) error {
	if u.errors.listErr != nil {
		return u.errors.listErr
	}
	return u.WithWatch.List(ctx, list, opts...)
}

func (u *failingClient) Get(ctx context.Context, key client.ObjectKey, obj client.Object, opts ...client.GetOption) error {
	if u.errors.getErr != nil {
		return u.errors.getErr
	}
	return u.WithWatch.Get(ctx, key, obj, opts...)
}

func (u *failingClient) Create(ctx context.Context, obj client.Object, opts ...client.CreateOption) error {
	if u.errors.createErr != nil {
		return u.errors.createErr
	}
	return u.WithWatch.Create(ctx, obj, opts...)
}

func TestReconcilieDeployment(t *testing.T) {
	namespacedName := types.NamespacedName{
		Name:      "jaeger-query",
		Namespace: "my-ns",
	}

	jaeger := v1.NewJaeger(types.NamespacedName{
		Namespace: "observability",
		Name:      "my-instance",
	})

	s := scheme.Scheme
	s.AddKnownTypes(v1.GroupVersion, jaeger)
	s.AddKnownTypes(v1.GroupVersion, &v1.JaegerList{})

	testCases := []struct {
		desc         string
		dep          *appsv1.Deployment
		jaeger       *v1.Jaeger
		resp         admission.Response
		errors       errorGroup
		emptyRequest bool
		watch_ns     string
	}{
		{
			desc: "no content to decode",
			dep:  &appsv1.Deployment{},
			resp: admission.Response{
				AdmissionResponse: admissionv1.AdmissionResponse{
					Allowed: false,
					Result: &metav1.Status{
						Message: "there is no content to decode",
						Code:    400,
					},
				},
			},
			emptyRequest: true,
		},
		{
			desc: "can not get namespaces and list jaegers",
			errors: errorGroup{
				listErr: fmt.Errorf("ups cant list"),
				getErr:  fmt.Errorf("ups cant get"),
			},
			dep: inject.Sidecar(jaeger, &appsv1.Deployment{
				ObjectMeta: metav1.ObjectMeta{
					Name:        namespacedName.Name,
					Namespace:   namespacedName.Namespace,
					Annotations: map[string]string{},
					Labels: map[string]string{
						"app": "not jaeger",
					},
				},
				Spec: appsv1.DeploymentSpec{
					Template: corev1.PodTemplateSpec{
						ObjectMeta: metav1.ObjectMeta{},
						Spec: corev1.PodSpec{
							Containers: []corev1.Container{{
								Name: "only_container",
							}},
						},
					},
				},
			}),
			resp: admission.Response{
				AdmissionResponse: admissionv1.AdmissionResponse{
					Allowed: false,
					Result: &metav1.Status{
						Message: "ups cant list",
						Code:    500,
					},
				},
			},
		},
		{
			desc: "Should not remove the instance from a jaeger component",
			dep: inject.Sidecar(jaeger, &appsv1.Deployment{
				ObjectMeta: metav1.ObjectMeta{
					Name:        namespacedName.Name,
					Namespace:   namespacedName.Namespace,
					Annotations: map[string]string{},
					Labels: map[string]string{
						"app": "jaeger",
					},
				},
				Spec: appsv1.DeploymentSpec{
					Template: corev1.PodTemplateSpec{
						ObjectMeta: metav1.ObjectMeta{},
						Spec: corev1.PodSpec{
							Containers: []corev1.Container{{
								Name: "only_container",
							}},
						},
					},
				},
			}),
			resp: admission.Response{
				AdmissionResponse: admissionv1.AdmissionResponse{
					Allowed: true,
					Result: &metav1.Status{
						Message: "is jaeger deployment, we do not touch it",
						Code:    200,
					},
				},
			},
		},
		{
			desc: "Should remove the instance",
			dep: inject.Sidecar(jaeger, &appsv1.Deployment{
				ObjectMeta: metav1.ObjectMeta{
					Name:        namespacedName.Name,
					Namespace:   namespacedName.Namespace,
					Annotations: map[string]string{},
				},
				Spec: appsv1.DeploymentSpec{
					Template: corev1.PodTemplateSpec{
						ObjectMeta: metav1.ObjectMeta{},
						Spec: corev1.PodSpec{
							Containers: []corev1.Container{{
								Name: "only_container",
							}},
						},
					},
				},
			}),
			resp: admission.Response{
				Patches: []jsonpatch.JsonPatchOperation{
					{
						Operation: "remove",
						Path:      "/metadata/labels",
					},
					{
						Operation: "remove",
						Path:      "/spec/template/spec/containers/1",
					},
				},
				AdmissionResponse: admissionv1.AdmissionResponse{
					Allowed:   true,
					PatchType: func() *admissionv1.PatchType { str := admissionv1.PatchTypeJSONPatch; return &str }(),
				},
			},
		},
		{
			desc: "Should inject but no jaeger instace found",
			dep: inject.Sidecar(jaeger, &appsv1.Deployment{
				ObjectMeta: metav1.ObjectMeta{
					Name:      namespacedName.Name,
					Namespace: namespacedName.Namespace,
					Annotations: map[string]string{
						inject.Annotation: "true",
					},
					Labels: map[string]string{
						"app": "something",
					},
				},
				Spec: appsv1.DeploymentSpec{
					Template: corev1.PodTemplateSpec{
						ObjectMeta: metav1.ObjectMeta{},
						Spec: corev1.PodSpec{
							Containers: []corev1.Container{{
								Name: "only_container",
							}},
						},
					},
				},
			}),
			resp: admission.Response{
				AdmissionResponse: admissionv1.AdmissionResponse{
					Allowed: true,
					Result: &metav1.Status{
						Message: "no suitable Jaeger instances found to inject a sidecar",
						Code:    200,
					},
				},
			},
		},
		{
			desc: "Should inject but empty instance - no patch",
			dep: inject.Sidecar(jaeger, &appsv1.Deployment{
				ObjectMeta: metav1.ObjectMeta{
					Name:      namespacedName.Name,
					Namespace: namespacedName.Namespace,
					Annotations: map[string]string{
						inject.Annotation: "true",
					},
					Labels: map[string]string{
						"app": "something",
					},
				},
				Spec: appsv1.DeploymentSpec{
					Template: corev1.PodTemplateSpec{
						ObjectMeta: metav1.ObjectMeta{},
						Spec: corev1.PodSpec{
							Containers: []corev1.Container{{
								Name: "only_container",
							}},
						},
					},
				},
			}),
			resp: admission.Response{
				Patches: []jsonpatch.JsonPatchOperation{},
				AdmissionResponse: admissionv1.AdmissionResponse{
					Allowed: true,
				},
			},
			jaeger: &v1.Jaeger{},
		},
		{
			desc: "should not touch deployment on other namespaces != watch_namespaces",
			dep: &appsv1.Deployment{
				ObjectMeta: metav1.ObjectMeta{
					Name:        namespacedName.Name,
					Namespace:   namespacedName.Namespace,
					Annotations: map[string]string{},
					Labels: map[string]string{
						"app": "not jaeger",
					},
				},
				Spec: appsv1.DeploymentSpec{},
			},
			resp: admission.Response{
				AdmissionResponse: admissionv1.AdmissionResponse{
					Allowed: true,
					Result: &metav1.Status{
						Message: "not watching in namespace, we do not touch the deployment",
						Code:    200,
					},
				},
			},
			watch_ns: "my-other-ns, other-ns-2",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.desc, func(t *testing.T) {
			viper.Set(v1.ConfigWatchNamespace, tc.watch_ns)
			defer viper.Reset()
			ns := &corev1.Namespace{
				ObjectMeta: metav1.ObjectMeta{
					Name: namespacedName.Namespace,
				},
			}

			res := []runtime.Object{tc.dep, ns}
			if tc.jaeger != nil {
				res = append(res, tc.jaeger)
			}

			cl := &failingClient{
				WithWatch: fake.NewClientBuilder().WithRuntimeObjects(res...).Build(),
				errors:    tc.errors,
			}

			decoder := admission.NewDecoder(scheme.Scheme)
			r := NewDeploymentInterceptorWebhook(cl, decoder)

			req := admission.Request{}
			if !tc.emptyRequest {
				req = admission.Request{
					AdmissionRequest: admissionv1.AdmissionRequest{
						Name:      tc.dep.Name,
						Namespace: tc.dep.Namespace,
						Object: runtime.RawExtension{
							Raw: func() []byte {
								var buf bytes.Buffer
								if getErr := json.NewEncoder(&buf).Encode(tc.dep); getErr != nil {
									t.Fatal(getErr)
								}
								return buf.Bytes()
							}(),
						},
					},
				}
			}

			resp := r.Handle(context.Background(), req)

			assert.Len(t, resp.Patches, len(tc.resp.Patches))
			sort.Slice(resp.Patches, func(i, j int) bool {
				return resp.Patches[i].Path < resp.Patches[j].Path
			})
			sort.Slice(tc.resp.Patches, func(i, j int) bool {
				return tc.resp.Patches[i].Path < tc.resp.Patches[j].Path
			})

			assert.Equal(t, tc.resp, resp)
		})
	}
}
