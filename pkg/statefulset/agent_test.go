package statefulset

import (
	"testing"

	"github.com/spf13/viper"
	"github.com/stretchr/testify/assert"
	appsv1 "k8s.io/api/apps/v1"
	"k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/jaegertracing/jaeger-operator/pkg/apis/io/v1alpha1"
)

func setDefaults() {
	viper.SetDefault("jaeger-version", "1.7")
	viper.SetDefault("jaeger-agent-image", "jaegertracing/jaeger-agent")
}

func init() {
	setDefaults()
}

func reset() {
	viper.Reset()
	setDefaults()
}

func TestInjectSidecar(t *testing.T) {
	jaeger := v1alpha1.NewJaeger("TestInjectSidecar")
	sset := sset(map[string]string{Annotation: jaeger.Name})
	Sidecar(sset, jaeger)
	assert.Len(t, sset.Spec.Template.Spec.Containers, 2)
	assert.Contains(t, sset.Spec.Template.Spec.Containers[1].Image, "jaeger-agent")
}

func TestSkipInjectSidecar(t *testing.T) {
	jaeger := v1alpha1.NewJaeger("TestSkipInjectSidecar")
	sset := sset(map[string]string{Annotation: "non-existing-operator"})
	Sidecar(sset, jaeger)
	assert.Len(t, sset.Spec.Template.Spec.Containers, 1)
	assert.NotContains(t, sset.Spec.Template.Spec.Containers[0].Image, "jaeger-agent")
}

func TestSidecarNotNeeded(t *testing.T) {
	sset := &appsv1.StatefulSet{
		Spec: appsv1.StatefulSetSpec{
			Template: v1.PodTemplateSpec{
				Spec: v1.PodSpec{
					Containers: []v1.Container{
						v1.Container{},
					},
				},
			},
		},
	}

	assert.False(t, Needed(sset))
}

func TestSidecarNeeded(t *testing.T) {
	sset := sset(map[string]string{Annotation: "some-jaeger-instance"})
	assert.True(t, Needed(sset))
}

func TestHasSidecarAlready(t *testing.T) {
	sset := sset(map[string]string{Annotation: "TestHasSidecarAlready"})
	assert.True(t, Needed(sset))
	jaeger := v1alpha1.NewJaeger("TestHasSidecarAlready")
	Sidecar(sset, jaeger)
	assert.False(t, Needed(sset))
}

func TestSelectSingleJaegerPod(t *testing.T) {
	sset := sset(map[string]string{Annotation: "true"})
	jaegerPods := &v1alpha1.JaegerList{
		Items: []v1alpha1.Jaeger{
			v1alpha1.Jaeger{
				ObjectMeta: metav1.ObjectMeta{
					Name: "the-only-jaeger-instance-available",
				},
			},
		},
	}

	jaeger := Select(sset, jaegerPods)
	assert.NotNil(t, jaeger)
	assert.Equal(t, "the-only-jaeger-instance-available", jaeger.Name)
}

func TestCannotSelectFromMultipleJaegerPods(t *testing.T) {
	sset := sset(map[string]string{Annotation: "true"})
	jaegerPods := &v1alpha1.JaegerList{
		Items: []v1alpha1.Jaeger{
			v1alpha1.Jaeger{
				ObjectMeta: metav1.ObjectMeta{
					Name: "the-first-jaeger-instance-available",
				},
			},
			v1alpha1.Jaeger{
				ObjectMeta: metav1.ObjectMeta{
					Name: "the-second-jaeger-instance-available",
				},
			},
		},
	}

	jaeger := Select(sset, jaegerPods)
	assert.Nil(t, jaeger)
}

func TestNoAvailableJaegerPods(t *testing.T) {
	sset := sset(map[string]string{Annotation: "true"})
	jaeger := Select(sset, &v1alpha1.JaegerList{})
	assert.Nil(t, jaeger)
}

func TestSelectBasedOnName(t *testing.T) {
	sset := sset(map[string]string{Annotation: "the-second-jaeger-instance-available"})

	jaegerPods := &v1alpha1.JaegerList{
		Items: []v1alpha1.Jaeger{
			v1alpha1.Jaeger{
				ObjectMeta: metav1.ObjectMeta{
					Name: "the-first-jaeger-instance-available",
				},
			},
			v1alpha1.Jaeger{
				ObjectMeta: metav1.ObjectMeta{
					Name: "the-second-jaeger-instance-available",
				},
			},
		},
	}

	jaeger := Select(sset, jaegerPods)
	assert.NotNil(t, jaeger)
	assert.Equal(t, "the-second-jaeger-instance-available", jaeger.Name)
}

func sset(annotations map[string]string) *appsv1.StatefulSet {
	return &appsv1.StatefulSet{
		ObjectMeta: metav1.ObjectMeta{
			Annotations: annotations,
		},
		Spec: appsv1.StatefulSetSpec{
			Template: v1.PodTemplateSpec{
				Spec: v1.PodSpec{
					Containers: []v1.Container{
						v1.Container{},
					},
				},
			},
		},
	}
}
