package inject

import (
	"fmt"
	"strings"
	"testing"

	"github.com/spf13/viper"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"

	v1 "github.com/jaegertracing/jaeger-operator/pkg/apis/jaegertracing/v1"
	"github.com/jaegertracing/jaeger-operator/pkg/config/ca"
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
	jaeger := v1.NewJaeger(types.NamespacedName{Name: "my-instance"})
	dep := dep(map[string]string{}, map[string]string{})
	dep = Sidecar(jaeger, dep)
	assert.Equal(t, dep.Labels[Label], jaeger.Name)
	assert.Len(t, dep.Spec.Template.Spec.Containers, 2)
	assert.Contains(t, dep.Spec.Template.Spec.Containers[1].Image, "jaeger-agent")
	assert.Len(t, dep.Spec.Template.Spec.Containers[0].Env, 0)
	assert.Len(t, dep.Spec.Template.Spec.Containers[1].VolumeMounts, 0)
	assert.Len(t, dep.Spec.Template.Spec.Volumes, 0)
}

func TestInjectSidecarOpenShift(t *testing.T) {
	viper.Set("platform", v1.FlagPlatformOpenShift)
	defer reset()

	jaeger := v1.NewJaeger(types.NamespacedName{Name: "my-instance"})
	dep := dep(map[string]string{}, map[string]string{})
	dep = Sidecar(jaeger, dep)
	assert.Equal(t, dep.Labels[Label], jaeger.Name)
	assert.Len(t, dep.Spec.Template.Spec.Containers, 2)
	assert.Contains(t, dep.Spec.Template.Spec.Containers[1].Image, "jaeger-agent")
	assert.Len(t, dep.Spec.Template.Spec.Containers[0].Env, 0)
	assert.Len(t, dep.Spec.Template.Spec.Containers[1].VolumeMounts, 2)
	assert.Len(t, dep.Spec.Template.Spec.Volumes, 2)
}

func TestInjectSidecarWithEnvVars(t *testing.T) {
	// prepare
	jaeger := v1.NewJaeger(types.NamespacedName{Name: "my-instance"})
	dep := dep(map[string]string{Annotation: jaeger.Name}, map[string]string{"app": "testapp"})

	// test
	dep = Sidecar(jaeger, dep)

	// verify
	assert.Len(t, dep.Spec.Template.Spec.Containers, 2)
	assert.Contains(t, dep.Spec.Template.Spec.Containers[1].Image, "jaeger-agent")
	containsEnvVarNamed(t, dep.Spec.Template.Spec.Containers[1].Env, envVarPodName)
	containsEnvVarNamed(t, dep.Spec.Template.Spec.Containers[1].Env, envVarHostIP)

	assert.Len(t, dep.Spec.Template.Spec.Containers[0].Env, 2)
	assert.Contains(t, dep.Spec.Template.Spec.Containers[0].Env, corev1.EnvVar{Name: envVarPropagation, Value: "jaeger,b3"})
	assert.Contains(t, dep.Spec.Template.Spec.Containers[0].Env, corev1.EnvVar{Name: envVarServiceName, Value: "testapp.default"})
}

func TestInjectSidecarWithEnvVarsK8sAppName(t *testing.T) {
	// prepare
	jaeger := v1.NewJaeger(types.NamespacedName{Name: "my-instance"})
	dep := dep(map[string]string{}, map[string]string{
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
	jaeger := v1.NewJaeger(types.NamespacedName{Name: "my-instance"})
	dep := dep(map[string]string{}, map[string]string{
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
	jaeger := v1.NewJaeger(types.NamespacedName{Name: "my-instance"})
	dep := dep(map[string]string{}, map[string]string{"app": "testapp"})
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
	jaeger := v1.NewJaeger(types.NamespacedName{Name: "my-instance"})
	dep := dep(map[string]string{}, map[string]string{"app": "testapp"})
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
	jaeger := v1.NewJaeger(types.NamespacedName{Name: "my-instance"})
	traceContextEnvVar := corev1.EnvVar{
		Name:  envVarPropagation,
		Value: "tracecontext",
	}
	dep := dep(map[string]string{}, map[string]string{"app": "testapp"})
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

func TestInjectSidecarWithVolumeMounts(t *testing.T) {
	// prepare
	jaeger := v1.NewJaeger(types.NamespacedName{Name: "my-instance"})
	dep := dep(map[string]string{}, map[string]string{})

	agentVolume := corev1.Volume{
		Name: "test-volume1",
		VolumeSource: corev1.VolumeSource{
			Secret: &corev1.SecretVolumeSource{
				SecretName: "test-secret1",
			},
		},
	}
	agentVolumeMount := corev1.VolumeMount{
		Name:      "test-volume1",
		MountPath: "/test-volume1",
		ReadOnly:  true,
	}

	commonVolume := corev1.Volume{
		Name: "test-volume2",
		VolumeSource: corev1.VolumeSource{
			Secret: &corev1.SecretVolumeSource{
				SecretName: "test-secret2",
			},
		},
	}
	commonVolumeMount := corev1.VolumeMount{
		Name:      "test-volume2",
		MountPath: "/test-volume2",
		ReadOnly:  true,
	}

	jaeger.Spec.Agent.Volumes = append(jaeger.Spec.Agent.Volumes, agentVolume)
	jaeger.Spec.Agent.VolumeMounts = append(jaeger.Spec.Agent.VolumeMounts, agentVolumeMount)
	jaeger.Spec.Volumes = append(jaeger.Spec.Volumes, commonVolume)
	jaeger.Spec.VolumeMounts = append(jaeger.Spec.VolumeMounts, commonVolumeMount)

	// test
	dep = Sidecar(jaeger, dep)

	// verify
	assert.Contains(t, dep.Spec.Template.Spec.Volumes, agentVolume)
	assert.NotContains(t, dep.Spec.Template.Spec.Volumes, commonVolume)
	assert.Contains(t, dep.Spec.Template.Spec.Containers[1].Image, "jaeger-agent")
	assert.Contains(t, dep.Spec.Template.Spec.Containers[1].VolumeMounts, agentVolumeMount)
	assert.NotContains(t, dep.Spec.Template.Spec.Containers[1].VolumeMounts, commonVolumeMount)
}

func TestSidecarImagePullSecrets(t *testing.T) {

	deploymentImagePullSecrets := []corev1.LocalObjectReference{{
		Name: "deploymentImagePullSecret",
	}}

	agentImagePullSecrets := []corev1.LocalObjectReference{{
		Name: "agentImagePullSecret",
	}}

	jaeger := v1.NewJaeger(types.NamespacedName{Name: "my-instance"})
	jaeger.Spec.Agent.ImagePullSecrets = agentImagePullSecrets

	dep := dep(map[string]string{}, map[string]string{})
	dep.Spec.Template.Spec.ImagePullSecrets = deploymentImagePullSecrets
	dep = Sidecar(jaeger, dep)

	assert.Len(t, dep.Spec.Template.Spec.ImagePullSecrets, 2)
	assert.Equal(t, dep.Spec.Template.Spec.ImagePullSecrets[0].Name, "deploymentImagePullSecret")
	assert.Equal(t, dep.Spec.Template.Spec.ImagePullSecrets[1].Name, "agentImagePullSecret")
}

func TestSidecarDefaultPorts(t *testing.T) {
	// prepare
	jaeger := v1.NewJaeger(types.NamespacedName{Name: "my-instance"})
	dep := dep(map[string]string{}, map[string]string{"app": "testapp"})

	// test
	dep = Sidecar(jaeger, dep)

	// verify
	assert.Len(t, dep.Spec.Template.Spec.Containers, 2)
	assert.Contains(t, dep.Spec.Template.Spec.Containers[1].Image, "jaeger-agent")

	assert.Len(t, dep.Spec.Template.Spec.Containers[1].Ports, 5)
	assert.Contains(t, dep.Spec.Template.Spec.Containers[1].Ports, corev1.ContainerPort{ContainerPort: 5775, Name: "zk-compact-trft", Protocol: corev1.ProtocolUDP})
	assert.Contains(t, dep.Spec.Template.Spec.Containers[1].Ports, corev1.ContainerPort{ContainerPort: 5778, Name: "config-rest"})
	assert.Contains(t, dep.Spec.Template.Spec.Containers[1].Ports, corev1.ContainerPort{ContainerPort: 6831, Name: "jg-compact-trft", Protocol: corev1.ProtocolUDP})
	assert.Contains(t, dep.Spec.Template.Spec.Containers[1].Ports, corev1.ContainerPort{ContainerPort: 6832, Name: "jg-binary-trft", Protocol: corev1.ProtocolUDP})
	assert.Contains(t, dep.Spec.Template.Spec.Containers[1].Ports, corev1.ContainerPort{ContainerPort: 14271, Name: "admin-http"})
}

func TestSkipInjectSidecar(t *testing.T) {
	// prepare
	jaeger := v1.NewJaeger(types.NamespacedName{Name: "my-instance"})
	dep := dep(map[string]string{}, map[string]string{Label: "non-existing-operator"})

	// test
	dep = Sidecar(jaeger, dep)

	// verify
	assert.Len(t, dep.Spec.Template.Spec.Containers, 1)
	assert.NotContains(t, dep.Spec.Template.Spec.Containers[0].Image, "jaeger-agent")
}

func TestSidecarNeeded(t *testing.T) {

	jaeger := v1.NewJaeger(types.NamespacedName{Name: "some-jaeger-instance"})

	depWithAgent := dep(map[string]string{
		Annotation: "some-jaeger-instance",
	}, map[string]string{})

	depWithAgent = Sidecar(jaeger, depWithAgent)

	explicitInjected := dep(map[string]string{}, map[string]string{})
	explicitInjected.Spec.Template.Spec.Containers = append(explicitInjected.Spec.Template.Spec.Containers, corev1.Container{
		Name: "jaeger-agent",
	})

	tests := []struct {
		dep    *appsv1.Deployment
		ns     *corev1.Namespace
		needed bool
	}{
		{
			dep:    &appsv1.Deployment{},
			ns:     &corev1.Namespace{},
			needed: false,
		},
		{
			dep:    dep(map[string]string{Annotation: "some-jaeger-instance"}, map[string]string{}),
			ns:     ns(map[string]string{}),
			needed: true,
		},
		{
			dep:    dep(map[string]string{Annotation: "some-jaeger-instance"}, map[string]string{}),
			ns:     ns(map[string]string{Annotation: "some-jaeger-instance"}),
			needed: true,
		},
		{
			dep:    dep(map[string]string{}, map[string]string{}),
			ns:     ns(map[string]string{Annotation: "some-jaeger-instance"}),
			needed: true,
		},
		{
			dep:    depWithAgent,
			ns:     ns(map[string]string{}),
			needed: true,
		},
		{
			dep:    dep(map[string]string{}, map[string]string{"app": "jaeger"}),
			ns:     ns(map[string]string{Annotation: "true"}),
			needed: false,
		},
		{
			dep:    explicitInjected,
			ns:     ns(map[string]string{}),
			needed: false,
		},
		{
			dep:    explicitInjected,
			ns:     ns(map[string]string{Annotation: "true"}),
			needed: false,
		},
	}
	for _, test := range tests {
		t.Run(fmt.Sprintf("dep:%s, ns: %s", test.dep.Annotations, test.ns.Annotations), func(t *testing.T) {
			assert.Equal(t, test.needed, Needed(test.dep, test.ns))
			assert.LessOrEqual(t, len(test.dep.Spec.Template.Spec.Containers), 2)
		})
	}
}

func TestSelect(t *testing.T) {
	jTest := v1.NewJaeger(types.NamespacedName{Name: "test"})
	jProd := v1.NewJaeger(types.NamespacedName{Name: "prod"})

	depNsProd := dep(map[string]string{Annotation: "true"}, map[string]string{})
	depNsProd.Namespace = "nsprod"

	jTestNsTest := v1.NewJaeger(types.NamespacedName{Name: "test", Namespace: "nstest"})
	jProdNsProd := v1.NewJaeger(types.NamespacedName{Name: "prod", Namespace: "nsprod"})

	tests := []struct {
		dep      *appsv1.Deployment
		ns       *corev1.Namespace
		jaegers  *v1.JaegerList
		expected *v1.Jaeger
		cap      string
	}{
		{
			dep:      dep(map[string]string{Annotation: "prod"}, map[string]string{}),
			ns:       ns(map[string]string{}),
			jaegers:  &v1.JaegerList{Items: []v1.Jaeger{*jProd}},
			expected: jProd,
			cap:      "dep explicit, ns empty",
		},
		{
			dep:      dep(map[string]string{Annotation: "prod"}, map[string]string{}),
			ns:       ns(map[string]string{Annotation: "true"}),
			jaegers:  &v1.JaegerList{Items: []v1.Jaeger{*jProd}},
			expected: jProd,
			cap:      "dep explicit, ns true",
		},
		{
			dep:      dep(map[string]string{Annotation: "prod"}, map[string]string{}),
			ns:       ns(map[string]string{Annotation: "test"}),
			jaegers:  &v1.JaegerList{Items: []v1.Jaeger{*jProd, *jTest}},
			expected: jProd,
			cap:      "dep explicit, ns explicit",
		},
		{
			dep:      dep(map[string]string{Annotation: "doesNotExist"}, map[string]string{}),
			ns:       ns(map[string]string{Annotation: "test"}),
			jaegers:  &v1.JaegerList{Items: []v1.Jaeger{*jProd, *jTest}},
			expected: nil,
			cap:      "dep explicit does not exist, ns explicit",
		},
		{
			dep:      dep(map[string]string{Annotation: "true"}, map[string]string{}),
			ns:       ns(map[string]string{Annotation: "true"}),
			jaegers:  &v1.JaegerList{Items: []v1.Jaeger{*jProd}},
			expected: jProd,
			cap:      "dep true, ns true",
		},
		{
			dep:      dep(map[string]string{Annotation: "true"}, map[string]string{}),
			ns:       ns(map[string]string{Annotation: "true"}),
			jaegers:  &v1.JaegerList{Items: []v1.Jaeger{*jTest, *jProd}},
			expected: nil,
			cap:      "dep true, ns true, ambiguous",
		},
		{
			dep:      dep(map[string]string{Annotation: "true"}, map[string]string{}),
			ns:       ns(map[string]string{Annotation: "prod"}),
			jaegers:  &v1.JaegerList{Items: []v1.Jaeger{*jTest, *jProd}},
			expected: jProd,
			cap:      "dep true, ns explicit",
		},
		{
			dep:      dep(map[string]string{Annotation: "true"}, map[string]string{}),
			ns:       ns(map[string]string{}),
			jaegers:  &v1.JaegerList{Items: []v1.Jaeger{*jTest}},
			expected: jTest,
			cap:      "dep true, ns missing",
		},
		{
			dep:      dep(map[string]string{}, map[string]string{}),
			ns:       ns(map[string]string{Annotation: "prod"}),
			jaegers:  &v1.JaegerList{Items: []v1.Jaeger{*jTest, *jProd}},
			expected: jProd,
			cap:      "dep none, ns explicit",
		},
		{
			dep:      dep(map[string]string{}, map[string]string{}),
			ns:       ns(map[string]string{Annotation: "true"}),
			jaegers:  &v1.JaegerList{Items: []v1.Jaeger{*jProd}},
			expected: jProd,
			cap:      "dep none, ns true",
		},
		{
			dep:      dep(map[string]string{}, map[string]string{}),
			ns:       ns(map[string]string{Annotation: "true"}),
			jaegers:  &v1.JaegerList{Items: []v1.Jaeger{}},
			expected: nil,
			cap:      "dep none, ns true, no jaegers",
		},
		{
			dep:      depNsProd,
			ns:       ns(map[string]string{}),
			jaegers:  &v1.JaegerList{Items: []v1.Jaeger{*jTestNsTest, *jProdNsProd}},
			expected: jProdNsProd,
			cap:      "dep true, two jaeger instances one in the same ns",
		},
		{
			dep:      depNsProd,
			ns:       ns(map[string]string{Annotation: "true"}),
			jaegers:  &v1.JaegerList{Items: []v1.Jaeger{*jTestNsTest, *jProdNsProd}},
			expected: jProdNsProd,
			cap:      "dep none, ns true, two jaeger instances one in the same ns",
		},
	}

	for _, test := range tests {
		t.Run(test.cap, func(t *testing.T) {
			jaeger := Select(test.dep, test.ns, test.jaegers)
			assert.Equal(t, test.expected, jaeger)
		})
	}
}

func TestSelectBasedOnName(t *testing.T) {
	dep := dep(map[string]string{Annotation: "the-second-jaeger-instance-available"}, map[string]string{})

	jaegerPods := &v1.JaegerList{
		Items: []v1.Jaeger{
			{
				ObjectMeta: metav1.ObjectMeta{
					Name: "the-first-jaeger-instance-available",
				},
			},
			{
				ObjectMeta: metav1.ObjectMeta{
					Name: "the-second-jaeger-instance-available",
				},
			},
		},
	}

	jaeger := Select(dep, &corev1.Namespace{}, jaegerPods)
	assert.NotNil(t, jaeger)
	assert.Equal(t, "the-second-jaeger-instance-available", jaeger.Name)
	assert.Equal(t, "the-second-jaeger-instance-available", dep.Annotations[Annotation])
}

func TestSidecarOrderOfArguments(t *testing.T) {
	jaeger := v1.NewJaeger(types.NamespacedName{Name: "my-instance"})
	jaeger.Spec.Agent.Options = v1.NewOptions(map[string]interface{}{
		"b-option": "b-value",
		"a-option": "a-value",
		"c-option": "c-value",
	})

	dep := dep(map[string]string{}, map[string]string{})
	dep = Sidecar(jaeger, dep)

	assert.Len(t, dep.Spec.Template.Spec.Containers, 2)
	assert.Len(t, dep.Spec.Template.Spec.Containers[1].Args, 5)
	containsOptionWithPrefix(t, dep.Spec.Template.Spec.Containers[1].Args, "--a-option")
	containsOptionWithPrefix(t, dep.Spec.Template.Spec.Containers[1].Args, "--b-option")
	containsOptionWithPrefix(t, dep.Spec.Template.Spec.Containers[1].Args, "--c-option")
	containsOptionWithPrefix(t, dep.Spec.Template.Spec.Containers[1].Args, "--jaeger.tags")
	containsOptionWithPrefix(t, dep.Spec.Template.Spec.Containers[1].Args, "--reporter.grpc.host-port")
	agentTags := agentTags(dep.Spec.Template.Spec.Containers[1].Args)
	assert.Contains(t, agentTags, "container.name=only_container")
}

func TestSidecarExplicitTags(t *testing.T) {
	// prepare
	jaeger := v1.NewJaeger(types.NamespacedName{Name: "my-instance"})
	jaeger.Spec.Agent.Options = v1.NewOptions(map[string]interface{}{"jaeger.tags": "key=val"})
	dep := dep(map[string]string{}, map[string]string{})

	// test
	dep = Sidecar(jaeger, dep)

	// verify
	assert.Len(t, dep.Spec.Template.Spec.Containers, 2)
	agentTags := agentTags(dep.Spec.Template.Spec.Containers[1].Args)
	assert.Equal(t, []string{"key=val"}, agentTags)
}

func TestSidecarCustomReporterPort(t *testing.T) {
	jaeger := v1.NewJaeger(types.NamespacedName{Name: "my-instance"})
	jaeger.Spec.Agent.Options = v1.NewOptions(map[string]interface{}{
		"reporter.grpc.host-port": "collector:5000",
	})

	dep := dep(map[string]string{}, map[string]string{})
	dep = Sidecar(jaeger, dep)

	assert.Len(t, dep.Spec.Template.Spec.Containers, 2)
	assert.Len(t, dep.Spec.Template.Spec.Containers[1].Args, 2)
	assert.Contains(t, dep.Spec.Template.Spec.Containers[1].Args, "--reporter.grpc.host-port=collector:5000")
}

func TestSidecarAgentResources(t *testing.T) {
	jaeger := v1.NewJaeger(types.NamespacedName{Name: "my-instance"})
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

	dep := dep(map[string]string{}, map[string]string{})
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
	instanceName := "my-instance"
	nsn := types.NamespacedName{
		Name:      instanceName,
		Namespace: "Test",
	}
	jaeger := v1.NewJaeger(nsn)
	dep1 := Sidecar(jaeger, dep(map[string]string{}, map[string]string{}))
	assert.Equal(t, 2, len(dep1.Spec.Template.Spec.Containers))
	assert.Equal(t, 0, len(dep1.Spec.Template.Spec.Volumes))
	CleanSidecar(instanceName, dep1)
	assert.Equal(t, 1, len(dep1.Spec.Template.Spec.Containers))
	assert.Equal(t, 0, len(dep1.Spec.Template.Spec.Volumes))
}

func TestCleanSidecarsOpenShift(t *testing.T) {
	// prepare
	viper.Set("platform", v1.FlagPlatformOpenShift)
	defer viper.Reset()

	instanceName := "my-instance"
	nsn := types.NamespacedName{
		Name:      instanceName,
		Namespace: "Test",
	}
	jaeger := v1.NewJaeger(nsn)
	dep1 := Sidecar(jaeger, dep(map[string]string{}, map[string]string{}))

	// sanity check
	require.Equal(t, 2, len(dep1.Spec.Template.Spec.Containers))
	require.Equal(t, 2, len(dep1.Spec.Template.Spec.Volumes))

	// test
	CleanSidecar(instanceName, dep1)

	// verify
	assert.Equal(t, 1, len(dep1.Spec.Template.Spec.Containers))
	assert.Equal(t, 0, len(dep1.Spec.Template.Spec.Volumes))
}

func TestSidecarWithLabel(t *testing.T) {
	nsn := types.NamespacedName{
		Name:      "my-instance",
		Namespace: "Test",
	}
	jaeger := v1.NewJaeger(nsn)
	dep1 := dep(map[string]string{}, map[string]string{})
	dep1 = Sidecar(jaeger, dep1)
	assert.Equal(t, dep1.Labels[Label], "my-instance")
	dep2 := dep(map[string]string{}, map[string]string{})
	dep2.Labels = map[string]string{"anotherLabel": "anotherValue"}
	dep2 = Sidecar(jaeger, dep2)
	assert.Equal(t, len(dep2.Labels), 2)
	assert.Equal(t, dep2.Labels["anotherLabel"], "anotherValue")
	assert.Equal(t, dep2.Labels[Label], jaeger.Name)
}

func TestSidecarWithoutPrometheusAnnotations(t *testing.T) {
	// prepare
	jaeger := v1.NewJaeger(types.NamespacedName{Name: "my-instance"})
	dep := Sidecar(jaeger, dep(map[string]string{}, map[string]string{}))

	// test
	dep = Sidecar(jaeger, dep)

	// verify
	assert.Contains(t, dep.Annotations, "prometheus.io/scrape")
	assert.Contains(t, dep.Annotations, "prometheus.io/port")
}

func TestSidecarWithPrometheusAnnotations(t *testing.T) {
	// prepare
	jaeger := v1.NewJaeger(types.NamespacedName{Name: "my-instance"})
	dep := dep(map[string]string{
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
	jaeger := v1.NewJaeger(types.NamespacedName{Name: "my-instance"})
	dep := Sidecar(jaeger, depWithTwoContainers(map[string]string{}, map[string]string{}))

	assert.Equal(t, dep.Labels[Label], jaeger.Name)
	assert.Len(t, dep.Spec.Template.Spec.Containers, 3, "Expected 3 containers")
	assert.Equal(t, "jaeger-agent", dep.Spec.Template.Spec.Containers[2].Name)
	containsOptionWithPrefix(t, dep.Spec.Template.Spec.Containers[2].Args, "--jaeger.tags")
	agentTags := agentTags(dep.Spec.Template.Spec.Containers[2].Args)
	assert.Equal(t, "", util.FindItem("container.name=", agentTags))
}

func ns(annotations map[string]string) *corev1.Namespace {
	return &corev1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Annotations: annotations,
		},
		Spec: corev1.NamespaceSpec{},
	}
}

func dep(annotations map[string]string, labels map[string]string) *appsv1.Deployment {
	return &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Annotations: annotations,
			Labels:      labels,
		},
		Spec: appsv1.DeploymentSpec{
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: labels,
				},
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{{
						Name: "only_container",
					}},
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

func TestSidecarArgumentsOpenshiftTLS(t *testing.T) {
	viper.Set("platform", v1.FlagPlatformOpenShift)
	defer viper.Reset()

	jaeger := v1.NewJaeger(types.NamespacedName{
		Name:      "my-instance",
		Namespace: "test",
	})
	jaeger.Spec.Agent.Options = v1.NewOptions(map[string]interface{}{
		"a-option": "a-value",
	})

	dep := dep(map[string]string{Annotation: jaeger.Name}, map[string]string{})
	dep = Sidecar(jaeger, dep)

	assert.Len(t, dep.Spec.Template.Spec.Containers, 2)
	assert.Len(t, dep.Spec.Template.Spec.Containers[1].Args, 5)
	assert.Greater(t, len(util.FindItem("--a-option=a-value", dep.Spec.Template.Spec.Containers[1].Args)), 0)
	assert.Greater(t, len(util.FindItem("--jaeger.tags", dep.Spec.Template.Spec.Containers[1].Args)), 0)
	assert.Greater(t, len(util.FindItem("--reporter.grpc.host-port=dns:///my-instance-collector-headless.test.svc:14250", dep.Spec.Template.Spec.Containers[1].Args)), 0)
	assert.Greater(t, len(util.FindItem("--reporter.grpc.tls.enabled=true", dep.Spec.Template.Spec.Containers[1].Args)), 0)
	assert.Greater(t, len(util.FindItem("--reporter.grpc.tls.ca="+ca.ServiceCAPath, dep.Spec.Template.Spec.Containers[1].Args)), 0)
	agentTags := agentTags(dep.Spec.Template.Spec.Containers[1].Args)
	assert.Contains(t, agentTags, "container.name=only_container")
}

func TestEqualSidecar(t *testing.T) {
	jaeger := v1.NewJaeger(types.NamespacedName{
		Name:      "my-instance",
		Namespace: "test",
	})

	dep1 := dep(map[string]string{Annotation: jaeger.Name}, map[string]string{})
	dep1 = Sidecar(jaeger, dep1)

	dep1Equal := dep(map[string]string{Annotation: jaeger.Name}, map[string]string{})
	dep1Equal = Sidecar(jaeger, dep1Equal)
	assert.True(t, EqualSidecar(dep1, dep1Equal))

	// Change flags.
	jaeger.Spec.Agent.Options = v1.NewOptions(map[string]interface{}{
		"--jaeger.tags": "changed-tag=newvalue",
	})

	dep2 := dep(map[string]string{Annotation: jaeger.Name}, map[string]string{})
	dep2 = Sidecar(jaeger, dep2)
	assert.False(t, EqualSidecar(dep1, dep2))

	// When no agent is present on the deploy
	dep3 := dep(map[string]string{Annotation: jaeger.Name}, map[string]string{})
	assert.False(t, EqualSidecar(dep1, dep3))
}

func TestAgentOTELConfig(t *testing.T) {
	jaeger := v1.NewJaeger(types.NamespacedName{Name: "my-instance"})
	jaeger.Spec.Agent.Config = v1.NewFreeForm(map[string]interface{}{"foo": "bar"})

	d := Sidecar(jaeger, &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Labels:      map[string]string{Label: "my-instance"},
			Annotations: map[string]string{},
		},
	})
	assert.True(t, hasArgument("--config=/etc/jaeger/otel/config.yaml", d.Spec.Template.Spec.Containers[0].Args))
	assert.True(t, hasVolume("my-instance-agent-otel-config", d.Spec.Template.Spec.Volumes))
	assert.True(t, hasVolumeMount("my-instance-agent-otel-config", d.Spec.Template.Spec.Containers[0].VolumeMounts))
}

func hasVolume(name string, volumes []corev1.Volume) bool {
	for _, v := range volumes {
		if v.Name == name {
			return true
		}
	}
	return false
}

func hasVolumeMount(name string, volumeMounts []corev1.VolumeMount) bool {
	for _, v := range volumeMounts {
		if v.Name == name {
			return true
		}
	}
	return false
}

func hasArgument(arg string, args []string) bool {
	for _, v := range args {
		if v == arg {
			return true
		}
	}
	return false
}

func TestInjectSidecarOnOpenShift(t *testing.T) {
	viper.Set("platform", v1.FlagPlatformOpenShift)
	defer viper.Reset()

	jaeger := v1.NewJaeger(types.NamespacedName{Name: "my-instance"})
	dep := dep(map[string]string{}, map[string]string{})
	dep = Sidecar(jaeger, dep)
	assert.Equal(t, dep.Labels[Label], jaeger.Name)
	assert.Len(t, dep.Spec.Template.Spec.Containers, 2)
	assert.Len(t, dep.Spec.Template.Spec.Containers[1].VolumeMounts, 2)
	assert.Len(t, dep.Spec.Template.Spec.Volumes, 2)
}
