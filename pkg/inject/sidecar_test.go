package inject

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/spf13/viper"
	"github.com/stretchr/testify/assert"
	admission "k8s.io/api/admission/v1beta1"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/kubernetes/scheme"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"

	v1 "github.com/jaegertracing/jaeger-operator/pkg/apis/jaegertracing/v1"
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
	pod := pod(map[string]string{Annotation: jaeger.Name}, map[string]string{})
	pod = Sidecar(jaeger, pod)
	assert.Len(t, pod.Spec.Containers, 2)
	assert.Contains(t, pod.Spec.Containers[1].Image, "jaeger-agent")
	assert.Len(t, pod.Spec.Containers[0].Env, 0)
}

func TestInjectSidecarWithLegacyAnnotation(t *testing.T) {
	jaeger := v1.NewJaeger("TestInjectSidecarWithLegacyAnnotation")
	pod := pod(map[string]string{AnnotationLegacy: jaeger.Name}, map[string]string{})
	pod = Sidecar(jaeger, pod)
	assert.Len(t, pod.Spec.Containers, 2)
	assert.Contains(t, pod.Spec.Containers[1].Image, "jaeger-agent")
	assert.Len(t, pod.Spec.Containers[0].Env, 0)
}

func TestInjectSidecarWithEnvVars(t *testing.T) {
	jaeger := v1.NewJaeger("TestInjectSidecarWithEnvVars")
	pod := pod(map[string]string{Annotation: jaeger.Name}, map[string]string{"app": "testapp"})
	pod = Sidecar(jaeger, pod)
	assert.Len(t, pod.Spec.Containers, 2)
	assert.Contains(t, pod.Spec.Containers[1].Image, "jaeger-agent")
	assert.Len(t, pod.Spec.Containers[0].Env, 2)
	assert.Equal(t, envVarServiceName, pod.Spec.Containers[0].Env[0].Name)
	assert.Equal(t, "testapp.default", pod.Spec.Containers[0].Env[0].Value)
	assert.Equal(t, envVarPropagation, pod.Spec.Containers[0].Env[1].Name)
	assert.Equal(t, "jaeger,b3", pod.Spec.Containers[0].Env[1].Value)
}

func TestInjectSidecarWithEnvVarsK8sAppName(t *testing.T) {
	jaeger := v1.NewJaeger("TestInjectSidecarWithEnvVarsK8sAppName")
	pod := pod(map[string]string{Annotation: jaeger.Name}, map[string]string{
		"app":                    "noapp",
		"app.kubernetes.io/name": "testapp",
	})
	pod = Sidecar(jaeger, pod)
	assert.Len(t, pod.Spec.Containers, 2)
	assert.Len(t, pod.Spec.Containers[0].Env, 2)
	assert.Equal(t, envVarServiceName, pod.Spec.Containers[0].Env[0].Name)
	assert.Equal(t, "testapp.default", pod.Spec.Containers[0].Env[0].Value)
}

func TestInjectSidecarWithEnvVarsK8sAppInstance(t *testing.T) {
	jaeger := v1.NewJaeger("TestInjectSidecarWithEnvVarsK8sAppInstance")
	pod := pod(map[string]string{Annotation: jaeger.Name}, map[string]string{
		"app":                        "noapp",
		"app.kubernetes.io/name":     "noname",
		"app.kubernetes.io/instance": "testapp",
	})
	pod = Sidecar(jaeger, pod)
	assert.Len(t, pod.Spec.Containers, 2)
	assert.Len(t, pod.Spec.Containers[0].Env, 2)
	assert.Equal(t, envVarServiceName, pod.Spec.Containers[0].Env[0].Name)
	assert.Equal(t, "testapp.default", pod.Spec.Containers[0].Env[0].Value)
}

func TestInjectSidecarWithEnvVarsWithNamespace(t *testing.T) {
	jaeger := v1.NewJaeger("TestInjectSidecarWithEnvVarsWithNamespace")
	pod := pod(map[string]string{Annotation: jaeger.Name}, map[string]string{"app": "testapp"})
	pod.Namespace = "mynamespace"
	pod = Sidecar(jaeger, pod)
	assert.Len(t, pod.Spec.Containers, 2)
	assert.Contains(t, pod.Spec.Containers[1].Image, "jaeger-agent")
	assert.Len(t, pod.Spec.Containers[0].Env, 2)
	assert.Equal(t, envVarServiceName, pod.Spec.Containers[0].Env[0].Name)
	assert.Equal(t, "testapp.mynamespace", pod.Spec.Containers[0].Env[0].Value)
	assert.Equal(t, envVarPropagation, pod.Spec.Containers[0].Env[1].Name)
	assert.Equal(t, "jaeger,b3", pod.Spec.Containers[0].Env[1].Value)
}

func TestInjectSidecarWithEnvVarsOverrideName(t *testing.T) {
	jaeger := v1.NewJaeger("TestInjectSidecarWithEnvVarsOverrideName")
	pod := pod(map[string]string{Annotation: jaeger.Name}, map[string]string{"app": "testapp"})
	pod.Spec.Containers[0].Env = append(pod.Spec.Containers[0].Env, corev1.EnvVar{
		Name:  envVarServiceName,
		Value: "otherapp",
	})
	pod = Sidecar(jaeger, pod)
	assert.Len(t, pod.Spec.Containers, 2)
	assert.Contains(t, pod.Spec.Containers[1].Image, "jaeger-agent")
	assert.Len(t, pod.Spec.Containers[0].Env, 2)
	assert.Equal(t, envVarServiceName, pod.Spec.Containers[0].Env[0].Name)
	// Explicitly provided env var is used instead of injected "app.namespace" value
	assert.Equal(t, "otherapp", pod.Spec.Containers[0].Env[0].Value)
	assert.Equal(t, envVarPropagation, pod.Spec.Containers[0].Env[1].Name)
	assert.Equal(t, "jaeger,b3", pod.Spec.Containers[0].Env[1].Value)
}

func TestInjectSidecarWithEnvVarsOverridePropagation(t *testing.T) {
	jaeger := v1.NewJaeger("TestInjectSidecarWithEnvVarsOverridePropagation")
	pod := pod(map[string]string{Annotation: jaeger.Name}, map[string]string{"app": "testapp"})
	pod.Spec.Containers[0].Env = append(pod.Spec.Containers[0].Env, corev1.EnvVar{
		Name:  envVarPropagation,
		Value: "tracecontext",
	})
	pod = Sidecar(jaeger, pod)
	assert.Len(t, pod.Spec.Containers, 2)
	assert.Contains(t, pod.Spec.Containers[1].Image, "jaeger-agent")
	assert.Len(t, pod.Spec.Containers[0].Env, 2)
	assert.Equal(t, envVarPropagation, pod.Spec.Containers[0].Env[0].Name)
	// Explicitly provided propagation env var used instead of injected "jaeger,b3" value
	assert.Equal(t, "tracecontext", pod.Spec.Containers[0].Env[0].Value)
	assert.Equal(t, envVarServiceName, pod.Spec.Containers[0].Env[1].Name)
	assert.Equal(t, "testapp.default", pod.Spec.Containers[0].Env[1].Value)
}

func TestSkipInjectSidecar(t *testing.T) {
	jaeger := v1.NewJaeger("TestSkipInjectSidecar")
	pod := pod(map[string]string{Annotation: "non-existing-operator"}, map[string]string{})
	pod = Sidecar(jaeger, pod)
	assert.Len(t, pod.Spec.Containers, 1)
	assert.NotContains(t, pod.Spec.Containers[0].Image, "jaeger-agent")
}

func TestSidecarNotNeeded(t *testing.T) {
	pod := corev1.Pod{
		Spec: corev1.PodSpec{
			Containers: []corev1.Container{
				corev1.Container{},
			},
		},
	}

	assert.False(t, Needed(pod))
}

func TestSidecarNeeded(t *testing.T) {
	pod := pod(map[string]string{Annotation: "some-jaeger-instance"}, map[string]string{})
	assert.True(t, Needed(pod))
}

func TestHasSidecarAlready(t *testing.T) {
	pod := pod(map[string]string{Annotation: "TestHasSidecarAlready"}, map[string]string{})
	assert.True(t, Needed(pod))
	jaeger := v1.NewJaeger("TestHasSidecarAlready")
	pod = Sidecar(jaeger, pod)
	assert.False(t, Needed(pod))
}

func TestSelectSingleJaegerPod(t *testing.T) {
	pod := pod(map[string]string{Annotation: "true"}, map[string]string{})
	jaegerPods := &v1.JaegerList{
		Items: []v1.Jaeger{
			v1.Jaeger{
				ObjectMeta: metav1.ObjectMeta{
					Name: "the-only-jaeger-instance-available",
				},
			},
		},
	}

	jaeger := Select(pod, jaegerPods)
	assert.NotNil(t, jaeger)
	assert.Equal(t, "the-only-jaeger-instance-available", jaeger.Name)
}

func TestCannotSelectFromMultipleJaegerPods(t *testing.T) {
	pod := pod(map[string]string{Annotation: "true"}, map[string]string{})
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

	jaeger := Select(pod, jaegerPods)
	assert.Nil(t, jaeger)
}

func TestNoAvailableJaegerPods(t *testing.T) {
	pod := pod(map[string]string{Annotation: "true"}, map[string]string{})
	jaeger := Select(pod, &v1.JaegerList{})
	assert.Nil(t, jaeger)
}

func TestSelectBasedOnName(t *testing.T) {
	pod := pod(map[string]string{Annotation: "the-second-jaeger-instance-available"}, map[string]string{})

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

	jaeger := Select(pod, jaegerPods)
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

	pod := pod(map[string]string{Annotation: jaeger.Name}, map[string]string{})
	pod = Sidecar(jaeger, pod)

	assert.Len(t, pod.Spec.Containers, 2)
	assert.Len(t, pod.Spec.Containers[1].Args, 5)
	assert.True(t, strings.HasPrefix(pod.Spec.Containers[1].Args[0], "--a-option"))
	assert.True(t, strings.HasPrefix(pod.Spec.Containers[1].Args[1], "--b-option"))
	assert.True(t, strings.HasPrefix(pod.Spec.Containers[1].Args[2], "--c-option"))
	assert.True(t, strings.HasPrefix(pod.Spec.Containers[1].Args[3], "--reporter.grpc.host-port"))
	assert.True(t, strings.HasPrefix(pod.Spec.Containers[1].Args[4], "--reporter.type"))
}

func TestSidecarOverrideReporter(t *testing.T) {
	jaeger := v1.NewJaeger("TestQueryOrderOfArguments")
	jaeger.Spec.Agent.Options = v1.NewOptions(map[string]interface{}{
		"reporter.type":             "thrift",
		"reporter.thrift.host-port": "collector:14267",
	})

	pod := pod(map[string]string{Annotation: jaeger.Name}, map[string]string{})
	pod = Sidecar(jaeger, pod)

	assert.Len(t, pod.Spec.Containers, 2)
	assert.Len(t, pod.Spec.Containers[1].Args, 2)
	assert.Equal(t, "--reporter.thrift.host-port=collector:14267", pod.Spec.Containers[1].Args[0])
	assert.Equal(t, "--reporter.type=thrift", pod.Spec.Containers[1].Args[1])
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

	pod := pod(map[string]string{Annotation: jaeger.Name}, map[string]string{})
	pod = Sidecar(jaeger, pod)

	assert.Len(t, pod.Spec.Containers, 2, "Expected 2 containers")
	assert.Equal(t, "jaeger-agent", pod.Spec.Containers[1].Name)
	assert.Equal(t, *resource.NewQuantity(2048, resource.BinarySI), pod.Spec.Containers[1].Resources.Limits[corev1.ResourceLimitsCPU])
	assert.Equal(t, *resource.NewQuantity(2048, resource.BinarySI), pod.Spec.Containers[1].Resources.Requests[corev1.ResourceRequestsCPU])
	assert.Equal(t, *resource.NewQuantity(123, resource.DecimalSI), pod.Spec.Containers[1].Resources.Limits[corev1.ResourceLimitsMemory])
	assert.Equal(t, *resource.NewQuantity(123, resource.DecimalSI), pod.Spec.Containers[1].Resources.Requests[corev1.ResourceRequestsMemory])
	assert.Equal(t, *resource.NewQuantity(512, resource.DecimalSI), pod.Spec.Containers[1].Resources.Limits[corev1.ResourceLimitsEphemeralStorage])
	assert.Equal(t, *resource.NewQuantity(512, resource.DecimalSI), pod.Spec.Containers[1].Resources.Requests[corev1.ResourceRequestsEphemeralStorage])
}

func TestProcessInjectingSingleJaegerInPod(t *testing.T) {
	jaeger := v1.NewJaeger("TestProcessInjectingSingleJaegerInPod")
	jaeger.Spec.Agent.Image = "the-agent-image"

	objs := []runtime.Object{
		jaeger,
	}

	s := scheme.Scheme
	v1.SchemeBuilder.AddToScheme(s)

	cl := fake.NewFakeClient(objs...)

	incomingPod, err := json.Marshal(pod(map[string]string{Annotation: "true"}, map[string]string{}))
	assert.NoError(t, err)

	ar := &admission.AdmissionReview{
		Request: &admission.AdmissionRequest{
			Kind: metav1.GroupVersionKind{Group: "", Version: "v1", Kind: "Pod"},
			Object: runtime.RawExtension{
				Raw: []byte(incomingPod),
			},
		},
	}

	resp, err := Process(ar, cl)
	assert.NoError(t, err)
	assert.NotNil(t, resp)
	assert.True(t, resp.Allowed)
	assert.Contains(t, string(resp.Patch), `"op":"add"`)
	assert.Contains(t, string(resp.Patch), `"path":"/spec/containers/-"`)
	assert.Contains(t, string(resp.Patch), `"name":"jaeger-agent"`)
	assert.Contains(t, string(resp.Patch), `"image":"the-agent-image"`)
}

func TestProcessNoSuitableJaeger(t *testing.T) {
	s := scheme.Scheme
	v1.SchemeBuilder.AddToScheme(s)

	objs := []runtime.Object{v1.NewJaeger("TestProcessNoSuitableJaeger")}
	cl := fake.NewFakeClient(objs...)

	incomingPod, err := json.Marshal(pod(map[string]string{Annotation: "non-existing-jaeger"}, map[string]string{}))
	assert.NoError(t, err)

	ar := &admission.AdmissionReview{
		Request: &admission.AdmissionRequest{
			Kind: metav1.GroupVersionKind{Group: "", Version: "v1", Kind: "Pod"},
			Object: runtime.RawExtension{
				Raw: []byte(incomingPod),
			},
		},
	}

	resp, err := Process(ar, cl)
	assert.NoError(t, err)
	assert.NotNil(t, resp)
	assert.True(t, resp.Allowed)
	assert.Empty(t, string(resp.Patch))
}

func TestProcessInvalidIncomingObject(t *testing.T) {
	incoming, err := json.Marshal(corev1.Secret{})
	assert.NoError(t, err)

	ar := &admission.AdmissionReview{
		Request: &admission.AdmissionRequest{
			Kind: metav1.GroupVersionKind{Group: "", Version: "v1", Kind: "Secret"},
			Object: runtime.RawExtension{
				Raw: []byte(incoming),
			},
		},
	}

	resp, err := Process(ar, fake.NewFakeClient())
	assert.NoError(t, err)
	assert.NotNil(t, resp)
	assert.True(t, resp.Allowed)
}

func TestProcessDeploymentWithAnnotation(t *testing.T) {
	incoming, err := json.Marshal(dep(map[string]string{Annotation: "some-jaeger"}, map[string]string{}))
	assert.NoError(t, err)

	ar := &admission.AdmissionReview{
		Request: &admission.AdmissionRequest{
			Kind: metav1.GroupVersionKind{Group: "apps", Version: "v1", Kind: "Deployment"},
			Object: runtime.RawExtension{
				Raw: []byte(incoming),
			},
		},
	}

	resp, err := Process(ar, fake.NewFakeClient())
	assert.NoError(t, err)
	assert.NotNil(t, resp)
	assert.True(t, resp.Allowed)
	assert.Contains(t, string(resp.Patch), `"op":"add"`)
	assert.Contains(t, string(resp.Patch), `"path":"/spec/template/metadata/annotations"`)
	assert.Contains(t, string(resp.Patch), `"sidecar.jaegertracing.io/inject":"some-jaeger"`)
}

func TestProcessDeploymentAndPodWithAnnotation(t *testing.T) {
	dep := dep(map[string]string{Annotation: "some-jaeger"}, map[string]string{})
	dep.Spec.Template.Annotations[Annotation] = "some-other-jaeger"

	incoming, err := json.Marshal(dep)
	assert.NoError(t, err)

	ar := &admission.AdmissionReview{
		Request: &admission.AdmissionRequest{
			Kind: metav1.GroupVersionKind{Group: "apps", Version: "v1", Kind: "Deployment"},
			Object: runtime.RawExtension{
				Raw: []byte(incoming),
			},
		},
	}

	resp, err := Process(ar, fake.NewFakeClient())
	assert.NoError(t, err)
	assert.NotNil(t, resp)
	assert.True(t, resp.Allowed)
	assert.Nil(t, resp.Patch)
}

func TestProcessDeploymentWithAnnotationLegacy(t *testing.T) {
	incoming, err := json.Marshal(dep(map[string]string{AnnotationLegacy: "some-jaeger"}, map[string]string{}))
	assert.NoError(t, err)

	ar := &admission.AdmissionReview{
		Request: &admission.AdmissionRequest{
			Kind: metav1.GroupVersionKind{Group: "apps", Version: "v1", Kind: "Deployment"},
			Object: runtime.RawExtension{
				Raw: []byte(incoming),
			},
		},
	}

	resp, err := Process(ar, fake.NewFakeClient())
	assert.NoError(t, err)
	assert.NotNil(t, resp)
	assert.True(t, resp.Allowed)
	assert.Contains(t, string(resp.Patch), `"op":"add"`)
	assert.Contains(t, string(resp.Patch), `"path":"/spec/template/metadata/annotations"`)

	// we should copy the value of the legacy annotation to the pod using the new annotation name
	assert.Contains(t, string(resp.Patch), `"sidecar.jaegertracing.io/inject":"some-jaeger"`)
}

func TestProcessDeploymentWithNoAnnotation(t *testing.T) {
	incoming, err := json.Marshal(dep(map[string]string{}, map[string]string{}))
	assert.NoError(t, err)

	ar := &admission.AdmissionReview{
		Request: &admission.AdmissionRequest{
			Kind: metav1.GroupVersionKind{Group: "apps", Version: "v1", Kind: "Deployment"},
			Object: runtime.RawExtension{
				Raw: []byte(incoming),
			},
		},
	}

	resp, err := Process(ar, fake.NewFakeClient())
	assert.NoError(t, err)
	assert.NotNil(t, resp)
	assert.True(t, resp.Allowed)
	assert.Nil(t, resp.Patch)
}

func TestProcessInvalidRawObject(t *testing.T) {
	ar := &admission.AdmissionReview{
		Request: &admission.AdmissionRequest{
			Kind: metav1.GroupVersionKind{Group: "", Version: "v1", Kind: "Pod"},
			Object: runtime.RawExtension{
				Raw: []byte("{a1"),
			},
		},
	}

	resp, err := Process(ar, fake.NewFakeClient())
	assert.Error(t, err)
	assert.NotEmpty(t, resp.Result.Message)
}

func pod(annotations map[string]string, labels map[string]string) corev1.Pod {
	return corev1.Pod{
		TypeMeta: metav1.TypeMeta{
			Kind:       "Pod",
			APIVersion: "v1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Annotations: annotations,
			Labels:      labels,
		},
		Spec: corev1.PodSpec{
			Containers: []corev1.Container{
				corev1.Container{},
			},
		},
	}
}

func dep(annotations map[string]string, labels map[string]string) *appsv1.Deployment {
	return &appsv1.Deployment{
		TypeMeta: metav1.TypeMeta{
			Kind:       "Deployment",
			APIVersion: "extensions/v1beta1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Annotations: annotations,
			Labels:      labels,
		},
		Spec: appsv1.DeploymentSpec{
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Annotations: map[string]string{},
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
