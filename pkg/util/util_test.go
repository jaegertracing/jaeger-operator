package util

import (
	"sort"
	"testing"

	"github.com/jaegertracing/jaeger-operator/pkg/version"

	"github.com/spf13/viper"
	"github.com/stretchr/testify/assert"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"

	v1 "github.com/jaegertracing/jaeger-operator/apis/v1"
)

func TestRemoveDuplicatedVolumes(t *testing.T) {
	volumes := []corev1.Volume{{
		Name:         "volume1",
		VolumeSource: corev1.VolumeSource{HostPath: &corev1.HostPathVolumeSource{Path: "/data1"}},
	}, {
		Name:         "volume2",
		VolumeSource: corev1.VolumeSource{HostPath: &corev1.HostPathVolumeSource{Path: "/data2"}},
	}, {
		Name:         "volume1",
		VolumeSource: corev1.VolumeSource{HostPath: &corev1.HostPathVolumeSource{Path: "/data3"}},
	}}

	assert.Len(t, RemoveDuplicatedVolumes(volumes), 2)
	assert.Equal(t, "volume1", volumes[0].Name)
	assert.Equal(t, "/data1", volumes[0].VolumeSource.HostPath.Path)
	assert.Equal(t, "volume2", volumes[1].Name)
}

func TestRemoveDuplicatedVolumeMounts(t *testing.T) {
	volumeMounts := []corev1.VolumeMount{{
		Name:     "data1",
		ReadOnly: false,
	}, {
		Name:     "data2",
		ReadOnly: false,
	}, {
		Name:     "data1",
		ReadOnly: true,
	}}

	assert.Len(t, RemoveDuplicatedVolumeMounts(volumeMounts), 2)
	assert.Equal(t, "data1", volumeMounts[0].Name)
	assert.False(t, volumeMounts[0].ReadOnly)
	assert.Equal(t, "data2", volumeMounts[1].Name)
}

func TestRemoveDuplicatedImagePullSecrets(t *testing.T) {
	imagePullSecrets := []corev1.LocalObjectReference{{
		Name: "secret1",
	}, {
		Name: "secret2",
	}, {
		Name: "secret1",
	}}

	assert.Len(t, RemoveDuplicatedImagePullSecrets(imagePullSecrets), 2)
	assert.Equal(t, "secret1", imagePullSecrets[0].Name)
	assert.Equal(t, "secret2", imagePullSecrets[1].Name)
}

func TestMergeImagePullSecrets(t *testing.T) {
	emptySpec := v1.JaegerCommonSpec{}
	generalSpec := v1.JaegerCommonSpec{
		ImagePullSecrets: []corev1.LocalObjectReference{
			{
				Name: "abc",
			},
		},
	}
	specificSpec := v1.JaegerCommonSpec{
		ImagePullSecrets: []corev1.LocalObjectReference{
			{
				Name: "abc",
			},
			{
				Name: "def",
			},
			{
				Name: "xyz",
			},
		},
	}
	anotherSpec := v1.JaegerCommonSpec{
		ImagePullSecrets: []corev1.LocalObjectReference{
			{
				Name: "hij",
			},
			{
				Name: "xyz",
			},
		},
	}

	merged := Merge([]v1.JaegerCommonSpec{specificSpec, generalSpec, emptySpec, anotherSpec})

	assert.Len(t, merged.ImagePullSecrets, 4)
	assert.Equal(t, "abc", merged.ImagePullSecrets[0].Name)
	assert.Equal(t, "def", merged.ImagePullSecrets[1].Name)
	assert.Equal(t, "xyz", merged.ImagePullSecrets[2].Name)
	assert.Equal(t, "hij", merged.ImagePullSecrets[3].Name)
}

func TestMergeImagePullPolicy(t *testing.T) {
	emptySpec := v1.JaegerCommonSpec{}
	generalSpec := v1.JaegerCommonSpec{
		ImagePullPolicy: corev1.PullPolicy("Never"),
	}
	specificSpec := v1.JaegerCommonSpec{
		ImagePullPolicy: corev1.PullPolicy("Always"),
	}
	anotherSpec := v1.JaegerCommonSpec{
		ImagePullPolicy: corev1.PullPolicy("IfNotPresent"),
	}

	merged := Merge([]v1.JaegerCommonSpec{specificSpec, generalSpec, emptySpec, anotherSpec})

	assert.Equal(t, corev1.PullPolicy("Always"), merged.ImagePullPolicy)
}

func TestMergeAnnotations(t *testing.T) {
	generalSpec := v1.JaegerCommonSpec{
		Annotations: map[string]string{
			"name":  "operator",
			"hello": "jaeger",
		},
	}
	specificSpec := v1.JaegerCommonSpec{
		Annotations: map[string]string{
			"hello":                "world", // Override general annotation
			"prometheus.io/scrape": "false",
		},
	}

	merged := Merge([]v1.JaegerCommonSpec{specificSpec, generalSpec})

	assert.Equal(t, "operator", merged.Annotations["name"])
	assert.Equal(t, "world", merged.Annotations["hello"])
	assert.Equal(t, "false", merged.Annotations["prometheus.io/scrape"])
}

func TestMergeLabels(t *testing.T) {
	generalSpec := v1.JaegerCommonSpec{
		Labels: map[string]string{
			"name":  "operator",
			"hello": "jaeger",
		},
	}
	specificSpec := v1.JaegerCommonSpec{
		Labels: map[string]string{
			"hello":   "world", // Override general annotation
			"another": "false",
		},
	}

	merged := Merge([]v1.JaegerCommonSpec{specificSpec, generalSpec})

	assert.Equal(t, "operator", merged.Labels["name"])
	assert.Equal(t, "world", merged.Labels["hello"])
	assert.Equal(t, "false", merged.Labels["another"])
}

func TestMergeMountVolumes(t *testing.T) {
	generalSpec := v1.JaegerCommonSpec{
		VolumeMounts: []corev1.VolumeMount{{
			Name:     "data1",
			ReadOnly: true,
		}},
	}
	specificSpec := v1.JaegerCommonSpec{
		VolumeMounts: []corev1.VolumeMount{{
			Name:     "data1",
			ReadOnly: false,
		}, {
			Name:     "data2",
			ReadOnly: false,
		}},
	}

	merged := Merge([]v1.JaegerCommonSpec{specificSpec, generalSpec})

	assert.Equal(t, "data1", merged.VolumeMounts[0].Name)
	assert.False(t, merged.VolumeMounts[0].ReadOnly)
	assert.Equal(t, "data2", merged.VolumeMounts[1].Name)
}

func TestMergeVolumes(t *testing.T) {
	generalSpec := v1.JaegerCommonSpec{
		Volumes: []corev1.Volume{{
			Name:         "volume1",
			VolumeSource: corev1.VolumeSource{HostPath: &corev1.HostPathVolumeSource{Path: "/data3"}},
		}},
	}
	specificSpec := v1.JaegerCommonSpec{
		Volumes: []corev1.Volume{{
			Name:         "volume1",
			VolumeSource: corev1.VolumeSource{HostPath: &corev1.HostPathVolumeSource{Path: "/data1"}},
		}, {
			Name:         "volume2",
			VolumeSource: corev1.VolumeSource{HostPath: &corev1.HostPathVolumeSource{Path: "/data2"}},
		}},
	}

	merged := Merge([]v1.JaegerCommonSpec{specificSpec, generalSpec})

	assert.Equal(t, "volume1", merged.Volumes[0].Name)
	assert.Equal(t, "/data1", merged.Volumes[0].VolumeSource.HostPath.Path)
	assert.Equal(t, "volume2", merged.Volumes[1].Name)
}

func TestMergeResourceLimits(t *testing.T) {
	generalSpec := v1.JaegerCommonSpec{
		Resources: corev1.ResourceRequirements{
			Limits: corev1.ResourceList{
				corev1.ResourceLimitsCPU:              *resource.NewQuantity(1024, resource.BinarySI),
				corev1.ResourceLimitsEphemeralStorage: *resource.NewQuantity(123, resource.DecimalSI),
			},
		},
	}
	specificSpec := v1.JaegerCommonSpec{
		Resources: corev1.ResourceRequirements{
			Limits: corev1.ResourceList{
				corev1.ResourceLimitsCPU:    *resource.NewQuantity(2048, resource.BinarySI),
				corev1.ResourceLimitsMemory: *resource.NewQuantity(1024, resource.BinarySI),
			},
		},
	}

	merged := Merge([]v1.JaegerCommonSpec{specificSpec, generalSpec})

	assert.Equal(t, *resource.NewQuantity(2048, resource.BinarySI), merged.Resources.Limits[corev1.ResourceLimitsCPU])
	assert.Equal(t, *resource.NewQuantity(1024, resource.BinarySI), merged.Resources.Limits[corev1.ResourceLimitsMemory])
	assert.Equal(t, *resource.NewQuantity(123, resource.DecimalSI), merged.Resources.Limits[corev1.ResourceLimitsEphemeralStorage])
}

func TestMergeResourceRequests(t *testing.T) {
	generalSpec := v1.JaegerCommonSpec{
		Resources: corev1.ResourceRequirements{
			Requests: corev1.ResourceList{
				corev1.ResourceRequestsCPU:              *resource.NewQuantity(1024, resource.BinarySI),
				corev1.ResourceRequestsEphemeralStorage: *resource.NewQuantity(123, resource.DecimalSI),
			},
		},
	}
	specificSpec := v1.JaegerCommonSpec{
		Resources: corev1.ResourceRequirements{
			Requests: corev1.ResourceList{
				corev1.ResourceRequestsCPU:    *resource.NewQuantity(2048, resource.BinarySI),
				corev1.ResourceRequestsMemory: *resource.NewQuantity(1024, resource.BinarySI),
			},
		},
	}

	merged := Merge([]v1.JaegerCommonSpec{specificSpec, generalSpec})

	assert.Equal(t, *resource.NewQuantity(2048, resource.BinarySI), merged.Resources.Requests[corev1.ResourceRequestsCPU])
	assert.Equal(t, *resource.NewQuantity(1024, resource.BinarySI), merged.Resources.Requests[corev1.ResourceRequestsMemory])
	assert.Equal(t, *resource.NewQuantity(123, resource.DecimalSI), merged.Resources.Requests[corev1.ResourceRequestsEphemeralStorage])
}

func TestAffinityDefault(t *testing.T) {
	generalSpec := v1.JaegerCommonSpec{}
	specificSpec := v1.JaegerCommonSpec{}

	merged := Merge([]v1.JaegerCommonSpec{specificSpec, generalSpec})

	assert.Nil(t, merged.Affinity)
}

func TestAffinityOverride(t *testing.T) {
	generalSpec := v1.JaegerCommonSpec{
		Affinity: &corev1.Affinity{
			NodeAffinity: &corev1.NodeAffinity{},
		},
	}
	specificSpec := v1.JaegerCommonSpec{
		Affinity: &corev1.Affinity{
			PodAffinity: &corev1.PodAffinity{},
		},
	}

	merged := Merge([]v1.JaegerCommonSpec{specificSpec, generalSpec})

	assert.NotNil(t, merged.Affinity)
	assert.NotNil(t, merged.Affinity.PodAffinity)
	assert.Nil(t, merged.Affinity.NodeAffinity)
}

func TestSecurityContextDefault(t *testing.T) {
	generalSpec := v1.JaegerCommonSpec{}
	specificSpec := v1.JaegerCommonSpec{}

	merged := Merge([]v1.JaegerCommonSpec{specificSpec, generalSpec})

	assert.Nil(t, merged.SecurityContext)
}

func TestSecurityContextOverride(t *testing.T) {
	intVal := int64(1000)
	generalSpec := v1.JaegerCommonSpec{
		SecurityContext: &corev1.PodSecurityContext{
			RunAsUser: &intVal,
		},
	}
	specificSpec := v1.JaegerCommonSpec{
		SecurityContext: &corev1.PodSecurityContext{
			RunAsGroup: &intVal,
		},
	}

	merged := Merge([]v1.JaegerCommonSpec{specificSpec, generalSpec})

	assert.NotNil(t, merged.SecurityContext)
	assert.NotNil(t, merged.SecurityContext.RunAsGroup)
	assert.Nil(t, merged.SecurityContext.RunAsUser)
}

func TestMergeTolerations(t *testing.T) {
	generalSpec := v1.JaegerCommonSpec{
		Tolerations: []corev1.Toleration{{
			Key: "toleration1",
		}},
	}
	specificSpec := v1.JaegerCommonSpec{
		Tolerations: []corev1.Toleration{{
			Key: "toleration1",
		}, {
			Key: "toleration2",
		}},
	}

	merged := Merge([]v1.JaegerCommonSpec{specificSpec, generalSpec})

	// Keys do not need to be unique, so should be aggregation of all tolerations
	// See https://kubernetes.io/docs/concepts/configuration/taint-and-toleration/ for more details
	assert.Len(t, merged.Tolerations, 3)
	assert.Equal(t, "toleration1", merged.Tolerations[0].Key)
	assert.Equal(t, "toleration2", merged.Tolerations[1].Key)
	assert.Equal(t, "toleration1", merged.Tolerations[2].Key)
}

func TestGetEsHostname(t *testing.T) {
	tests := []struct {
		underTest map[string]interface{}
		hostname  string
	}{
		{hostname: ""},
		{underTest: map[string]interface{}{"": ""}, hostname: ""},
		{underTest: map[string]interface{}{"es.server-urls": ""}, hostname: ""},
		{underTest: map[string]interface{}{"es.server-urls": "goo:tar"}, hostname: "goo:tar"},
		{underTest: map[string]interface{}{"es.server-urls": "http://es:9000,https://es2:9200"}, hostname: "http://es:9000"},
	}
	for _, test := range tests {
		assert.Equal(t, test.hostname, GetEsHostname(test.underTest))
	}
}

func TestAsOwner(t *testing.T) {
	j := v1.NewJaeger(types.NamespacedName{Name: "joe"})
	j.Kind = "human"
	j.APIVersion = "homosapiens"
	j.UID = "boom!"
	ow := AsOwner(j)
	trueVar := true
	assert.Equal(t, metav1.OwnerReference{Name: "joe", Kind: "human", APIVersion: "homosapiens", UID: "boom!", Controller: &trueVar}, ow)
}

func TestLabels(t *testing.T) {
	assert.Equal(t, map[string]string{
		"app":                          "jaeger",
		"app.kubernetes.io/name":       "joe",
		"app.kubernetes.io/instance":   "thatone",
		"app.kubernetes.io/component":  "leg",
		"app.kubernetes.io/part-of":    "jaeger",
		"app.kubernetes.io/managed-by": "jaeger-operator",
	}, Labels("joe", "leg", *v1.NewJaeger(types.NamespacedName{Name: "thatone"})))
}

func TestFindItem(t *testing.T) {
	opts := v1.NewOptions(map[string]interface{}{
		"reporter.type":             "thrift",
		"reporter.thrift.host-port": "collector:14267",
	})

	args := opts.ToArgs()

	assert.Equal(t, "--reporter.type=thrift", FindItem("--reporter.type=", args))
	assert.Empty(t, FindItem("--c-option", args))
}

func TestGetPortDefault(t *testing.T) {
	opts := v1.NewOptions(map[string]interface{}{})

	args := opts.ToArgs()

	assert.Equal(t, int32(1234), GetPort("--processor.jaeger-compact.server-host-port=", args, 1234))
}

func TestGetPortSpecified(t *testing.T) {
	opts := v1.NewOptions(map[string]interface{}{
		"processor.jaeger-compact.server-host-port": ":6831",
	})

	args := opts.ToArgs()

	assert.Equal(t, int32(6831), GetPort("--processor.jaeger-compact.server-host-port=", args, 1234))
}

func TestGetAdminPort(t *testing.T) {
	tests := map[string]struct {
		opts         v1.Options
		defaultPort  int32
		expectedPort int32
	}{
		"Use default port when no admin port flag provided": {
			opts:         v1.NewOptions(map[string]interface{}{}),
			defaultPort:  1234,
			expectedPort: 1234,
		},
		"Use deprecated flag when new flag not provided and deprecated flag provided": {
			opts: v1.NewOptions(map[string]interface{}{
				"admin-http-port": ":1111",
			}),
			defaultPort:  1234,
			expectedPort: 1111,
		},
		"Use new flag when provided": {
			opts: v1.NewOptions(map[string]interface{}{
				"admin-http-port":      ":1111",
				"admin.http.host-port": ":2222",
			}),
			defaultPort:  1234,
			expectedPort: 2222,
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			args := test.opts.ToArgs()
			assert.Equal(t, test.expectedPort, GetAdminPort(args, test.defaultPort))
		})
	}
}

func TestInitObjectMeta(t *testing.T) {
	tests := map[string]struct {
		obj metav1.Object
		exp metav1.Object
	}{
		"A object without initialized labels shouldn't have a nil map after initialization.": {
			obj: &corev1.Pod{
				ObjectMeta: metav1.ObjectMeta{
					Name:        "test",
					Annotations: map[string]string{"gopher": "jaeger"},
				},
			},
			exp: &corev1.Pod{
				ObjectMeta: metav1.ObjectMeta{
					Name:        "test",
					Annotations: map[string]string{"gopher": "jaeger"},
					Labels:      map[string]string{},
				},
			},
		},

		"A object without initialized annotations shouldn't have a nil map after initialization.": {
			obj: &corev1.Pod{
				ObjectMeta: metav1.ObjectMeta{
					Name:   "test",
					Labels: map[string]string{"gopher": "jaeger"},
				},
			},
			exp: &corev1.Pod{
				ObjectMeta: metav1.ObjectMeta{
					Name:        "test",
					Labels:      map[string]string{"gopher": "jaeger"},
					Annotations: map[string]string{},
				},
			},
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			InitObjectMeta(test.obj)
			assert.Equal(t, test.exp, test.obj)
		})
	}
}

func TestImageNameSupplied(t *testing.T) {
	viper.Set("test-image", "org/custom-image")
	defer viper.Reset()

	assert.Equal(t, "org/actual-image:1.2.3", ImageName("org/actual-image:1.2.3", "test-image"))
}

func TestImageNameParamNoTag(t *testing.T) {
	viper.Set("test-image", "org/custom-image")
	defer viper.Reset()

	assert.Equal(t, "org/custom-image:"+version.Get().Jaeger, ImageName("", "test-image"))
}

func TestImageNameParamWithTag(t *testing.T) {
	viper.Set("test-image", "org/custom-image:1.2.3")
	defer viper.Reset()

	assert.Equal(t, "org/custom-image:1.2.3", ImageName("", "test-image"))
}

func TestImageNameParamWithDigest(t *testing.T) {
	viper.Set("test-image", "org/custom-image@sha256:2a7ef4373262fa5fa3b3eaac86015650f8f3eee65d6e2674df931657873e318e")
	defer viper.Reset()

	assert.Equal(t, "org/custom-image@sha256:2a7ef4373262fa5fa3b3eaac86015650f8f3eee65d6e2674df931657873e318e", ImageName("", "test-image"))
}

func TestImageNameParamDefaultNoTag(t *testing.T) {
	viper.SetDefault("test-image", "org/default-image")
	defer viper.Reset()

	assert.Equal(t, "org/default-image:"+version.Get().Jaeger, ImageName("", "test-image"))
}

func TestImageNameParamDefaultWithTag(t *testing.T) {
	viper.SetDefault("test-image", "org/default-image:1.2.3")
	defer viper.Reset()

	assert.Equal(t, "org/default-image:1.2.3", ImageName("", "test-image"))
}

func TestRemoveEmptyVars(t *testing.T) {
	tests := []struct {
		underTest []corev1.EnvVar
		expected  []corev1.EnvVar
	}{
		{},
		{
			underTest: []corev1.EnvVar{{Name: "foo", Value: "bar"}, {Name: "foo3"}, {Name: "foo2", ValueFrom: &corev1.EnvVarSource{}}},
			expected:  []corev1.EnvVar{{Name: "foo", Value: "bar"}, {Name: "foo2", ValueFrom: &corev1.EnvVarSource{}}},
		},
		{underTest: []corev1.EnvVar{{Name: "foo"}}},
	}
	for _, test := range tests {
		exp := RemoveEmptyVars(test.underTest)
		assert.Equal(t, test.expected, exp)
	}
}

func TestCreateFromSecret(t *testing.T) {
	tests := []struct {
		secret   string
		expected []corev1.EnvFromSource
	}{
		{},
		{
			secret: "foobar", expected: []corev1.EnvFromSource{
				{SecretRef: &corev1.SecretEnvSource{LocalObjectReference: corev1.LocalObjectReference{Name: "foobar"}}},
			},
		},
	}
	for _, test := range tests {
		exp := CreateEnvsFromSecret(test.secret)
		assert.Equal(t, test.expected, exp)
	}
}

func TestReplaceArgument(t *testing.T) {
	newValue := "SECRET2"
	prefix := "--cookie-secret="

	tests := []struct {
		input    []string
		expected []string
		count    int
	}{
		{
			input: []string{
				"--cookie-secret=SECRET1",
				"--https-address=:8443",
				"--provider=openshift",
			},
			expected: []string{
				"--cookie-secret=" + newValue,
				"--https-address=:8443",
				"--provider=openshift",
			},
			count: 1,
		},
		{
			input: []string{
				"--cookie-secret=SECRET1",
				"--cookie-secret=SECRET3",
				"--https-address=:8443",
				"--provider=openshift",
			},
			expected: []string{
				"--cookie-secret=" + newValue,
				"--cookie-secret=" + newValue,
				"--https-address=:8443",
				"--provider=openshift",
			},
			count: 2,
		},
		{
			input: []string{
				"--https-address=:8443",
				"--provider=openshift",
			},
			expected: []string{
				"--https-address=:8443",
				"--provider=openshift",
			},
			count: 0,
		},
	}

	for _, test := range tests {
		counter := ReplaceArgument(prefix, prefix+newValue, test.input)
		assert.Equal(t, test.count, counter)
		assert.Equal(t, test.expected, test.input)
	}
}

func TestArgs(t *testing.T) {
	// prepare
	jaeger := v1.NewJaeger(types.NamespacedName{Name: "TestArgs"})
	jaeger.Spec.Storage.Options = v1.NewOptions(map[string]interface{}{"memory.max-traces": 10000})
	jaeger.Spec.AllInOne.Options = v1.NewOptions(map[string]interface{}{"collector.http-port": 14268})

	// test
	args := AllArgs(jaeger.Spec.Storage.Options, jaeger.Spec.AllInOne.Options)

	// verify
	sort.Strings(args)
	assert.Equal(t, "--collector.http-port=14268", args[0])
	assert.Equal(t, "--memory.max-traces=10000", args[1])
}

func TestFindEnvVars(t *testing.T) {
	myEnvVar := corev1.EnvVar{
		Name:  "my_env_var",
		Value: "v1",
	}

	envVars := []corev1.EnvVar{
		myEnvVar,
		{
			Name:  "other_env",
			Value: "v2",
		},
	}

	tests := []struct {
		name     string
		envName  string
		expected *corev1.EnvVar
	}{
		{
			name:     "env var found",
			envName:  "my_env_var",
			expected: &myEnvVar,
		},
		{
			name:     "env var found",
			envName:  "no_exist_env",
			expected: nil,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			result := FindEnvVar(envVars, tc.envName)
			assert.Equal(t, tc.expected, result)
		})
	}
}

func TestIsOTLPEnable(t *testing.T) {
	tests := []struct {
		name     string
		options  v1.Options
		expected bool
	}{
		{
			name:     "explicit set to true",
			options:  v1.NewOptions(map[string]interface{}{"collector.otlp.enabled": true}),
			expected: true,
		},
		{
			name:     "explicit set to false",
			options:  v1.NewOptions(map[string]interface{}{"collector.otlp.enabled": false}),
			expected: false,
		},
		{
			name:     "no present in options",
			options:  v1.NewOptions(map[string]interface{}{}),
			expected: true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			enable := IsOTLPEnable(AllArgs(tc.options))
			assert.Equal(t, tc.expected, enable)
		})
	}
}

func TestIsOTLPExplcitSet(t *testing.T) {
	tests := []struct {
		name     string
		options  v1.Options
		expected bool
	}{
		{
			name:     "explicit set to true",
			options:  v1.NewOptions(map[string]interface{}{"collector.otlp.enabled": true}),
			expected: true,
		},
		{
			name:     "explicit set to false",
			options:  v1.NewOptions(map[string]interface{}{"collector.otlp.enabled": false}),
			expected: true,
		},
		{
			name:     "no present in options",
			options:  v1.NewOptions(map[string]interface{}{}),
			expected: false,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			enable := IsOTLPExplcitSet(AllArgs(tc.options))
			assert.Equal(t, tc.expected, enable)
		})
	}
}
