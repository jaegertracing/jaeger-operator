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
	"k8s.io/apimachinery/pkg/types"

	v1 "github.com/jaegertracing/jaeger-operator/pkg/apis/jaegertracing/v1"
	"github.com/jaegertracing/jaeger-operator/pkg/util"
)

func setDefaults() {
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
	jaeger := v1.NewJaeger(types.NamespacedName{Name: "TestInjectSidecar"})
	dep := dep(map[string]string{Annotation: jaeger.Name}, map[string]string{})
	dep = Sidecar(jaeger, dep)
	assert.Len(t, dep.Spec.Template.Spec.Containers, 2)
	assert.Contains(t, dep.Spec.Template.Spec.Containers[1].Image, "jaeger-agent")
	assert.Len(t, dep.Spec.Template.Spec.Containers[0].Env, 0)
}

func TestInjectSidecarWithLegacyAnnotation(t *testing.T) {
	jaeger := v1.NewJaeger(types.NamespacedName{Name: "TestInjectSidecarWithLegacyAnnotation"})
	dep := dep(map[string]string{AnnotationLegacy: jaeger.Name}, map[string]string{})
	dep = Sidecar(jaeger, dep)
	assert.Len(t, dep.Spec.Template.Spec.Containers, 2)
	assert.Contains(t, dep.Spec.Template.Spec.Containers[1].Image, "jaeger-agent")
	assert.Len(t, dep.Spec.Template.Spec.Containers[0].Env, 0)
}

func TestInjectSidecarWithEnvVars(t *testing.T) {
	// prepare
	jaeger := v1.NewJaeger(types.NamespacedName{Name: "TestInjectSidecarWithEnvVars"})
	dep := dep(map[string]string{Annotation: jaeger.Name}, map[string]string{"app": "testapp"})

	// test
	dep = Sidecar(jaeger, dep)

	// verify
	assert.Len(t, dep.Spec.Template.Spec.Containers, 2)
	assert.Contains(t, dep.Spec.Template.Spec.Containers[1].Image, "jaeger-agent")
	containsEnvVarNamed(t, dep.Spec.Template.Spec.Containers[1].Env, envVarPodName)

	assert.Len(t, dep.Spec.Template.Spec.Containers[0].Env, 2)
	assert.Contains(t, dep.Spec.Template.Spec.Containers[0].Env, corev1.EnvVar{Name: envVarPropagation, Value: "jaeger,b3"})
	assert.Contains(t, dep.Spec.Template.Spec.Containers[0].Env, corev1.EnvVar{Name: envVarServiceName, Value: "testapp.default"})
}

func TestInjectSidecarWithEnvVarsK8sAppName(t *testing.T) {
	// prepare
	jaeger := v1.NewJaeger(types.NamespacedName{Name: "TestInjectSidecarWithEnvVarsK8sAppName"})
	dep := dep(map[string]string{Annotation: jaeger.Name}, map[string]string{
		"app":                    "noapp",
		"app.kubernetes.io/name": "testapp",
	})

	// test
	dep = Sidecar(jaeger, dep)

	// verify
	assert.Len(t, dep.Spec.Template.Spec.Containers, 2)
	assert.Len(t, dep.Spec.Template.Spec.Containers[0].Env, 2)
	assert.Contains(t, dep.Spec.Template.Spec.Containers[0].Env, corev1.EnvVar{Name: envVarServiceName, Value: "testapp.default"})
}

func TestInjectSidecarWithEnvVarsK8sAppInstance(t *testing.T) {
	// prepare
	jaeger := v1.NewJaeger(types.NamespacedName{Name: "TestInjectSidecarWithEnvVarsK8sAppInstance"})
	dep := dep(map[string]string{Annotation: jaeger.Name}, map[string]string{
		"app":                        "noapp",
		"app.kubernetes.io/name":     "noname",
		"app.kubernetes.io/instance": "testapp",
	})

	// test
	dep = Sidecar(jaeger, dep)

	// verify
	assert.Len(t, dep.Spec.Template.Spec.Containers, 2)
	assert.Len(t, dep.Spec.Template.Spec.Containers[0].Env, 2)
	assert.Contains(t, dep.Spec.Template.Spec.Containers[0].Env, corev1.EnvVar{Name: envVarServiceName, Value: "testapp.default"})
}

func TestInjectSidecarWithEnvVarsWithNamespace(t *testing.T) {
	// prepare
	jaeger := v1.NewJaeger(types.NamespacedName{Name: "TestInjectSidecarWithEnvVarsWithNamespace"})
	dep := dep(map[string]string{Annotation: jaeger.Name}, map[string]string{"app": "testapp"})
	dep.Namespace = "mynamespace"

	// test
	dep = Sidecar(jaeger, dep)

	// verify
	assert.Len(t, dep.Spec.Template.Spec.Containers, 2)
	assert.Contains(t, dep.Spec.Template.Spec.Containers[1].Image, "jaeger-agent")

	assert.Len(t, dep.Spec.Template.Spec.Containers[0].Env, 2)
	assert.Contains(t, dep.Spec.Template.Spec.Containers[0].Env, corev1.EnvVar{Name: envVarServiceName, Value: "testapp.mynamespace"})
	assert.Contains(t, dep.Spec.Template.Spec.Containers[0].Env, corev1.EnvVar{Name: envVarPropagation, Value: "jaeger,b3"})
}

func TestInjectSidecarWithEnvVarsOverrideName(t *testing.T) {
	// prepare
	jaeger := v1.NewJaeger(types.NamespacedName{Name: "TestInjectSidecarWithEnvVarsOverrideName"})
	dep := dep(map[string]string{Annotation: jaeger.Name}, map[string]string{"app": "testapp"})
	envVar := corev1.EnvVar{
		Name:  envVarServiceName,
		Value: "otherapp",
	}
	dep.Spec.Template.Spec.Containers[0].Env = append(dep.Spec.Template.Spec.Containers[0].Env, envVar)

	// test
	dep = Sidecar(jaeger, dep)

	// verify
	assert.Len(t, dep.Spec.Template.Spec.Containers, 2)
	assert.Contains(t, dep.Spec.Template.Spec.Containers[1].Image, "jaeger-agent")

	assert.Len(t, dep.Spec.Template.Spec.Containers[0].Env, 2)
	assert.Contains(t, dep.Spec.Template.Spec.Containers[0].Env, envVar)
	assert.Contains(t, dep.Spec.Template.Spec.Containers[0].Env, corev1.EnvVar{Name: envVarPropagation, Value: "jaeger,b3"})
}

func TestInjectSidecarWithEnvVarsOverridePropagation(t *testing.T) {
	// prepare
	jaeger := v1.NewJaeger(types.NamespacedName{Name: "TestInjectSidecarWithEnvVarsOverridePropagation"})
	traceContextEnvVar := corev1.EnvVar{
		Name:  envVarPropagation,
		Value: "tracecontext",
	}
	dep := dep(map[string]string{Annotation: jaeger.Name}, map[string]string{"app": "testapp"})
	dep.Spec.Template.Spec.Containers[0].Env = append(dep.Spec.Template.Spec.Containers[0].Env, traceContextEnvVar)

	// test
	dep = Sidecar(jaeger, dep)

	// verify
	assert.Len(t, dep.Spec.Template.Spec.Containers, 2)
	assert.Contains(t, dep.Spec.Template.Spec.Containers[1].Image, "jaeger-agent")

	assert.Len(t, dep.Spec.Template.Spec.Containers[0].Env, 2)
	assert.Contains(t, dep.Spec.Template.Spec.Containers[0].Env, traceContextEnvVar)
	assert.Contains(t, dep.Spec.Template.Spec.Containers[0].Env, corev1.EnvVar{Name: envVarServiceName, Value: "testapp.default"})
}

func TestSidecarDefaultPorts(t *testing.T) {
	// prepare
	jaeger := v1.NewJaeger(types.NamespacedName{Name: "TestSidecarPorts"})
	dep := dep(map[string]string{Annotation: jaeger.Name}, map[string]string{"app": "testapp"})

	// test
	dep = Sidecar(jaeger, dep)

	// verify
	assert.Len(t, dep.Spec.Template.Spec.Containers, 2)
	assert.Contains(t, dep.Spec.Template.Spec.Containers[1].Image, "jaeger-agent")

	assert.Len(t, dep.Spec.Template.Spec.Containers[1].Ports, 4)
	assert.Contains(t, dep.Spec.Template.Spec.Containers[1].Ports, corev1.ContainerPort{ContainerPort: 5775, Name: "zk-compact-trft", Protocol: corev1.ProtocolUDP})
	assert.Contains(t, dep.Spec.Template.Spec.Containers[1].Ports, corev1.ContainerPort{ContainerPort: 5778, Name: "config-rest"})
	assert.Contains(t, dep.Spec.Template.Spec.Containers[1].Ports, corev1.ContainerPort{ContainerPort: 6831, Name: "jg-compact-trft", Protocol: corev1.ProtocolUDP})
	assert.Contains(t, dep.Spec.Template.Spec.Containers[1].Ports, corev1.ContainerPort{ContainerPort: 6832, Name: "jg-binary-trft", Protocol: corev1.ProtocolUDP})
}

func TestSkipInjectSidecar(t *testing.T) {
	// prepare
	jaeger := v1.NewJaeger(types.NamespacedName{Name: "TestSkipInjectSidecar"})
	dep := dep(map[string]string{Annotation: "non-existing-operator"}, map[string]string{})

	// test
	dep = Sidecar(jaeger, dep)

	// verify
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
	jaeger := v1.NewJaeger(types.NamespacedName{Name: "TestHasSidecarAlready"})
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
	jaeger := v1.NewJaeger(types.NamespacedName{Name: "TestQueryOrderOfArguments"})
	jaeger.Spec.Agent.Options = v1.NewOptions(map[string]interface{}{
		"b-option": "b-value",
		"a-option": "a-value",
		"c-option": "c-value",
	})

	dep := dep(map[string]string{Annotation: jaeger.Name}, map[string]string{})
	dep = Sidecar(jaeger, dep)

	assert.Len(t, dep.Spec.Template.Spec.Containers, 2)
	assert.Len(t, dep.Spec.Template.Spec.Containers[1].Args, 6)
	containsOptionWithPrefix(t, dep.Spec.Template.Spec.Containers[1].Args, "--a-option")
	containsOptionWithPrefix(t, dep.Spec.Template.Spec.Containers[1].Args, "--b-option")
	containsOptionWithPrefix(t, dep.Spec.Template.Spec.Containers[1].Args, "--c-option")
	containsOptionWithPrefix(t, dep.Spec.Template.Spec.Containers[1].Args, "--jaeger.tags")
	containsOptionWithPrefix(t, dep.Spec.Template.Spec.Containers[1].Args, "--reporter.grpc.host-port")
	containsOptionWithPrefix(t, dep.Spec.Template.Spec.Containers[1].Args, "--reporter.type")
	agentTags := agentTags(dep.Spec.Template.Spec.Containers[1].Args)
	assert.Contains(t, agentTags, "container.name=only_container")
}

func TestSidecarExplicitTags(t *testing.T) {
	// prepare
	jaeger := v1.NewJaeger(types.NamespacedName{Name: "TestSidecarExplicitTags"})
	jaeger.Spec.Agent.Options = v1.NewOptions(map[string]interface{}{"jaeger.tags": "key=val"})
	dep := dep(map[string]string{Annotation: jaeger.Name}, map[string]string{})

	// test
	dep = Sidecar(jaeger, dep)

	// verify
	assert.Len(t, dep.Spec.Template.Spec.Containers, 2)
	agentTags := agentTags(dep.Spec.Template.Spec.Containers[1].Args)
	assert.Equal(t, []string{"key=val"}, agentTags)
}

func TestSidecarOverrideReporter(t *testing.T) {
	jaeger := v1.NewJaeger(types.NamespacedName{Name: "TestQueryOrderOfArguments"})
	jaeger.Spec.Agent.Options = v1.NewOptions(map[string]interface{}{
		"reporter.type":             "thrift",
		"reporter.thrift.host-port": "collector:14267",
	})

	dep := dep(map[string]string{Annotation: jaeger.Name}, map[string]string{})
	dep = Sidecar(jaeger, dep)

	assert.Len(t, dep.Spec.Template.Spec.Containers, 2)
	assert.Len(t, dep.Spec.Template.Spec.Containers[1].Args, 3)
	assert.Contains(t, dep.Spec.Template.Spec.Containers[1].Args, "--reporter.thrift.host-port=collector:14267")
	assert.Contains(t, dep.Spec.Template.Spec.Containers[1].Args, "--reporter.type=thrift")
}

func TestSidecarAgentResources(t *testing.T) {
	jaeger := v1.NewJaeger(types.NamespacedName{Name: "TestSidecarAgentResources"})
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

func TestCleanSidecars(t *testing.T) {
	nsn := types.NamespacedName{
		Name:      "TestCleanSideCars",
		Namespace: "Test",
	}
	jaeger := v1.NewJaeger(nsn)
	dep1 := Sidecar(jaeger, dep(map[string]string{Annotation: jaeger.Name}, map[string]string{}))
	CleanSidecar(dep1)
	assert.Equal(t, len(dep1.Spec.Template.Spec.Containers), 1)

}

func TestSidecarWithLabel(t *testing.T) {
	nsn := types.NamespacedName{
		Name:      "TestSidecarWithLabel",
		Namespace: "Test",
	}
	jaeger := v1.NewJaeger(nsn)
	dep1 := dep(map[string]string{Annotation: jaeger.Name}, map[string]string{})
	dep1 = Sidecar(jaeger, dep1)
	assert.Equal(t, dep1.Labels[Label], "TestSidecarWithLabel")
	dep2 := dep(map[string]string{Annotation: jaeger.Name}, map[string]string{})
	dep2.Labels = map[string]string{"anotherLabel": "anotherValue"}
	dep2 = Sidecar(jaeger, dep2)
	assert.Equal(t, len(dep2.Labels), 2)
	assert.Equal(t, dep2.Labels["anotherLabel"], "anotherValue")
	assert.Equal(t, dep2.Labels[Label], jaeger.Name)
}

func TestSidecarWithoutPrometheusAnnotations(t *testing.T) {
	// prepare
	jaeger := v1.NewJaeger(types.NamespacedName{Name: "TestSidecarWithoutPrometheusAnnotations"})
	dep := Sidecar(jaeger, dep(map[string]string{Annotation: jaeger.Name}, map[string]string{}))

	// test
	dep = Sidecar(jaeger, dep)

	// verify
	assert.Contains(t, dep.Annotations, "prometheus.io/scrape")
	assert.Contains(t, dep.Annotations, "prometheus.io/port")
}

func TestSidecarWithPrometheusAnnotations(t *testing.T) {
	// prepare
	jaeger := v1.NewJaeger(types.NamespacedName{Name: "TestSidecarWithPrometheusAnnotations"})
	dep := dep(map[string]string{
		Annotation:             jaeger.Name,
		"prometheus.io/scrape": "false",
		"prometheus.io/port":   "9090",
	}, map[string]string{})

	// test
	dep = Sidecar(jaeger, dep)

	// verify
	assert.Equal(t, dep.Annotations["prometheus.io/scrape"], "false")
	assert.Equal(t, dep.Annotations["prometheus.io/port"], "9090")
}

func TestSidecarAgentTagsWithMultipleContainers(t *testing.T) {
	jaeger := v1.NewJaeger(types.NamespacedName{Name: "TestSidecarAgentTagsWithMultipleContainers"})
	dep := Sidecar(jaeger, depWithTwoContainers(map[string]string{Annotation: jaeger.Name}, map[string]string{}))

	assert.Len(t, dep.Spec.Template.Spec.Containers, 3, "Expected 3 containers")
	assert.Equal(t, "jaeger-agent", dep.Spec.Template.Spec.Containers[2].Name)
	containsOptionWithPrefix(t, dep.Spec.Template.Spec.Containers[2].Args, "--jaeger.tags")
	agentTags := agentTags(dep.Spec.Template.Spec.Containers[2].Args)
	assert.Equal(t, "", util.FindItem("container.name=", agentTags))
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
						corev1.Container{
							Name: "only_container",
						},
					},
				},
			},
		},
	}
}

func depWithTwoContainers(annotations map[string]string, labels map[string]string) *appsv1.Deployment {
	dep := dep(annotations, labels)
	dep.Spec.Template.Spec.Containers[0].Name = "container_0"
	dep.Spec.Template.Spec.Containers = append(dep.Spec.Template.Spec.Containers, corev1.Container{
		Name: "container_1",
	})
	return dep
}

func containsEnvVarNamed(t *testing.T, envVars []corev1.EnvVar, key string) bool {
	for _, envVar := range envVars {
		if envVar.Name == key {
			return true
		}
	}
	assert.Fail(t, "element with key '%s' not found", key)
	return false
}

func containsOptionWithPrefix(t *testing.T, args []string, prefix string) bool {
	for _, arg := range args {
		if strings.HasPrefix(arg, prefix) {
			return true
		}
	}
	assert.Fail(t, "list of arguments didn't have an option starting with '%s'", prefix)
	return false
}

func agentTags(args []string) []string {
	tagsArg := util.FindItem("--jaeger.tags=", args)
	if tagsArg == "" {
		return []string{}
	}
	tagsParam := strings.SplitN(tagsArg, "=", 2)[1]
	return strings.Split(tagsParam, ",")
}
