package inject

import (
	"fmt"
	"testing"

	"github.com/spf13/viper"
	"github.com/stretchr/testify/assert"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"

	v1 "github.com/jaegertracing/jaeger-operator/apis/v1"
)

func TestSelectForPod(t *testing.T) {
	tests := []struct {
		name                string
		pod                 metav1.ObjectMeta
		ns                  metav1.ObjectMeta
		availableJaegerPods *v1.JaegerList
		isMatch             bool
	}{
		{
			name: "nil",
			pod: metav1.ObjectMeta{
				Annotations: map[string]string{},
				Namespace:   "",
			},
			ns: metav1.ObjectMeta{
				Annotations: map[string]string{},
				Namespace:   "",
			},
			availableJaegerPods: &v1.JaegerList{
				Items: []v1.Jaeger{},
			},
		},
		{
			name: "pod match",
			pod: metav1.ObjectMeta{
				Annotations: map[string]string{
					Annotation: "123",
				},
				Namespace: "",
			},
			ns: metav1.ObjectMeta{
				Annotations: map[string]string{},
				Namespace:   "",
			},
			availableJaegerPods: &v1.JaegerList{
				Items: []v1.Jaeger{
					{
						ObjectMeta: metav1.ObjectMeta{Name: "123"},
					},
				},
			},
			isMatch: true,
		},
		{
			name: "ns match",
			pod: metav1.ObjectMeta{
				Annotations: map[string]string{},
				Namespace:   "",
			},
			ns: metav1.ObjectMeta{
				Annotations: map[string]string{
					Annotation: "123",
				},
				Namespace: "",
			},
			availableJaegerPods: &v1.JaegerList{
				Items: []v1.Jaeger{
					{
						ObjectMeta: metav1.ObjectMeta{Name: "123"},
					},
				},
			},
			isMatch: true,
		},
		{
			name: "pod true",
			pod: metav1.ObjectMeta{
				Annotations: map[string]string{
					Annotation: "true",
				},
				Namespace: "",
			},
			ns: metav1.ObjectMeta{
				Annotations: map[string]string{},
				Namespace:   "",
			},
			availableJaegerPods: &v1.JaegerList{
				Items: []v1.Jaeger{
					{
						ObjectMeta: metav1.ObjectMeta{Name: "123"},
					},
				},
			},
			isMatch: true,
		},
		{
			name: "ns true",
			pod: metav1.ObjectMeta{
				Annotations: map[string]string{},
				Namespace:   "",
			},
			ns: metav1.ObjectMeta{
				Annotations: map[string]string{
					Annotation: "true",
				},
				Namespace: "",
			},
			availableJaegerPods: &v1.JaegerList{
				Items: []v1.Jaeger{
					{
						ObjectMeta: metav1.ObjectMeta{Name: "123"},
					},
				},
			},
			isMatch: true,
		},
		{
			name: "jaeger in namespace",
			pod: metav1.ObjectMeta{
				Annotations: map[string]string{},
				Namespace:   "abc",
			},
			ns: metav1.ObjectMeta{
				Annotations: map[string]string{
					Annotation: "true",
				},
				Namespace: "",
			},
			availableJaegerPods: &v1.JaegerList{
				Items: []v1.Jaeger{
					{
						ObjectMeta: metav1.ObjectMeta{
							Name:      "123",
							Namespace: "abc",
						},
					},
					{
						ObjectMeta: metav1.ObjectMeta{
							Name:      "234",
							Namespace: "def",
						},
					},
				},
			},
			isMatch: true,
		},
		{
			name: "jaeger not in namespace",
			pod: metav1.ObjectMeta{
				Annotations: map[string]string{},
				Namespace:   "xyz",
			},
			ns: metav1.ObjectMeta{
				Annotations: map[string]string{
					Annotation: "true",
				},
				Namespace: "",
			},
			availableJaegerPods: &v1.JaegerList{
				Items: []v1.Jaeger{
					{
						ObjectMeta: metav1.ObjectMeta{
							Name:      "123",
							Namespace: "abc",
						},
					},
					{
						ObjectMeta: metav1.ObjectMeta{
							Name:      "234",
							Namespace: "def",
						},
					},
				},
			},
		},
		{
			name: "multiple jaegers in namespace",
			pod: metav1.ObjectMeta{
				Annotations: map[string]string{},
				Namespace:   "abc",
			},
			ns: metav1.ObjectMeta{
				Annotations: map[string]string{
					Annotation: "true",
				},
				Namespace: "",
			},
			availableJaegerPods: &v1.JaegerList{
				Items: []v1.Jaeger{
					{
						ObjectMeta: metav1.ObjectMeta{
							Name:      "123",
							Namespace: "abc",
						},
					},
					{
						ObjectMeta: metav1.ObjectMeta{
							Name:      "234",
							Namespace: "abc",
						},
					},
				},
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			pod := &corev1.Pod{ObjectMeta: tc.pod}
			ns := &corev1.Namespace{ObjectMeta: tc.ns}
			j := SelectForPod(pod, ns, tc.availableJaegerPods)
			if tc.isMatch {
				assert.NotNil(t, j)
			} else {
				assert.Nil(t, j)
			}
		})
	}

}

func TestInjectSidecarInPod(t *testing.T) {
	jaeger := v1.NewJaeger(types.NamespacedName{Name: "my-instance"})
	pod := pod(map[string]string{}, map[string]string{})
	pod = SidecarPod(jaeger, pod)
	assert.Equal(t, pod.Labels[Label], jaeger.Name)
	assert.Len(t, pod.Spec.Containers, 2)
	assert.Contains(t, pod.Spec.Containers[1].Image, "jaeger-agent")
	assert.Len(t, pod.Spec.Containers[0].Env, 0)
	assert.Len(t, pod.Spec.Containers[1].VolumeMounts, 0)
	assert.Len(t, pod.Spec.Volumes, 0)
}

func TestInjectSidecarInPodOpenShift(t *testing.T) {
	viper.Set("platform", v1.FlagPlatformOpenShift)
	defer reset()

	jaeger := v1.NewJaeger(types.NamespacedName{Name: "my-instance"})
	assert.Len(t, jaeger.Spec.Agent.VolumeMounts, 0)
	assert.Len(t, jaeger.Spec.Agent.Volumes, 0)

	pod := pod(map[string]string{}, map[string]string{})
	pod = SidecarPod(jaeger, pod)
	assert.Equal(t, pod.Labels[Label], jaeger.Name)
	assert.Len(t, pod.Spec.Containers, 2)
	assert.Contains(t, pod.Spec.Containers[1].Image, "jaeger-agent")
	assert.Len(t, pod.Spec.Containers[0].Env, 0)
	assert.Len(t, pod.Spec.Containers[1].VolumeMounts, 2)
	assert.Len(t, pod.Spec.Volumes, 2)

	// CR should not be touched.
	assert.Len(t, jaeger.Spec.Agent.VolumeMounts, 0)
	assert.Len(t, jaeger.Spec.Agent.Volumes, 0)
}

func TestInjectSidecarPodWithEnvVars(t *testing.T) {
	// prepare
	jaeger := v1.NewJaeger(types.NamespacedName{Name: "my-instance"})
	pod := pod(map[string]string{Annotation: jaeger.Name}, map[string]string{"app": "testapp"})

	// test
	pod = SidecarPod(jaeger, pod)

	// verify
	assert.Len(t, pod.Spec.Containers, 2)
	assert.Contains(t, pod.Spec.Containers[1].Image, "jaeger-agent")
	containsEnvVarNamed(t, pod.Spec.Containers[1].Env, envVarPodName)
	containsEnvVarNamed(t, pod.Spec.Containers[1].Env, envVarHostIP)

	assert.Len(t, pod.Spec.Containers[0].Env, 2)
	assert.Contains(t, pod.Spec.Containers[0].Env, corev1.EnvVar{Name: envVarPropagation, Value: "jaeger,b3,w3c"})
	assert.Contains(t, pod.Spec.Containers[0].Env, corev1.EnvVar{Name: envVarServiceName, Value: "testapp.default"})
}

func TestInjectSidecarInPodWithEnvVarsK8sAppName(t *testing.T) {
	// prepare
	jaeger := v1.NewJaeger(types.NamespacedName{Name: "my-instance"})
	pod := pod(map[string]string{}, map[string]string{
		"app":                    "noapp",
		"app.kubernetes.io/name": "testapp",
	})

	// test
	pod = SidecarPod(jaeger, pod)

	// verify
	assert.Len(t, pod.Spec.Containers, 2)
	assert.Len(t, pod.Spec.Containers[0].Env, 2)
	assert.Contains(t, pod.Spec.Containers[0].Env, corev1.EnvVar{Name: envVarServiceName, Value: "testapp.default"})
}

func TestInjectSidecarInPodWithEnvVarsK8sAppInstance(t *testing.T) {
	// prepare
	jaeger := v1.NewJaeger(types.NamespacedName{Name: "my-instance"})
	pod := pod(map[string]string{}, map[string]string{
		"app":                        "noapp",
		"app.kubernetes.io/name":     "noname",
		"app.kubernetes.io/instance": "testapp",
	})

	// test
	pod = SidecarPod(jaeger, pod)

	// verify
	assert.Len(t, pod.Spec.Containers, 2)
	assert.Len(t, pod.Spec.Containers[0].Env, 2)
	assert.Contains(t, pod.Spec.Containers[0].Env, corev1.EnvVar{Name: envVarServiceName, Value: "testapp.default"})
}

func TestInjectSidecarInPodWithEnvVarsWithNamespace(t *testing.T) {
	// prepare
	jaeger := v1.NewJaeger(types.NamespacedName{Name: "my-instance"})
	pod := pod(map[string]string{}, map[string]string{"app": "testapp"})
	pod.Namespace = "mynamespace"

	// test
	pod = SidecarPod(jaeger, pod)

	// verify
	assert.Len(t, pod.Spec.Containers, 2)
	assert.Contains(t, pod.Spec.Containers[1].Image, "jaeger-agent")

	assert.Len(t, pod.Spec.Containers[0].Env, 2)
	assert.Contains(t, pod.Spec.Containers[0].Env, corev1.EnvVar{Name: envVarServiceName, Value: "testapp.mynamespace"})
	assert.Contains(t, pod.Spec.Containers[0].Env, corev1.EnvVar{Name: envVarPropagation, Value: "jaeger,b3,w3c"})
}

func TestInjectSidecarInPodWithEnvVarsOverrideName(t *testing.T) {
	// prepare
	jaeger := v1.NewJaeger(types.NamespacedName{Name: "my-instance"})
	pod := pod(map[string]string{}, map[string]string{"app": "testapp"})
	envVar := corev1.EnvVar{
		Name:  envVarServiceName,
		Value: "otherapp",
	}
	pod.Spec.Containers[0].Env = append(pod.Spec.Containers[0].Env, envVar)

	// test
	pod = SidecarPod(jaeger, pod)

	// verify
	assert.Len(t, pod.Spec.Containers, 2)
	assert.Contains(t, pod.Spec.Containers[1].Image, "jaeger-agent")

	assert.Len(t, pod.Spec.Containers[0].Env, 2)
	assert.Contains(t, pod.Spec.Containers[0].Env, envVar)
	assert.Contains(t, pod.Spec.Containers[0].Env, corev1.EnvVar{Name: envVarPropagation, Value: "jaeger,b3,w3c"})
}

func TestInjectSidecarInPodWithEnvVarsOverridePropagation(t *testing.T) {
	// prepare
	jaeger := v1.NewJaeger(types.NamespacedName{Name: "my-instance"})
	traceContextEnvVar := corev1.EnvVar{
		Name:  envVarPropagation,
		Value: "tracecontext",
	}
	pod := pod(map[string]string{}, map[string]string{"app": "testapp"})
	pod.Spec.Containers[0].Env = append(pod.Spec.Containers[0].Env, traceContextEnvVar)

	// test
	pod = SidecarPod(jaeger, pod)

	// verify
	assert.Len(t, pod.Spec.Containers, 2)
	assert.Contains(t, pod.Spec.Containers[1].Image, "jaeger-agent")

	assert.Len(t, pod.Spec.Containers[0].Env, 2)
	assert.Contains(t, pod.Spec.Containers[0].Env, traceContextEnvVar)
	assert.Contains(t, pod.Spec.Containers[0].Env, corev1.EnvVar{Name: envVarServiceName, Value: "testapp.default"})
}

func TestInjectSidecarInPodWithVolumeMounts(t *testing.T) {
	// prepare
	jaeger := v1.NewJaeger(types.NamespacedName{Name: "my-instance"})
	pod := pod(map[string]string{}, map[string]string{})

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
	pod = SidecarPod(jaeger, pod)

	// verify
	assert.Contains(t, pod.Spec.Volumes, agentVolume)
	assert.NotContains(t, pod.Spec.Volumes, commonVolume)
	assert.Contains(t, pod.Spec.Containers[1].Image, "jaeger-agent")
	assert.Contains(t, pod.Spec.Containers[1].VolumeMounts, agentVolumeMount)
	assert.NotContains(t, pod.Spec.Containers[1].VolumeMounts, commonVolumeMount)
}

func TestSidecarInPodImagePullSecrets(t *testing.T) {

	podloymentImagePullSecrets := []corev1.LocalObjectReference{{
		Name: "podloymentImagePullSecret",
	}}

	agentImagePullSecrets := []corev1.LocalObjectReference{{
		Name: "agentImagePullSecret",
	}}

	jaeger := v1.NewJaeger(types.NamespacedName{Name: "my-instance"})
	jaeger.Spec.Agent.ImagePullSecrets = agentImagePullSecrets

	pod := pod(map[string]string{}, map[string]string{})
	pod.Spec.ImagePullSecrets = podloymentImagePullSecrets
	pod = SidecarPod(jaeger, pod)

	assert.Len(t, pod.Spec.ImagePullSecrets, 2)
	assert.Equal(t, pod.Spec.ImagePullSecrets[0].Name, "podloymentImagePullSecret")
	assert.Equal(t, pod.Spec.ImagePullSecrets[1].Name, "agentImagePullSecret")
}

func TestSidecarInPodDefaultPorts(t *testing.T) {
	// prepare
	jaeger := v1.NewJaeger(types.NamespacedName{Name: "my-instance"})
	pod := pod(map[string]string{}, map[string]string{"app": "testapp"})

	// test
	pod = SidecarPod(jaeger, pod)

	// verify
	assert.Len(t, pod.Spec.Containers, 2)
	assert.Contains(t, pod.Spec.Containers[1].Image, "jaeger-agent")

	assert.Len(t, pod.Spec.Containers[1].Ports, 5)
	assert.Contains(t, pod.Spec.Containers[1].Ports, corev1.ContainerPort{ContainerPort: 5775, Name: "zk-compact-trft", Protocol: corev1.ProtocolUDP})
	assert.Contains(t, pod.Spec.Containers[1].Ports, corev1.ContainerPort{ContainerPort: 5778, Name: "config-rest"})
	assert.Contains(t, pod.Spec.Containers[1].Ports, corev1.ContainerPort{ContainerPort: 6831, Name: "jg-compact-trft", Protocol: corev1.ProtocolUDP})
	assert.Contains(t, pod.Spec.Containers[1].Ports, corev1.ContainerPort{ContainerPort: 6832, Name: "jg-binary-trft", Protocol: corev1.ProtocolUDP})
	assert.Contains(t, pod.Spec.Containers[1].Ports, corev1.ContainerPort{ContainerPort: 14271, Name: "admin-http"})
}

func TestSkipInjectSidecarInPod(t *testing.T) {
	// prepare
	jaeger := v1.NewJaeger(types.NamespacedName{Name: "my-instance"})
	pod := pod(map[string]string{}, map[string]string{Label: "non-existing-operator"})

	// test
	pod = SidecarPod(jaeger, pod)

	// verify
	assert.Len(t, pod.Spec.Containers, 1)
	assert.NotContains(t, pod.Spec.Containers[0].Image, "jaeger-agent")
}

func TestSidecarInPodNeeded(t *testing.T) {

	jaeger := v1.NewJaeger(types.NamespacedName{Name: "some-jaeger-instance"})

	podWithAgent := pod(map[string]string{
		Annotation: "some-jaeger-instance",
	}, map[string]string{})

	podWithAgent = SidecarPod(jaeger, podWithAgent)

	explicitInjected := pod(map[string]string{}, map[string]string{})
	explicitInjected.Spec.Containers = append(explicitInjected.Spec.Containers, corev1.Container{
		Name: "jaeger-agent",
	})

	tests := []struct {
		pod    *corev1.Pod
		ns     *corev1.Namespace
		needed bool
	}{
		{
			pod:    &corev1.Pod{},
			ns:     &corev1.Namespace{},
			needed: false,
		},
		{ // annotation cannot work anymore
			pod:    pod(map[string]string{}, map[string]string{}),
			ns:     ns(map[string]string{}),
			needed: false,
		},
		{
			pod:    pod(map[string]string{Annotation: "some-jaeger-instance"}, map[string]string{}),
			ns:     ns(map[string]string{}),
			needed: true,
		},
		{
			pod:    pod(map[string]string{Annotation: "some-jaeger-instance"}, map[string]string{}),
			ns:     nsWithLabel(map[string]string{Annotation: "true"}, map[string]string{}),
			needed: true,
		},
		{
			pod:    pod(map[string]string{}, map[string]string{}),
			ns:     nsWithLabel(map[string]string{Annotation: "true"}, map[string]string{}),
			needed: true,
		},
		{
			pod:    pod(map[string]string{}, map[string]string{}),
			ns:     nsWithLabel(map[string]string{Annotation: "false"}, map[string]string{}),
			needed: false,
		},
		{
			pod:    pod(map[string]string{}, map[string]string{}),
			ns:     nsWithLabel(map[string]string{}, map[string]string{Label: "strange-value"}),
			needed: false,
		},
		{
			pod:    pod(map[string]string{}, map[string]string{}),
			ns:     ns(map[string]string{Annotation: "true"}),
			needed: true,
		},
		{
			pod:    pod(map[string]string{}, map[string]string{}),
			ns:     ns(map[string]string{Label: "some-jaeger-instance"}),
			needed: false,
		},
		{
			pod:    podWithAgent,
			ns:     ns(map[string]string{}),
			needed: false, // we will not inject or update for pods already being injected
		},
		{
			pod:    pod(map[string]string{}, map[string]string{"app": "jaeger"}),
			ns:     ns(map[string]string{Annotation: "true"}),
			needed: false,
		},
		{
			pod:    explicitInjected,
			ns:     ns(map[string]string{}),
			needed: false,
		},
		{
			pod:    explicitInjected,
			ns:     ns(map[string]string{Annotation: "true"}),
			needed: false,
		},
		{
			pod:    pod(map[string]string{Annotation: "false"}, map[string]string{}),
			ns:     ns(map[string]string{}),
			needed: false,
		},
	}
	for _, test := range tests {
		t.Run(fmt.Sprintf("pod:%s, ns: %s", test.pod.Annotations, test.ns.Annotations), func(t *testing.T) {
			assert.Equal(t, test.needed, PodNeeded(test.pod, test.ns))
			assert.LessOrEqual(t, len(test.pod.Spec.Containers), 2)
		})
	}
}

func TestPod_SidecarOrderOfArguments(t *testing.T) {
	jaeger := v1.NewJaeger(types.NamespacedName{Name: "my-instance"})
	jaeger.Spec.Agent.Options = v1.NewOptions(map[string]interface{}{
		"b-option": "b-value",
		"a-option": "a-value",
		"c-option": "c-value",
	})

	pod := pod(map[string]string{}, map[string]string{})
	pod = SidecarPod(jaeger, pod)

	assert.Len(t, pod.Spec.Containers, 2)
	assert.Len(t, pod.Spec.Containers[1].Args, 5)
	containsOptionWithPrefix(t, pod.Spec.Containers[1].Args, "--a-option")
	containsOptionWithPrefix(t, pod.Spec.Containers[1].Args, "--b-option")
	containsOptionWithPrefix(t, pod.Spec.Containers[1].Args, "--c-option")
	containsOptionWithPrefix(t, pod.Spec.Containers[1].Args, "--agent.tags")
	containsOptionWithPrefix(t, pod.Spec.Containers[1].Args, "--reporter.grpc.host-port")
	agentTagsMap := parseAgentTags(pod.Spec.Containers[1].Args)
	assert.Equal(t, agentTagsMap["container.name"], "only_container")
}

func TestPod_SidecarExplicitTags(t *testing.T) {
	// prepare
	jaeger := v1.NewJaeger(types.NamespacedName{Name: "my-instance"})
	jaeger.Spec.Agent.Options = v1.NewOptions(map[string]interface{}{"agent.tags": "key=val"})
	pod := pod(map[string]string{}, map[string]string{})

	// test
	pod = SidecarPod(jaeger, pod)

	// verify
	assert.Len(t, pod.Spec.Containers, 2)
	agentTags := parseAgentTags(pod.Spec.Containers[1].Args)
	assert.Equal(t, agentTags, map[string]string{"key": "val"})
}

func TestPod_SidecarCustomReporterPort(t *testing.T) {
	jaeger := v1.NewJaeger(types.NamespacedName{Name: "my-instance"})
	jaeger.Spec.Agent.Options = v1.NewOptions(map[string]interface{}{
		"reporter.grpc.host-port": "collector:5000",
	})

	pod := pod(map[string]string{}, map[string]string{})
	pod = SidecarPod(jaeger, pod)

	assert.Len(t, pod.Spec.Containers, 2)
	assert.Len(t, pod.Spec.Containers[1].Args, 2)
	assert.Contains(t, pod.Spec.Containers[1].Args, "--reporter.grpc.host-port=collector:5000")
}

func TestPod_SidecarAgentResources(t *testing.T) {
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

	pod := pod(map[string]string{}, map[string]string{})
	pod = SidecarPod(jaeger, pod)

	assert.Len(t, pod.Spec.Containers, 2, "Expected 2 containers")
	assert.Equal(t, "jaeger-agent", pod.Spec.Containers[1].Name)
	assert.Equal(t, *resource.NewQuantity(2048, resource.BinarySI), pod.Spec.Containers[1].Resources.Limits[corev1.ResourceLimitsCPU])
	assert.Equal(t, *resource.NewQuantity(2048, resource.BinarySI), pod.Spec.Containers[1].Resources.Requests[corev1.ResourceRequestsCPU])
	assert.Equal(t, *resource.NewQuantity(123, resource.DecimalSI), pod.Spec.Containers[1].Resources.Limits[corev1.ResourceLimitsMemory])
	assert.Equal(t, *resource.NewQuantity(123, resource.DecimalSI), pod.Spec.Containers[1].Resources.Requests[corev1.ResourceRequestsMemory])
	assert.Equal(t, *resource.NewQuantity(512, resource.DecimalSI), pod.Spec.Containers[1].Resources.Limits[corev1.ResourceLimitsEphemeralStorage])
	assert.Equal(t, *resource.NewQuantity(512, resource.DecimalSI), pod.Spec.Containers[1].Resources.Requests[corev1.ResourceRequestsEphemeralStorage])
}

func TestPod_SidecarWithLabel(t *testing.T) {
	nsn := types.NamespacedName{
		Name:      "my-instance",
		Namespace: "Test",
	}
	jaeger := v1.NewJaeger(nsn)
	pod1 := pod(map[string]string{}, map[string]string{})
	pod1 = SidecarPod(jaeger, pod1)
	assert.Equal(t, pod1.Labels[Label], "my-instance")
	pod2 := pod(map[string]string{}, map[string]string{})
	pod2.Labels = map[string]string{"anotherLabel": "anotherValue"}
	pod2 = SidecarPod(jaeger, pod2)
	assert.Equal(t, len(pod2.Labels), 2)
	assert.Equal(t, pod2.Labels["anotherLabel"], "anotherValue")
	assert.Equal(t, pod2.Labels[Label], jaeger.Name)
}

func TestPod_SidecarWithoutPrometheusAnnotations(t *testing.T) {
	// prepare
	jaeger := v1.NewJaeger(types.NamespacedName{Name: "my-instance"})
	pod := SidecarPod(jaeger, pod(map[string]string{}, map[string]string{}))

	// test
	pod = SidecarPod(jaeger, pod)

	// verify
	assert.Contains(t, pod.Annotations, "prometheus.io/scrape")
	assert.Contains(t, pod.Annotations, "prometheus.io/port")
}

func TestPod_SidecarWithPrometheusAnnotations(t *testing.T) {
	// prepare
	jaeger := v1.NewJaeger(types.NamespacedName{Name: "my-instance"})
	pod := pod(map[string]string{
		"prometheus.io/scrape": "false",
		"prometheus.io/port":   "9090",
	}, map[string]string{})

	// test
	pod = SidecarPod(jaeger, pod)

	// verify
	assert.Equal(t, pod.Annotations["prometheus.io/scrape"], "false")
	assert.Equal(t, pod.Annotations["prometheus.io/port"], "9090")
}

func TestPod_SidecarAgentTagsWithMultipleContainers(t *testing.T) {
	jaeger := v1.NewJaeger(types.NamespacedName{Name: "my-instance"})
	pod := SidecarPod(jaeger, podWithTwoContainers(map[string]string{}, map[string]string{}))

	assert.Equal(t, pod.Labels[Label], jaeger.Name)
	assert.Len(t, pod.Spec.Containers, 3, "Expected 3 containers")
	assert.Equal(t, "jaeger-agent", pod.Spec.Containers[2].Name)
	containsOptionWithPrefix(t, pod.Spec.Containers[2].Args, "--agent.tags")
	agentTagsMap := parseAgentTags(pod.Spec.Containers[2].Args)
	assert.NotContains(t, agentTagsMap, "container.name")
}

func TestPod_SidecarAgentContainerNameTagWithDoubleInjectedContainer(t *testing.T) {
	jaeger := v1.NewJaeger(types.NamespacedName{Name: "my-instance"})
	pod := SidecarPod(jaeger, pod(map[string]string{}, map[string]string{}))

	// inject - 1st time
	assert.Equal(t, pod.Labels[Label], jaeger.Name)
	assert.Len(t, pod.Spec.Containers, 2, "Expected 2 containers")
	assert.Equal(t, "jaeger-agent", pod.Spec.Containers[1].Name)
	containsOptionWithPrefix(t, pod.Spec.Containers[1].Args, "--agent.tags")
	agentTagsMap := parseAgentTags(pod.Spec.Containers[1].Args)
	assert.Equal(t, agentTagsMap["container.name"], "only_container")

	// inject - 2nd time due to pod/namespace reconciliation
	pod = SidecarPod(jaeger, pod)
	assert.Len(t, pod.Spec.Containers, 2, "Expected 2 containers")
	assert.Equal(t, "jaeger-agent", pod.Spec.Containers[1].Name)
	containsOptionWithPrefix(t, pod.Spec.Containers[1].Args, "--agent.tags")
	agentTagsMap = parseAgentTags(pod.Spec.Containers[1].Args)
	assert.Equal(t, agentTagsMap["container.name"], "only_container")
}

func pod(annotations map[string]string, labels map[string]string) *corev1.Pod {
	return &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Annotations: annotations,
			Labels:      labels,
		},
		Spec: corev1.PodSpec{
			Containers: []corev1.Container{{
				Name: "only_container",
			}},
		},
	}
}

func podWithTwoContainers(annotations map[string]string, labels map[string]string) *corev1.Pod {
	pod := pod(annotations, labels)
	pod.Spec.Containers[0].Name = "container_0"
	pod.Spec.Containers = append(pod.Spec.Containers, corev1.Container{
		Name: "container_1",
	})
	return pod
}

func nsWithLabel(annotations, labels map[string]string) *corev1.Namespace {
	return &corev1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Annotations: annotations,
			Labels:      labels,
		},
		Spec: corev1.NamespaceSpec{},
	}
}
