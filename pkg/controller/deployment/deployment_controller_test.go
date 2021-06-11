package deployment

import (
	"context"
	"sort"
	"testing"

	"github.com/spf13/viper"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes/scheme"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	v1 "github.com/jaegertracing/jaeger-operator/pkg/apis/jaegertracing/v1"
	"github.com/jaegertracing/jaeger-operator/pkg/inject"
)

func TestSyncOnJaegerChanges(t *testing.T) {
	// prepare
	jaeger := v1.NewJaeger(types.NamespacedName{
		Namespace: "observability",
		Name:      "my-instance",
	})

	objs := []runtime.Object{
		&corev1.Namespace{ObjectMeta: metav1.ObjectMeta{
			Name: "ns-with-annotation",
			Annotations: map[string]string{
				inject.Annotation: "true",
			},
		}},
		&appsv1.Deployment{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "dep-without-annotation",
				Namespace: "ns-with-annotation",
			},
		},
		&appsv1.Deployment{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "dep-with-annotation",
				Namespace: "ns-with-annotation",
				Annotations: map[string]string{
					inject.Annotation: "true",
				},
			},
		},

		&corev1.Namespace{ObjectMeta: metav1.ObjectMeta{
			Name: "ns-without-annotation",
		}},
		&appsv1.Deployment{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "dep-without-annotation",
				Namespace: "ns-without-annotation",
			},
		},
		&appsv1.Deployment{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "dep-with-annotation",
				Namespace: "ns-without-annotation",
				Annotations: map[string]string{
					inject.Annotation: "true",
				},
			},
		},
		&appsv1.Deployment{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "dep-with-another-jaegers-label",
				Namespace: "ns-without-annotation",
				Annotations: map[string]string{
					inject.Annotation: "true",
				},
				Labels: map[string]string{
					inject.Label: "some-other-jaeger",
				},
			},
		},
		&appsv1.Deployment{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "dep-affected-jaeger-label",
				Namespace: "ns-without-annotation",
				Annotations: map[string]string{
					inject.Annotation: "true",
				},
				Labels: map[string]string{
					inject.Label: jaeger.Name,
				},
			},
		},
	}

	s := scheme.Scheme
	cl := fake.NewFakeClient(objs...)
	r := &ReconcileDeployment{
		client:  cl,
		rClient: cl,
		scheme:  s,
	}

	// test
	requests := r.syncOnJaegerChanges(handler.MapObject{
		Meta:   &jaeger.ObjectMeta,
		Object: jaeger,
	})

	// verify
	assert.Len(t, requests, 4)

	expected := []reconcile.Request{
		{NamespacedName: types.NamespacedName{
			Name:      "dep-without-annotation",
			Namespace: "ns-with-annotation",
		}},
		{NamespacedName: types.NamespacedName{
			Name:      "dep-with-annotation",
			Namespace: "ns-with-annotation",
		}},
		{NamespacedName: types.NamespacedName{
			Name:      "dep-with-annotation",
			Namespace: "ns-without-annotation",
		}},
		{NamespacedName: types.NamespacedName{
			Name:      "dep-affected-jaeger-label",
			Namespace: "ns-without-annotation",
		}},
	}
	sort.Slice(requests, func(i, j int) bool {
		return requests[i].NamespacedName.String() < requests[j].NamespacedName.String()
	})
	sort.Slice(expected, func(i, j int) bool {
		return expected[i].NamespacedName.String() < expected[j].NamespacedName.String()
	})
	assert.Equal(t, expected, requests)
}

func TestReconcileConfigMaps(t *testing.T) {
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
			dep := appsv1.Deployment{
				ObjectMeta: metav1.ObjectMeta{
					Namespace: "ns1",
					Name:      "my-dep",
				},
			}

			s := scheme.Scheme
			cl := fake.NewFakeClient(tC.existing...)
			r := &ReconcileDeployment{
				client:  cl,
				rClient: cl,
				scheme:  s,
			}

			viper.Set("platform", v1.FlagPlatformOpenShift)
			defer viper.Reset()

			// test
			err := r.reconcileConfigMaps(context.Background(), jaeger, &dep)

			// verify
			assert.NoError(t, err)

			cms := corev1.ConfigMapList{}
			err = cl.List(context.Background(), &cms)
			require.NoError(t, err)

			assert.Len(t, cms.Items, 2)
		})
	}
}
