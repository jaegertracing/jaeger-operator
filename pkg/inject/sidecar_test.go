package inject

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
	dep := dep(map[string]string{Annotation: jaeger.Name}, map[string]string{})
	Sidecar(dep, jaeger)
	assert.Len(t, dep.Spec.Template.Spec.Containers, 2)
	assert.Contains(t, dep.Spec.Template.Spec.Containers[1].Image, "jaeger-agent")
	assert.Len(t, dep.Spec.Template.Spec.Containers[0].Env, 0)
}

func TestInjectSidecarWithEnvVars(t *testing.T) {
	jaeger := v1alpha1.NewJaeger("TestInjectSidecarWithEnvVars")
	dep := dep(map[string]string{Annotation: jaeger.Name}, map[string]string{"app": "testapp"})
	Sidecar(dep, jaeger)
	assert.Len(t, dep.Spec.Template.Spec.Containers, 2)
	assert.Contains(t, dep.Spec.Template.Spec.Containers[1].Image, "jaeger-agent")
	assert.Len(t, dep.Spec.Template.Spec.Containers[0].Env, 2)
	assert.Equal(t, envVarServiceName, dep.Spec.Template.Spec.Containers[0].Env[0].Name)
	assert.Equal(t, "testapp.default", dep.Spec.Template.Spec.Containers[0].Env[0].Value)
	assert.Equal(t, envVarPropagation, dep.Spec.Template.Spec.Containers[0].Env[1].Name)
	assert.Equal(t, "jaeger,b3", dep.Spec.Template.Spec.Containers[0].Env[1].Value)
}

func TestInjectSidecarWithEnvVarsWithNamespace(t *testing.T) {
	jaeger := v1alpha1.NewJaeger("TestInjectSidecarWithEnvVarsWithNamespace")
	dep := dep(map[string]string{Annotation: jaeger.Name}, map[string]string{"app": "testapp"})
	dep.Namespace = "mynamespace"
	Sidecar(dep, jaeger)
	assert.Len(t, dep.Spec.Template.Spec.Containers, 2)
	assert.Contains(t, dep.Spec.Template.Spec.Containers[1].Image, "jaeger-agent")
	assert.Len(t, dep.Spec.Template.Spec.Containers[0].Env, 2)
	assert.Equal(t, envVarServiceName, dep.Spec.Template.Spec.Containers[0].Env[0].Name)
	assert.Equal(t, "testapp.mynamespace", dep.Spec.Template.Spec.Containers[0].Env[0].Value)
	assert.Equal(t, envVarPropagation, dep.Spec.Template.Spec.Containers[0].Env[1].Name)
	assert.Equal(t, "jaeger,b3", dep.Spec.Template.Spec.Containers[0].Env[1].Value)
}

func TestInjectSidecarWithEnvVarsOverrideName(t *testing.T) {
	jaeger := v1alpha1.NewJaeger("TestInjectSidecarWithEnvVarsOverrideName")
	dep := dep(map[string]string{Annotation: jaeger.Name}, map[string]string{"app": "testapp"})
	dep.Spec.Template.Spec.Containers[0].Env = append(dep.Spec.Template.Spec.Containers[0].Env, v1.EnvVar{
		Name:  envVarServiceName,
		Value: "otherapp",
	})
	Sidecar(dep, jaeger)
	assert.Len(t, dep.Spec.Template.Spec.Containers, 2)
	assert.Contains(t, dep.Spec.Template.Spec.Containers[1].Image, "jaeger-agent")
	assert.Len(t, dep.Spec.Template.Spec.Containers[0].Env, 2)
	assert.Equal(t, envVarServiceName, dep.Spec.Template.Spec.Containers[0].Env[0].Name)
	// Explicitly provided env var is used instead of injected "app.namespace" value
	assert.Equal(t, "otherapp", dep.Spec.Template.Spec.Containers[0].Env[0].Value)
	assert.Equal(t, envVarPropagation, dep.Spec.Template.Spec.Containers[0].Env[1].Name)
	assert.Equal(t, "jaeger,b3", dep.Spec.Template.Spec.Containers[0].Env[1].Value)
}

func TestInjectSidecarWithEnvVarsOverridePropagation(t *testing.T) {
	jaeger := v1alpha1.NewJaeger("TestInjectSidecarWithEnvVarsOverridePropagation")
	dep := dep(map[string]string{Annotation: jaeger.Name}, map[string]string{"app": "testapp"})
	dep.Spec.Template.Spec.Containers[0].Env = append(dep.Spec.Template.Spec.Containers[0].Env, v1.EnvVar{
		Name:  envVarPropagation,
		Value: "tracecontext",
	})
	Sidecar(dep, jaeger)
	assert.Len(t, dep.Spec.Template.Spec.Containers, 2)
	assert.Contains(t, dep.Spec.Template.Spec.Containers[1].Image, "jaeger-agent")
	assert.Len(t, dep.Spec.Template.Spec.Containers[0].Env, 2)
	assert.Equal(t, envVarPropagation, dep.Spec.Template.Spec.Containers[0].Env[0].Name)
	// Explicitly provided propagation env var used instead of injected "jaeger,b3" value
	assert.Equal(t, "tracecontext", dep.Spec.Template.Spec.Containers[0].Env[0].Value)
	assert.Equal(t, envVarServiceName, dep.Spec.Template.Spec.Containers[0].Env[1].Name)
	assert.Equal(t, "testapp.default", dep.Spec.Template.Spec.Containers[0].Env[1].Value)
}

func TestSkipInjectSidecar(t *testing.T) {
	jaeger := v1alpha1.NewJaeger("TestSkipInjectSidecar")
	dep := dep(map[string]string{Annotation: "non-existing-operator"}, map[string]string{})
	Sidecar(dep, jaeger)
	assert.Len(t, dep.Spec.Template.Spec.Containers, 1)
	assert.NotContains(t, dep.Spec.Template.Spec.Containers[0].Image, "jaeger-agent")
}

func TestSidecarNotNeeded(t *testing.T) {
	dep := &appsv1.Deployment{
		Spec: appsv1.DeploymentSpec{
			Template: v1.PodTemplateSpec{
				Spec: v1.PodSpec{
					Containers: []v1.Container{
						v1.Container{},
					},
				},
			},
		},
	}

	assert.False(t, Needed(dep.Name, dep.Annotations, dep.Spec.Template.Spec.Containers))
}

func TestSidecarNeeded(t *testing.T) {
	dep := dep(map[string]string{Annotation: "some-jaeger-instance"}, map[string]string{})
	assert.True(t, Needed(dep.Name, dep.Annotations, dep.Spec.Template.Spec.Containers))
}

func TestHasSidecarAlready(t *testing.T) {
	dep := dep(map[string]string{Annotation: "TestHasSidecarAlready"}, map[string]string{})
	assert.True(t, Needed(dep.Name, dep.Annotations, dep.Spec.Template.Spec.Containers))
	jaeger := v1alpha1.NewJaeger("TestHasSidecarAlready")
	Sidecar(dep, jaeger)
	assert.False(t, Needed(dep.Name, dep.Annotations, dep.Spec.Template.Spec.Containers))
}

func TestSelectSingleJaegerPod(t *testing.T) {
	dep := dep(map[string]string{Annotation: "true"}, map[string]string{})
	jaegerPods := &v1alpha1.JaegerList{
		Items: []v1alpha1.Jaeger{
			v1alpha1.Jaeger{
				ObjectMeta: metav1.ObjectMeta{
					Name: "the-only-jaeger-instance-available",
				},
			},
		},
	}

	jaeger := Select(dep.Annotations, jaegerPods)
	assert.NotNil(t, jaeger)
	assert.Equal(t, "the-only-jaeger-instance-available", jaeger.Name)
}

func TestCannotSelectFromMultipleJaegerPods(t *testing.T) {
	dep := dep(map[string]string{Annotation: "true"}, map[string]string{})
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

	jaeger := Select(dep.Annotations, jaegerPods)
	assert.Nil(t, jaeger)
}

func TestNoAvailableJaegerPods(t *testing.T) {
	dep := dep(map[string]string{Annotation: "true"}, map[string]string{})
	jaeger := Select(dep.Annotations, &v1alpha1.JaegerList{})
	assert.Nil(t, jaeger)
}

func TestSelectBasedOnName(t *testing.T) {
	dep := dep(map[string]string{Annotation: "the-second-jaeger-instance-available"}, map[string]string{})

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

	jaeger := Select(dep.Annotations, jaegerPods)
	assert.NotNil(t, jaeger)
	assert.Equal(t, "the-second-jaeger-instance-available", jaeger.Name)
}

func dep(annotations map[string]string, labels map[string]string) *appsv1.Deployment {
	return &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Annotations: annotations,
		},
		Spec: appsv1.DeploymentSpec{
			Template: v1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: labels,
				},
				Spec: v1.PodSpec{
					Containers: []v1.Container{
						v1.Container{},
					},
				},
			},
		},
	}
}
