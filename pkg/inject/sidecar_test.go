package inject

import (
	"testing"

	"github.com/spf13/viper"
	"github.com/stretchr/testify/assert"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/jaegertracing/jaeger-operator/pkg/apis/jaegertracing/v1"
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
	jaeger := v1.NewJaeger("TestInjectSidecar")
	dep := dep(map[string]string{Annotation: jaeger.Name}, map[string]string{})
	dep = Sidecar(jaeger, dep)
	assert.Len(t, dep.Spec.Template.Spec.Containers, 2)
	assert.Contains(t, dep.Spec.Template.Spec.Containers[1].Image, "jaeger-agent")
	assert.Len(t, dep.Spec.Template.Spec.Containers[0].Env, 0)
}

func TestInjectSidecarWithLegacyAnnotation(t *testing.T) {
	jaeger := v1.NewJaeger("TestInjectSidecarWithLegacyAnnotation")
	dep := dep(map[string]string{AnnotationLegacy: jaeger.Name}, map[string]string{})
	dep = Sidecar(jaeger, dep)
	assert.Len(t, dep.Spec.Template.Spec.Containers, 2)
	assert.Contains(t, dep.Spec.Template.Spec.Containers[1].Image, "jaeger-agent")
	assert.Len(t, dep.Spec.Template.Spec.Containers[0].Env, 0)
}

func TestInjectSidecarWithEnvVars(t *testing.T) {
	jaeger := v1.NewJaeger("TestInjectSidecarWithEnvVars")
	dep := dep(map[string]string{Annotation: jaeger.Name}, map[string]string{"app": "testapp"})
	dep = Sidecar(jaeger, dep)
	assert.Len(t, dep.Spec.Template.Spec.Containers, 2)
	assert.Contains(t, dep.Spec.Template.Spec.Containers[1].Image, "jaeger-agent")
	assert.Len(t, dep.Spec.Template.Spec.Containers[0].Env, 2)
	assert.Equal(t, envVarServiceName, dep.Spec.Template.Spec.Containers[0].Env[0].Name)
	assert.Equal(t, "testapp.default", dep.Spec.Template.Spec.Containers[0].Env[0].Value)
	assert.Equal(t, envVarPropagation, dep.Spec.Template.Spec.Containers[0].Env[1].Name)
	assert.Equal(t, "jaeger,b3", dep.Spec.Template.Spec.Containers[0].Env[1].Value)
}

func TestInjectSidecarWithEnvVarsWithNamespace(t *testing.T) {
	jaeger := v1.NewJaeger("TestInjectSidecarWithEnvVarsWithNamespace")
	dep := dep(map[string]string{Annotation: jaeger.Name}, map[string]string{"app": "testapp"})
	dep.Namespace = "mynamespace"
	dep = Sidecar(jaeger, dep)
	assert.Len(t, dep.Spec.Template.Spec.Containers, 2)
	assert.Contains(t, dep.Spec.Template.Spec.Containers[1].Image, "jaeger-agent")
	assert.Len(t, dep.Spec.Template.Spec.Containers[0].Env, 2)
	assert.Equal(t, envVarServiceName, dep.Spec.Template.Spec.Containers[0].Env[0].Name)
	assert.Equal(t, "testapp.mynamespace", dep.Spec.Template.Spec.Containers[0].Env[0].Value)
	assert.Equal(t, envVarPropagation, dep.Spec.Template.Spec.Containers[0].Env[1].Name)
	assert.Equal(t, "jaeger,b3", dep.Spec.Template.Spec.Containers[0].Env[1].Value)
}

func TestInjectSidecarWithEnvVarsOverrideName(t *testing.T) {
	jaeger := v1.NewJaeger("TestInjectSidecarWithEnvVarsOverrideName")
	dep := dep(map[string]string{Annotation: jaeger.Name}, map[string]string{"app": "testapp"})
	dep.Spec.Template.Spec.Containers[0].Env = append(dep.Spec.Template.Spec.Containers[0].Env, corev1.EnvVar{
		Name:  envVarServiceName,
		Value: "otherapp",
	})
	dep = Sidecar(jaeger, dep)
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
	jaeger := v1.NewJaeger("TestInjectSidecarWithEnvVarsOverridePropagation")
	dep := dep(map[string]string{Annotation: jaeger.Name}, map[string]string{"app": "testapp"})
	dep.Spec.Template.Spec.Containers[0].Env = append(dep.Spec.Template.Spec.Containers[0].Env, corev1.EnvVar{
		Name:  envVarPropagation,
		Value: "tracecontext",
	})
	dep = Sidecar(jaeger, dep)
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
	jaeger := v1.NewJaeger("TestSkipInjectSidecar")
	dep := dep(map[string]string{Annotation: "non-existing-operator"}, map[string]string{})
	dep = Sidecar(jaeger, dep)
	assert.Len(t, dep.Spec.Template.Spec.Containers, 1)
	assert.NotContains(t, dep.Spec.Template.Spec.Containers[0].Image, "jaeger-agent")
}

func TestSidecarNotNeeded(t *testing.T) {
	dep := &appsv1.Deployment{
		Spec: appsv1.DeploymentSpec{
			Template: corev1.PodTemplateSpec{
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{
						corev1.Container{},
					},
				},
			},
		},
	}

	assert.False(t, Needed(dep))
}

func TestSidecarNeeded(t *testing.T) {
	dep := dep(map[string]string{Annotation: "some-jaeger-instance"}, map[string]string{})
	assert.True(t, Needed(dep))
}

func TestHasSidecarAlready(t *testing.T) {
	dep := dep(map[string]string{Annotation: "TestHasSidecarAlready"}, map[string]string{})
	assert.True(t, Needed(dep))
	jaeger := v1.NewJaeger("TestHasSidecarAlready")
	dep = Sidecar(jaeger, dep)
	assert.False(t, Needed(dep))
}

func TestSelectSingleJaegerPod(t *testing.T) {
	dep := dep(map[string]string{Annotation: "true"}, map[string]string{})
	jaegerPods := &v1.JaegerList{
		Items: []v1.Jaeger{
			v1.Jaeger{
				ObjectMeta: metav1.ObjectMeta{
					Name: "the-only-jaeger-instance-available",
				},
			},
		},
	}

	jaeger := Select(dep, jaegerPods)
	assert.NotNil(t, jaeger)
	assert.Equal(t, "the-only-jaeger-instance-available", jaeger.Name)
}

func TestCannotSelectFromMultipleJaegerPods(t *testing.T) {
	dep := dep(map[string]string{Annotation: "true"}, map[string]string{})
	jaegerPods := &v1.JaegerList{
		Items: []v1.Jaeger{
			v1.Jaeger{
				ObjectMeta: metav1.ObjectMeta{
					Name: "the-first-jaeger-instance-available",
				},
			},
			v1.Jaeger{
				ObjectMeta: metav1.ObjectMeta{
					Name: "the-second-jaeger-instance-available",
				},
			},
		},
	}

	jaeger := Select(dep, jaegerPods)
	assert.Nil(t, jaeger)
}

func TestNoAvailableJaegerPods(t *testing.T) {
	dep := dep(map[string]string{Annotation: "true"}, map[string]string{})
	jaeger := Select(dep, &v1.JaegerList{})
	assert.Nil(t, jaeger)
}

func TestSelectBasedOnName(t *testing.T) {
	dep := dep(map[string]string{Annotation: "the-second-jaeger-instance-available"}, map[string]string{})

	jaegerPods := &v1.JaegerList{
		Items: []v1.Jaeger{
			v1.Jaeger{
				ObjectMeta: metav1.ObjectMeta{
					Name: "the-first-jaeger-instance-available",
				},
			},
			v1.Jaeger{
				ObjectMeta: metav1.ObjectMeta{
					Name: "the-second-jaeger-instance-available",
				},
			},
		},
	}

	jaeger := Select(dep, jaegerPods)
	assert.NotNil(t, jaeger)
	assert.Equal(t, "the-second-jaeger-instance-available", jaeger.Name)
}

func TestAgentResouceDefs(t *testing.T) {
	jaeger := v1alpha1.NewJaeger("TestAgentResouceDefs")
	dep := dep(map[string]string{Annotation: jaeger.Name}, map[string]string{})

	// Inject sidecar agent
	Sidecar(dep, jaeger)

	// Assert that the agent is injected.
	assert.Len(t, dep.Spec.Template.Spec.Containers, 2)
	assert.Contains(t, dep.Spec.Template.Spec.Containers[1].Image, "jaeger-agent")

	// Check resource values for the injected sidecar.
	assert.Equal(t, dep.Spec.Template.Spec.Containers[1].Resources.Limits[v1.ResourceLimitsCPU], *resource.NewMilliQuantity(int64(500), resource.BinarySI))
	assert.Equal(t, dep.Spec.Template.Spec.Containers[1].Resources.Limits[v1.ResourceLimitsMemory], *resource.NewScaledQuantity(int64(128), resource.Mega))
}

func TestAgentResouceDefsOverride(t *testing.T) {
	jaeger := v1alpha1.NewJaeger("TestAgentResouceDefsOverride")
	dep := dep(map[string]string{Annotation: jaeger.Name, "jaeger-agent-max-cpu": "1024", "jaeger-agent-max-memory": "100"}, map[string]string{})

	// Inject sidecar agent
	Sidecar(dep, jaeger)

	// Assert that the agent is injected.
	assert.Len(t, dep.Spec.Template.Spec.Containers, 2)
	assert.Contains(t, dep.Spec.Template.Spec.Containers[1].Image, "jaeger-agent")

	// Check resource values for the injected sidecar.
	assert.Equal(t, dep.Spec.Template.Spec.Containers[1].Resources.Limits[v1.ResourceLimitsCPU], *resource.NewMilliQuantity(int64(1024), resource.BinarySI))
	assert.Equal(t, dep.Spec.Template.Spec.Containers[1].Resources.Limits[v1.ResourceLimitsMemory], *resource.NewScaledQuantity(int64(100), resource.Mega))
}

func dep(annotations map[string]string, labels map[string]string) *appsv1.Deployment {
	return &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Annotations: annotations,
		},
		Spec: appsv1.DeploymentSpec{
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: labels,
				},
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{
						corev1.Container{},
					},
				},
			},
		},
	}
}
