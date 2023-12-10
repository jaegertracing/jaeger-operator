package namespace

import (
	"context"
	"fmt"
	"strconv"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes/scheme"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	v1 "github.com/jaegertracing/jaeger-operator/apis/v1"
	"github.com/jaegertracing/jaeger-operator/pkg/inject"
)

type bundle struct {
	dep              *appsv1.Deployment
	expectedRevision int
}

type failingClient struct {
	client.Client
	err error
}

func (f *failingClient) Update(ctx context.Context, obj client.Object, opts ...client.UpdateOption) error {
	if f.err != nil {
		return f.err
	}
	return f.Client.Update(ctx, obj, opts...)
}

func TestReconcilieDeployment(t *testing.T) {
	const depNamespace = "my-ns"

	jaeger := v1.NewJaeger(types.NamespacedName{
		Namespace: "observability",
		Name:      "my-instance",
	})

	s := scheme.Scheme
	s.AddKnownTypes(v1.GroupVersion, jaeger)
	s.AddKnownTypes(v1.GroupVersion, &v1.JaegerList{})

	errReconcile := fmt.Errorf("no no reconcile")

	testCases := []struct {
		desc         string
		bundle       []bundle
		errReconcile error
	}{
		{
			desc: "Should set annotations to reevaluate deployments",
			bundle: []bundle{
				{
					expectedRevision: -1,
					dep: &appsv1.Deployment{
						ObjectMeta: metav1.ObjectMeta{
							Name:        "jaeger-query",
							Namespace:   depNamespace,
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
					},
				},
				{
					expectedRevision: 0,
					dep: &appsv1.Deployment{
						ObjectMeta: metav1.ObjectMeta{
							Name:      "app1",
							Namespace: depNamespace,
							Annotations: map[string]string{
								inject.Annotation: "true",
							},
							Labels: map[string]string{
								"app": "app1",
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
					},
				},
				{
					expectedRevision: 6,
					dep: &appsv1.Deployment{
						ObjectMeta: metav1.ObjectMeta{
							Name:      "app2",
							Namespace: depNamespace,
							Annotations: map[string]string{
								inject.AnnotationRev: "5",
								inject.Annotation:    "true",
							},
							Labels: map[string]string{
								"app": "app2",
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
					},
				},
			},
		},
		{
			desc:         "Should fail updating deployment",
			errReconcile: errReconcile,
			bundle: []bundle{
				{
					expectedRevision: 6,
					dep: &appsv1.Deployment{
						ObjectMeta: metav1.ObjectMeta{
							Name:      "app2",
							Namespace: depNamespace,
							Annotations: map[string]string{
								inject.AnnotationRev: "5",
								inject.Annotation:    "true",
							},
							Labels: map[string]string{
								"app": "app2",
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
					},
				},
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.desc, func(t *testing.T) {
			ns := &corev1.Namespace{
				ObjectMeta: metav1.ObjectMeta{
					Name: depNamespace,
				},
			}

			objs := []runtime.Object{ns}
			for _, b := range tc.bundle {
				objs = append(objs, b.dep)
			}
			cl := fake.NewClientBuilder().WithScheme(s).WithRuntimeObjects(objs...).Build()
			r := &ReconcileNamespace{
				client:  &failingClient{Client: cl, err: tc.errReconcile},
				rClient: cl,
				scheme:  s,
			}
			req := reconcile.Request{
				NamespacedName: types.NamespacedName{
					Name: depNamespace,
				},
			}
			_, err := r.Reconcile(req)
			assert.Equal(t, tc.errReconcile, err)
			if tc.errReconcile != nil {
				return
			}

			persisted := &appsv1.DeploymentList{}
			require.NoError(t, cl.List(context.Background(), persisted))

			for _, p := range persisted.Items {
				const notFound = -2
				expectedRevision := notFound
				appName, ok := p.Labels["app"]
				assert.True(t, ok)
				for _, b := range tc.bundle {
					name, ok := b.dep.Labels["app"]
					assert.True(t, ok)
					if appName == name {
						expectedRevision = b.expectedRevision
						break
					}
				}
				if expectedRevision == notFound {
					t.Fatal("app not found")
				}

				revStr, ok := p.Annotations[inject.AnnotationRev]
				if expectedRevision == -1 {
					assert.False(t, ok)
				} else {
					assert.True(t, ok)
					rev, err := strconv.Atoi(revStr)
					require.NoError(t, err)
					assert.Equal(t, expectedRevision, rev)
				}
			}
		})
	}
}
