package namespace

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes/scheme"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	v1 "github.com/jaegertracing/jaeger-operator/apis/jaegertracing/v1"
	"github.com/jaegertracing/jaeger-operator/pkg/inject"
)

func TestReconcilieDeployment(t *testing.T) {
	depNamespacedName := types.NamespacedName{
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
		desc              string
		dep               *appsv1.Deployment
		expectedContiners int
	}{
		{
			desc: "Should not remove the instance from a jaeger component",
			dep: inject.Sidecar(jaeger, &appsv1.Deployment{
				ObjectMeta: metav1.ObjectMeta{
					Name:        depNamespacedName.Name,
					Namespace:   depNamespacedName.Namespace,
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
			expectedContiners: 2,
		},
		{
			desc: "Should remove the instance",
			dep: inject.Sidecar(jaeger, &appsv1.Deployment{
				ObjectMeta: metav1.ObjectMeta{
					Name:        depNamespacedName.Name,
					Namespace:   depNamespacedName.Namespace,
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
			expectedContiners: 1,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.desc, func(t *testing.T) {

			assert.Equal(t, 2, len(tc.dep.Spec.Template.Spec.Containers))
			ns := &corev1.Namespace{
				ObjectMeta: metav1.ObjectMeta{
					Name: depNamespacedName.Namespace,
				},
			}

			cl := fake.NewFakeClient(tc.dep, ns)
			r := &ReconcileNamespace{
				client:  cl,
				rClient: cl,
				scheme:  s,
			}
			req := reconcile.Request{
				NamespacedName: types.NamespacedName{
					Name: depNamespacedName.Namespace,
				},
			}

			_, err := r.Reconcile(req)
			persisted := &appsv1.Deployment{}
			cl.Get(context.Background(), depNamespacedName, persisted)

			assert.Equal(t, tc.expectedContiners, len(persisted.Spec.Template.Spec.Containers))

			require.NoError(t, err)
		})
	}
}
