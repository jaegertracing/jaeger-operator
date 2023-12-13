package cronjob

import (
	"testing"

	batchv1 "k8s.io/api/batch/v1"
	batchv1beta1 "k8s.io/api/batch/v1beta1"

	"github.com/spf13/viper"
	"github.com/stretchr/testify/assert"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	"k8s.io/apimachinery/pkg/types"

	v1 "github.com/jaegertracing/jaeger-operator/apis/v1"
)

func init() {
	// Always test with v1.  It is available at compile time and is exactly the same as v1beta1
	viper.SetDefault(v1.FlagCronJobsVersion, v1.FlagCronJobsVersionBatchV1)
}

func TestStorageEnvs(t *testing.T) {
	trueVar := true
	falseVar := false
	tests := []struct {
		storage  v1.JaegerStorageSpec
		expected []corev1.EnvVar
	}{
		{storage: v1.JaegerStorageSpec{Type: "foo"}},
		{
			storage: v1.JaegerStorageSpec{
				Type: v1.JaegerCassandraStorage,
				Options: v1.NewOptions(map[string]interface{}{
					"cassandra.servers": "lol:hol", "cassandra.keyspace": "haha",
					"cassandra.username": "jdoe", "cassandra.password": "none",
				}),
			},
			expected: []corev1.EnvVar{
				{Name: "CASSANDRA_CONTACT_POINTS", Value: "lol:hol"},
				{Name: "CASSANDRA_KEYSPACE", Value: "haha"},
				{Name: "CASSANDRA_USERNAME", Value: "jdoe"},
				{Name: "CASSANDRA_PASSWORD", Value: "none"},
				{Name: "CASSANDRA_USE_SSL", Value: ""},
				{Name: "CASSANDRA_LOCAL_DC", Value: ""},
				{Name: "CASSANDRA_CLIENT_AUTH_ENABLED", Value: "false"},
			},
		},
		{
			storage: v1.JaegerStorageSpec{
				Type: v1.JaegerCassandraStorage,
				Options: v1.NewOptions(map[string]interface{}{
					"cassandra.servers": "lol:hol", "cassandra.keyspace": "haha",
					"cassandra.username": "jdoe", "cassandra.password": "none", "cassandra.tls": "ofcourse!", "cassandra.local-dc": "no-remote",
				}),
			},
			expected: []corev1.EnvVar{
				{Name: "CASSANDRA_CONTACT_POINTS", Value: "lol:hol"},
				{Name: "CASSANDRA_KEYSPACE", Value: "haha"},
				{Name: "CASSANDRA_USERNAME", Value: "jdoe"},
				{Name: "CASSANDRA_PASSWORD", Value: "none"},
				{Name: "CASSANDRA_USE_SSL", Value: "ofcourse!"},
				{Name: "CASSANDRA_LOCAL_DC", Value: "no-remote"},
				{Name: "CASSANDRA_CLIENT_AUTH_ENABLED", Value: "false"},
			},
		},
		{
			storage: v1.JaegerStorageSpec{
				Type: v1.JaegerESStorage,
				Options: v1.NewOptions(map[string]interface{}{
					"es.server-urls": "lol:hol", "es.index-prefix": "haha",
					"es.index-date-separator": ".", "es.username": "jdoe", "es.password": "none",
					"es.use-aliases": "true",
				}),
			},
			expected: []corev1.EnvVar{
				{Name: "ES_NODES", Value: "lol:hol"},
				{Name: "ES_INDEX_PREFIX", Value: "haha"},
				{Name: "ES_INDEX_DATE_SEPARATOR", Value: "."},
				{Name: "ES_USERNAME", Value: "jdoe"},
				{Name: "ES_PASSWORD", Value: "none"},
				{Name: "ES_TIME_RANGE", Value: ""},
				{Name: "ES_USE_ALIASES", Value: "true"},
			},
		},
		{
			storage: v1.JaegerStorageSpec{
				Type: v1.JaegerESStorage,
				Options: v1.NewOptions(map[string]interface{}{
					"es.server-urls": "lol:hol", "es.index-prefix": "haha",
					"es.index-date-separator": ".", "es.username": "jdoe", "es.password": "none",
				}),
				Dependencies: v1.JaegerDependenciesSpec{ElasticsearchClientNodeOnly: &trueVar, ElasticsearchNodesWanOnly: &falseVar},
			},
			expected: []corev1.EnvVar{
				{Name: "ES_NODES", Value: "lol:hol"},
				{Name: "ES_INDEX_PREFIX", Value: "haha"},
				{Name: "ES_INDEX_DATE_SEPARATOR", Value: "."},
				{Name: "ES_USERNAME", Value: "jdoe"},
				{Name: "ES_PASSWORD", Value: "none"},
				{Name: "ES_TIME_RANGE", Value: ""},
				{Name: "ES_USE_ALIASES", Value: ""},
				{Name: "ES_NODES_WAN_ONLY", Value: "false"},
				{Name: "ES_CLIENT_NODE_ONLY", Value: "true"},
			},
		},
		{
			storage: v1.JaegerStorageSpec{
				Type: v1.JaegerESStorage,
				Options: v1.NewOptions(map[string]interface{}{
					"es.server-urls": "lol:hol", "es.index-prefix": "haha",
					"es.username": "jdoe", "es.password": "none",
					"es.use-aliases": "false",
				}),
				Dependencies: v1.JaegerDependenciesSpec{ElasticsearchTimeRange: "30m"},
			},
			expected: []corev1.EnvVar{
				{Name: "ES_NODES", Value: "lol:hol"},
				{Name: "ES_INDEX_PREFIX", Value: "haha"},
				{Name: "ES_INDEX_DATE_SEPARATOR", Value: ""},
				{Name: "ES_USERNAME", Value: "jdoe"},
				{Name: "ES_PASSWORD", Value: "none"},
				{Name: "ES_TIME_RANGE", Value: "30m"},
				{Name: "ES_USE_ALIASES", Value: "false"},
			},
		},
	}
	for _, test := range tests {
		envVars := getStorageEnvs(test.storage)
		assert.Equal(t, test.expected, envVars)
	}
}

func TestCreate(t *testing.T) {
	assert.NotNil(t, CreateSparkDependencies(&v1.Jaeger{Spec: v1.JaegerSpec{Storage: v1.JaegerStorageSpec{Type: v1.JaegerESStorage}}}))
}

func TestCreateTypeMeta(t *testing.T) {
	testData := []struct {
		Name string
		flag string
	}{
		{Name: "Test batch/v1beta1", flag: v1.FlagCronJobsVersionBatchV1Beta1},
		{Name: "Test batch/v1", flag: v1.FlagCronJobsVersionBatchV1},
	}
	for _, td := range testData {
		if td.flag == v1.FlagCronJobsVersionBatchV1Beta1 {
			viper.SetDefault(v1.FlagCronJobsVersion, v1.FlagCronJobsVersionBatchV1Beta1)
		}
		sd := CreateSparkDependencies(&v1.Jaeger{Spec: v1.JaegerSpec{Storage: v1.JaegerStorageSpec{Type: v1.JaegerESStorage}}})
		assert.NotNil(t, sd)
		switch tt := sd.(type) {
		case *batchv1beta1.CronJob:
			assert.Equal(t, "CronJob", tt.Kind)
			assert.Equal(t, v1.FlagCronJobsVersionBatchV1Beta1, tt.APIVersion)
			viper.SetDefault(v1.FlagCronJobsVersion, v1.FlagCronJobsVersionBatchV1)
		case *batchv1.CronJob:
			assert.Equal(t, "CronJob", tt.Kind)
			assert.Equal(t, v1.FlagCronJobsVersionBatchV1, tt.APIVersion)
		}
	}
}

func TestSparkDependenciesSecrets(t *testing.T) {
	jaeger := v1.NewJaeger(types.NamespacedName{Name: "TestSparkDependenciesSecrets"})
	secret := "mysecret"
	jaeger.Spec.Storage.SecretName = secret

	days := 0
	jaeger.Spec.Storage.EsIndexCleaner.NumberOfDays = &days
	cronJob := CreateSparkDependencies(jaeger).(*batchv1.CronJob)
	assert.Len(t, cronJob.Spec.JobTemplate.Spec.Template.Spec.Containers, 1)
	assert.Len(t, cronJob.Spec.JobTemplate.Spec.Template.Spec.Containers[0].EnvFrom, 1)
	assert.Equal(t, secret, cronJob.Spec.JobTemplate.Spec.Template.Spec.Containers[0].EnvFrom[0].SecretRef.LocalObjectReference.Name)
}

func TestSparkDependencies(t *testing.T) {
	j := &v1.Jaeger{Spec: v1.JaegerSpec{Storage: v1.JaegerStorageSpec{Type: v1.JaegerESStorage}}}
	historyLimits := int32(3)
	j.Spec.Storage.Dependencies.SuccessfulJobsHistoryLimit = &historyLimits
	cjob := CreateSparkDependencies(j).(*batchv1.CronJob)
	assert.Equal(t, j.Namespace, cjob.Namespace)
	assert.Equal(t, historyLimits, *cjob.Spec.SuccessfulJobsHistoryLimit)
}

func TestDependenciesAnnotations(t *testing.T) {
	jaeger := v1.NewJaeger(types.NamespacedName{Name: "TestDependenciesAnnotations"})
	jaeger.Spec.Annotations = map[string]string{
		"name":  "operator",
		"hello": "jaeger",
	}
	jaeger.Spec.Storage.Dependencies.Annotations = map[string]string{
		"hello":                "world", // Override top level annotation
		"prometheus.io/scrape": "false", // Override implicit value
	}

	cjob := CreateSparkDependencies(jaeger).(*batchv1.CronJob)

	assert.Equal(t, "operator", cjob.Spec.JobTemplate.Spec.Template.Annotations["name"])
	assert.Equal(t, "false", cjob.Spec.JobTemplate.Spec.Template.Annotations["sidecar.istio.io/inject"])
	assert.Equal(t, "world", cjob.Spec.JobTemplate.Spec.Template.Annotations["hello"])
	assert.Equal(t, "false", cjob.Spec.JobTemplate.Spec.Template.Annotations["prometheus.io/scrape"])
	assert.Equal(t, "disabled", cjob.Spec.JobTemplate.Spec.Template.Annotations["linkerd.io/inject"])
}

func TestSparkDependenciesBackoffLimit(t *testing.T) {
	jaeger := v1.NewJaeger(types.NamespacedName{Name: "TestSparkDependenciesSecrets"})

	BackoffLimit := int32(3)
	jaeger.Spec.Storage.Dependencies.BackoffLimit = &BackoffLimit

	cronJob := CreateSparkDependencies(jaeger).(*batchv1.CronJob)
	assert.Equal(t, &BackoffLimit, cronJob.Spec.JobTemplate.Spec.BackoffLimit)
}

func TestDependenciesLabels(t *testing.T) {
	jaeger := v1.NewJaeger(types.NamespacedName{Name: "TestDependenciesLabels"})
	jaeger.Spec.Labels = map[string]string{
		"name":  "operator",
		"hello": "jaeger",
	}
	jaeger.Spec.Storage.Dependencies.Labels = map[string]string{
		"hello":   "world", // Override top level label
		"another": "false",
	}

	cjob := CreateSparkDependencies(jaeger).(*batchv1.CronJob)

	assert.Equal(t, "operator", cjob.Spec.JobTemplate.Spec.Template.Labels["name"])
	assert.Equal(t, "world", cjob.Spec.JobTemplate.Spec.Template.Labels["hello"])
	assert.Equal(t, "false", cjob.Spec.JobTemplate.Spec.Template.Labels["another"])

	// Check if the labels of cronjob pod template equal to the labels of cronjob.
	assert.Equal(t, cjob.ObjectMeta.Labels, cjob.Spec.JobTemplate.Spec.Template.ObjectMeta.Labels)
}

func TestSparkDependenciesResources(t *testing.T) {
	parentResources := corev1.ResourceRequirements{
		Limits: corev1.ResourceList{
			corev1.ResourceLimitsCPU:              *resource.NewQuantity(1024, resource.BinarySI),
			corev1.ResourceLimitsEphemeralStorage: *resource.NewQuantity(512, resource.DecimalSI),
		},
		Requests: corev1.ResourceList{
			corev1.ResourceRequestsCPU:              *resource.NewQuantity(1024, resource.BinarySI),
			corev1.ResourceRequestsEphemeralStorage: *resource.NewQuantity(512, resource.DecimalSI),
		},
	}

	dependencyResources := corev1.ResourceRequirements{
		Limits: corev1.ResourceList{
			corev1.ResourceLimitsCPU:              *resource.NewQuantity(2048, resource.BinarySI),
			corev1.ResourceLimitsEphemeralStorage: *resource.NewQuantity(1024, resource.DecimalSI),
		},
		Requests: corev1.ResourceList{
			corev1.ResourceRequestsCPU:              *resource.NewQuantity(2048, resource.BinarySI),
			corev1.ResourceRequestsEphemeralStorage: *resource.NewQuantity(1024, resource.DecimalSI),
		},
	}

	tests := []struct {
		jaeger   *v1.Jaeger
		expected corev1.ResourceRequirements
	}{
		{
			jaeger:   &v1.Jaeger{Spec: v1.JaegerSpec{Storage: v1.JaegerStorageSpec{Type: v1.JaegerESStorage}}},
			expected: corev1.ResourceRequirements{},
		},
		{
			jaeger: &v1.Jaeger{Spec: v1.JaegerSpec{
				Storage: v1.JaegerStorageSpec{Type: v1.JaegerESStorage},
				JaegerCommonSpec: v1.JaegerCommonSpec{
					Resources: parentResources,
				},
			}},
			expected: parentResources,
		},
		{
			jaeger: &v1.Jaeger{Spec: v1.JaegerSpec{
				Storage: v1.JaegerStorageSpec{
					Type: v1.JaegerESStorage,
					Dependencies: v1.JaegerDependenciesSpec{
						JaegerCommonSpec: v1.JaegerCommonSpec{
							Resources: dependencyResources,
						},
					},
				},
				JaegerCommonSpec: v1.JaegerCommonSpec{
					Resources: parentResources,
				},
			}},
			expected: dependencyResources,
		},
	}
	for _, test := range tests {
		cjob := CreateSparkDependencies(test.jaeger).(*batchv1.CronJob)
		assert.Equal(t, test.expected, cjob.Spec.JobTemplate.Spec.Template.Spec.Containers[0].Resources)

	}
}

func TestDefaultSparkDependenciesImage(t *testing.T) {
	viper.SetDefault("jaeger-spark-dependencies-image", "ghcr.io/jaegertracing/spark-dependencies/spark-dependencies")

	jaeger := v1.NewJaeger(types.NamespacedName{Name: "TestDefaultSparkDependenciesImage"})

	cjob := CreateSparkDependencies(jaeger).(*batchv1.CronJob)
	assert.Empty(t, jaeger.Spec.Storage.Dependencies.Image)
	assert.Equal(t, "ghcr.io/jaegertracing/spark-dependencies/spark-dependencies", cjob.Spec.JobTemplate.Spec.Template.Spec.Containers[0].Image)
}

func TestCustomSparkDependenciesImage(t *testing.T) {
	viper.Set("jaeger-spark-dependencies-image", "org/custom-spark-dependencies-image")
	defer viper.Reset()

	jaeger := v1.NewJaeger(types.NamespacedName{Name: "TestDefaultSparkDependenciesImage"})

	cjob := CreateSparkDependencies(jaeger).(*batchv1.CronJob)
	assert.Empty(t, jaeger.Spec.Storage.Dependencies.Image)
	assert.Equal(t, "org/custom-spark-dependencies-image", cjob.Spec.JobTemplate.Spec.Template.Spec.Containers[0].Image)
}

func TestDependenciesVolumes(t *testing.T) {
	testVolumeName := "testDependenciesVolume"
	testConfigMapName := "dvConfigMap"
	jaeger := v1.NewJaeger(types.NamespacedName{Name: "TestDependenciesVolumes"})
	testVolume := corev1.Volume{
		Name: testVolumeName,
		VolumeSource: corev1.VolumeSource{
			ConfigMap: &corev1.ConfigMapVolumeSource{
				LocalObjectReference: corev1.LocalObjectReference{Name: testConfigMapName},
			},
		},
	}
	testVolumes := []corev1.Volume{testVolume}
	jaeger.Spec.Storage.Dependencies.JaegerCommonSpec.Volumes = testVolumes

	testVolumeMountName := "testVolumeMount"
	testMountPath := "/es-tls"
	testVolumeMount := corev1.VolumeMount{
		Name:      testVolumeMountName,
		ReadOnly:  false,
		MountPath: testMountPath,
	}
	testVolumeMounts := []corev1.VolumeMount{testVolumeMount}
	jaeger.Spec.Storage.Dependencies.JaegerCommonSpec.VolumeMounts = testVolumeMounts

	cjob := CreateSparkDependencies(jaeger).(*batchv1.CronJob)
	assert.Equal(t, testVolumeMountName, cjob.Spec.JobTemplate.Spec.Template.Spec.Containers[0].VolumeMounts[0].Name)
	assert.False(t, cjob.Spec.JobTemplate.Spec.Template.Spec.Containers[0].VolumeMounts[0].ReadOnly)
	assert.Equal(t, testMountPath, cjob.Spec.JobTemplate.Spec.Template.Spec.Containers[0].VolumeMounts[0].MountPath)

	assert.Equal(t, testVolumeName, cjob.Spec.JobTemplate.Spec.Template.Spec.Volumes[0].Name)
	assert.Equal(t, testConfigMapName, cjob.Spec.JobTemplate.Spec.Template.Spec.Volumes[0].ConfigMap.Name)
}

func TestSparkDependenciesImagePullSecrets(t *testing.T) {
	jaeger := v1.NewJaeger(types.NamespacedName{Name: "TestSparkDependenciesImagePullSecrets"})
	const pullSecret = "mysecret"
	jaeger.Spec.ImagePullSecrets = []corev1.LocalObjectReference{
		{
			Name: pullSecret,
		},
	}

	cjob := CreateSparkDependencies(jaeger).(*batchv1.CronJob)

	assert.Equal(t, pullSecret, cjob.Spec.JobTemplate.Spec.Template.Spec.ImagePullSecrets[0].Name)
}

func TestSparkDependenciesContainerSecurityContext(t *testing.T) {
	trueVar := true
	falseVar := false
	idVar := int64(1234)
	securityContextVar := corev1.SecurityContext{
		RunAsNonRoot:             &trueVar,
		RunAsGroup:               &idVar,
		RunAsUser:                &idVar,
		Capabilities:             &corev1.Capabilities{Drop: []corev1.Capability{"ALL"}},
		Privileged:               &falseVar,
		AllowPrivilegeEscalation: &falseVar,
		SeccompProfile:           &corev1.SeccompProfile{Type: "RuntimeDefault"},
	}
	jaeger := v1.NewJaeger(types.NamespacedName{Name: "TestSparkDependenciesContainerSecurityContext"})
	jaeger.Spec.Storage.Dependencies.JaegerCommonSpec.ContainerSecurityContext = &securityContextVar
	cjob := CreateSparkDependencies(jaeger).(*batchv1.CronJob)

	assert.Equal(t, securityContextVar, *cjob.Spec.JobTemplate.Spec.Template.Spec.Containers[0].SecurityContext)
}

func TestSparkDependenciesSecurityContextOverride(t *testing.T) {
	trueVar := true
	falseVar := false
	idVar1 := int64(1234)
	idVar2 := int64(4321)
	securityContextVar := corev1.SecurityContext{
		RunAsNonRoot:             &trueVar,
		RunAsGroup:               &idVar1,
		RunAsUser:                &idVar1,
		Capabilities:             &corev1.Capabilities{Drop: []corev1.Capability{"ALL"}},
		Privileged:               &falseVar,
		AllowPrivilegeEscalation: &falseVar,
		SeccompProfile:           &corev1.SeccompProfile{Type: "RuntimeDefault"},
	}
	overrideSecurityContextVar := corev1.SecurityContext{
		RunAsNonRoot:             &trueVar,
		RunAsGroup:               &idVar2,
		RunAsUser:                &idVar2,
		Capabilities:             &corev1.Capabilities{Drop: []corev1.Capability{"ALL"}},
		Privileged:               &falseVar,
		AllowPrivilegeEscalation: &falseVar,
		SeccompProfile:           &corev1.SeccompProfile{Type: "RuntimeDefault"},
	}
	jaeger := v1.NewJaeger(types.NamespacedName{Name: "TestSparkDependenciesSecurityContextOverride"})
	jaeger.Spec.ContainerSecurityContext = &securityContextVar
	jaeger.Spec.Storage.Dependencies.JaegerCommonSpec.ContainerSecurityContext = &overrideSecurityContextVar
	cjob := CreateSparkDependencies(jaeger).(*batchv1.CronJob)

	assert.Equal(t, overrideSecurityContextVar, *cjob.Spec.JobTemplate.Spec.Template.Spec.Containers[0].SecurityContext)
}
