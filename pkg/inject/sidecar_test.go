package inject

import (
	"strings"
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

func TestInjectSidecarWithEnvVarsK8sAppName(t *testing.T) {
	jaeger := v1.NewJaeger("TestInjectSidecarWithEnvVarsK8sAppName")
	dep := dep(map[string]string{Annotation: jaeger.Name}, map[string]string{
		"app":                    "noapp",
		"app.kubernetes.io/name": "testapp",
	})
	dep = Sidecar(jaeger, dep)
	assert.Len(t, dep.Spec.Template.Spec.Containers, 2)
	assert.Len(t, dep.Spec.Template.Spec.Containers[0].Env, 2)
	assert.Equal(t, envVarServiceName, dep.Spec.Template.Spec.Containers[0].Env[0].Name)
	assert.Equal(t, "testapp.default", dep.Spec.Template.Spec.Containers[0].Env[0].Value)
}

func TestInjectSidecarWithEnvVarsK8sAppInstance(t *testing.T) {
	jaeger := v1.NewJaeger("TestInjectSidecarWithEnvVarsK8sAppInstance")
	dep := dep(map[string]string{Annotation: jaeger.Name}, map[string]string{
		"app":                        "noapp",
		"app.kubernetes.io/name":     "noname",
		"app.kubernetes.io/instance": "testapp",
	})
	dep = Sidecar(jaeger, dep)
	assert.Len(t, dep.Spec.Template.Spec.Containers, 2)
	assert.Len(t, dep.Spec.Template.Spec.Containers[0].Env, 2)
	assert.Equal(t, envVarServiceName, dep.Spec.Template.Spec.Containers[0].Env[0].Name)
	assert.Equal(t, "testapp.default", dep.Spec.Template.Spec.Containers[0].Env[0].Value)
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

func TestSidecarOrderOfArguments(t *testing.T) {
	jaeger := v1.NewJaeger("TestQueryOrderOfArguments")
	jaeger.Spec.Agent.Options = v1.NewOptions(map[string]interface{}{
		"b-option": "b-value",
		"a-option": "a-value",
		"c-option": "c-value",
	})

	dep := dep(map[string]string{Annotation: jaeger.Name}, map[string]string{})
	dep = Sidecar(jaeger, dep)

	assert.Len(t, dep.Spec.Template.Spec.Containers, 2)
	assert.Len(t, dep.Spec.Template.Spec.Containers[1].Args, 5)
	assert.True(t, strings.HasPrefix(dep.Spec.Template.Spec.Containers[1].Args[0], "--a-option"))
	assert.True(t, strings.HasPrefix(dep.Spec.Template.Spec.Containers[1].Args[1], "--b-option"))
	assert.True(t, strings.HasPrefix(dep.Spec.Template.Spec.Containers[1].Args[2], "--c-option"))
	assert.True(t, strings.HasPrefix(dep.Spec.Template.Spec.Containers[1].Args[3], "--reporter.grpc.host-port"))
	assert.True(t, strings.HasPrefix(dep.Spec.Template.Spec.Containers[1].Args[4], "--reporter.type"))
}

func TestSidecarOverrideReporter(t *testing.T) {
	jaeger := v1.NewJaeger("TestQueryOrderOfArguments")
	jaeger.Spec.Agent.Options = v1.NewOptions(map[string]interface{}{
		"reporter.type":             "thrift",
		"reporter.thrift.host-port": "collector:14267",
	})

	dep := dep(map[string]string{Annotation: jaeger.Name}, map[string]string{})
	dep = Sidecar(jaeger, dep)

	assert.Len(t, dep.Spec.Template.Spec.Containers, 2)
	assert.Len(t, dep.Spec.Template.Spec.Containers[1].Args, 2)
	assert.Equal(t, "--reporter.thrift.host-port=collector:14267", dep.Spec.Template.Spec.Containers[1].Args[0])
	assert.Equal(t, "--reporter.type=thrift", dep.Spec.Template.Spec.Containers[1].Args[1])
}

func TestSidecarAgentResources(t *testing.T) {
	jaeger := v1.NewJaeger("TestSidecarAgentResources")
	jaeger.Spec.Resources = corev1.ResourceRequirements{
		Limits: corev1.ResourceList{
			corev1.ResourceLimitsCPU:              *resource.NewQuantity(1024, resource.BinarySI),
			corev1.ResourceLimitsEphemeralStorage: *resource.NewQuantity(512, resource.DecimalSI),
		},
		Requests: corev1.ResourceList{
			corev1.ResourceRequestsCPU:              *resource.NewQuantity(1024, resource.BinarySI),
			corev1.ResourceRequestsEphemeralStorage: *resource.NewQuantity(512, resource.DecimalSI),
		},
	}
	jaeger.Spec.Agent.Resources = corev1.ResourceRequirements{
		Limits: corev1.ResourceList{
			corev1.ResourceLimitsCPU:    *resource.NewQuantity(2048, resource.BinarySI),
			corev1.ResourceLimitsMemory: *resource.NewQuantity(123, resource.DecimalSI),
		},
		Requests: corev1.ResourceList{
			corev1.ResourceRequestsCPU:    *resource.NewQuantity(2048, resource.BinarySI),
			corev1.ResourceRequestsMemory: *resource.NewQuantity(123, resource.DecimalSI),
		},
	}

	dep := dep(map[string]string{Annotation: jaeger.Name}, map[string]string{})
	dep = Sidecar(jaeger, dep)

	assert.Len(t, dep.Spec.Template.Spec.Containers, 2, "Expected 2 containers")
	assert.Equal(t, "jaeger-agent", dep.Spec.Template.Spec.Containers[1].Name)
	assert.Equal(t, *resource.NewQuantity(2048, resource.BinarySI), dep.Spec.Template.Spec.Containers[1].Resources.Limits[corev1.ResourceLimitsCPU])
	assert.Equal(t, *resource.NewQuantity(2048, resource.BinarySI), dep.Spec.Template.Spec.Containers[1].Resources.Requests[corev1.ResourceRequestsCPU])
	assert.Equal(t, *resource.NewQuantity(123, resource.DecimalSI), dep.Spec.Template.Spec.Containers[1].Resources.Limits[corev1.ResourceLimitsMemory])
	assert.Equal(t, *resource.NewQuantity(123, resource.DecimalSI), dep.Spec.Template.Spec.Containers[1].Resources.Requests[corev1.ResourceRequestsMemory])
	assert.Equal(t, *resource.NewQuantity(512, resource.DecimalSI), dep.Spec.Template.Spec.Containers[1].Resources.Limits[corev1.ResourceLimitsEphemeralStorage])
	assert.Equal(t, *resource.NewQuantity(512, resource.DecimalSI), dep.Spec.Template.Spec.Containers[1].Resources.Requests[corev1.ResourceRequestsEphemeralStorage])
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
