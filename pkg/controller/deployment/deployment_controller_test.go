package deployment

import (
	"sort"
	"testing"

	v1 "github.com/jaegertracing/jaeger-operator/pkg/apis/jaegertracing/v1"
	"github.com/jaegertracing/jaeger-operator/pkg/inject"
	"github.com/stretchr/testify/assert"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes/scheme"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
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
		return requests[i].Namespace < requests[j].Namespace && requests[i].Name < requests[j].Name
	})

	assert.Equal(t, expected, requests)
}
