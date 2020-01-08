package strategy

import (
	"context"
	"fmt"
	"strings"
	"testing"

	"github.com/spf13/viper"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"

	v1 "github.com/jaegertracing/jaeger-operator/pkg/apis/jaegertracing/v1"
	"github.com/jaegertracing/jaeger-operator/pkg/storage"
)

func init() {
	viper.SetDefault("jaeger-agent-image", "jaegertracing/jaeger-agent")
}

func TestCreateStreamingDeployment(t *testing.T) {
	name := "my-instance"
	c := newStreamingStrategy(context.Background(), v1.NewJaeger(types.NamespacedName{Name: name}), &storage.ElasticsearchDeployment{})
	assertDeploymentsAndServicesForStreaming(t, name, c, false, false, false)
}

func TestStreamingKafkaProvisioning(t *testing.T) {
	name := "my-instance"
	c := newStreamingStrategy(context.Background(), v1.NewJaeger(types.NamespacedName{Name: name}), &storage.ElasticsearchDeployment{})

	// one Kafka, one KafkaUser
	assert.Len(t, c.Kafkas(), 1)
	assert.Len(t, c.KafkaUsers(), 1)
}

func TestStreamingNoKafkaProvisioningWhenConsumerBrokersSet(t *testing.T) {
	name := "my-instance"
	jaeger := v1.NewJaeger(types.NamespacedName{Name: name})
	jaeger.Spec.Ingester.Options = v1.NewOptions(map[string]interface{}{
		"kafka.consumer.brokers": "my-cluster-kafka-brokers.kafka:9092",
	})
	c := newStreamingStrategy(context.Background(), jaeger, &storage.ElasticsearchDeployment{Jaeger: jaeger})

	// one Kafka, one KafkaUser
	assert.Len(t, c.Kafkas(), 0)
}

func TestStreamingNoKafkaProvisioningWhenProducerBrokersSet(t *testing.T) {
	name := "my-instance"
	jaeger := v1.NewJaeger(types.NamespacedName{Name: name})
	jaeger.Spec.Collector.Options = v1.NewOptions(map[string]interface{}{
		"kafka.producer.brokers": "my-cluster-kafka-brokers.kafka:9092",
	})
	c := newStreamingStrategy(context.Background(), jaeger, &storage.ElasticsearchDeployment{Jaeger: jaeger})

	// one Kafka, one KafkaUser
	assert.Len(t, c.Kafkas(), 0)
}

func TestCreateStreamingDeploymentOnOpenShift(t *testing.T) {
	viper.Set("platform", "openshift")
	defer viper.Reset()
	name := "my-instance"

	jaeger := v1.NewJaeger(types.NamespacedName{Name: name})
	normalize(context.Background(), jaeger)

	c := newStreamingStrategy(context.Background(), jaeger, &storage.ElasticsearchDeployment{Jaeger: jaeger})
	assertDeploymentsAndServicesForStreaming(t, name, c, false, true, false)
}

func TestCreateStreamingDeploymentWithDaemonSetAgent(t *testing.T) {
	name := "my-instance"

	j := v1.NewJaeger(types.NamespacedName{Name: name})
	j.Spec.Agent.Strategy = "DaemonSet"

	c := newStreamingStrategy(context.Background(), j, &storage.ElasticsearchDeployment{Jaeger: j})
	assertDeploymentsAndServicesForStreaming(t, name, c, true, false, false)
}

func TestCreateStreamingDeploymentWithUIConfigMap(t *testing.T) {
	name := "my-instance"

	j := v1.NewJaeger(types.NamespacedName{Name: name})
	j.Spec.UI.Options = v1.NewFreeForm(map[string]interface{}{
		"tracking": map[string]interface{}{
			"gaID": "UA-000000-2",
		},
	})

	c := newStreamingStrategy(context.Background(), j, &storage.ElasticsearchDeployment{Jaeger: j})
	assertDeploymentsAndServicesForStreaming(t, name, c, false, false, true)
}

func TestStreamingOptionsArePassed(t *testing.T) {
	jaeger := &v1.Jaeger{
		TypeMeta: metav1.TypeMeta{
			Kind:       "Jaeger",
			APIVersion: "jaegertracing.io/v1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      "simple-prod",
			Namespace: "simple-prod-ns",
		},
		Spec: v1.JaegerSpec{
			Strategy: v1.DeploymentStrategyStreaming,
			Collector: v1.JaegerCollectorSpec{
				Options: v1.NewOptions(map[string]interface{}{
					"kafka.producer.topic":   "mytopic",
					"kafka.producer.brokers": "my.broker:9092",
				}),
			},
			Ingester: v1.JaegerIngesterSpec{
				Options: v1.NewOptions(map[string]interface{}{
					"kafka.consumer.topic":    "mytopic",
					"kafka.consumer.brokers":  "my.broker:9092",
					"kafka.consumer.group-id": "mygroup",
				}),
			},
			Storage: v1.JaegerStorageSpec{
				Type: "elasticsearch",
				Options: v1.NewOptions(map[string]interface{}{
					"es.server-urls": "http://elasticsearch.default.svc:9200",
					"es.username":    "elastic",
					"es.password":    "changeme",
				}),
			},
		},
	}

	ctrl := For(context.TODO(), jaeger, []corev1.Secret{})
	deployments := ctrl.Deployments()
	for _, dep := range deployments {
		args := dep.Spec.Template.Spec.Containers[0].Args
		// Only the query and ingester should have the ES parameters defined
		var escount int
		for _, arg := range args {
			if strings.Contains(arg, "es.") {
				escount++
			}
		}
		if strings.Contains(dep.Name, "collector") {
			// Including parameters for sampling config and kafka topic
			assert.Len(t, args, 3)
			assert.Equal(t, 0, escount)
		} else if strings.Contains(dep.Name, "ingester") {
			// Including parameters for ES and kafka topic
			assert.Len(t, args, 6)
			assert.Equal(t, 3, escount)
		} else {
			// Including parameters for ES only
			assert.Len(t, args, 3)
			assert.Equal(t, 3, escount)
		}
	}
}

func TestDelegateStreamingDependencies(t *testing.T) {
	// for now, we just have storage dependencies
	j := v1.NewJaeger(types.NamespacedName{Name: "my-instance"})
	c := newStreamingStrategy(context.Background(), j, &storage.ElasticsearchDeployment{Jaeger: j})
	assert.Equal(t, c.Dependencies(), storage.Dependencies(j))
}

func assertDeploymentsAndServicesForStreaming(t *testing.T, name string, s S, hasDaemonSet bool, hasOAuthProxy bool, hasConfigMap bool) {
	expectedNumObjs := 7

	if hasDaemonSet {
		expectedNumObjs++
	}

	if hasOAuthProxy {
		expectedNumObjs++
	}

	if hasConfigMap {
		expectedNumObjs++
	}

	deployments := map[string]bool{
		fmt.Sprintf("%s-collector", name): false,
		fmt.Sprintf("%s-query", name):     false,
	}

	daemonsets := map[string]bool{
		fmt.Sprintf("%s-agent-daemonset", name): !hasDaemonSet,
	}

	services := map[string]bool{
		fmt.Sprintf("%s-collector", strings.ToLower(name)): false,
		fmt.Sprintf("%s-query", strings.ToLower(name)):     false,
	}

	ingresses := map[string]bool{}
	routes := map[string]bool{}
	if viper.GetString("platform") == v1.FlagPlatformOpenShift {
		routes[name] = false
	} else {
		ingresses[fmt.Sprintf("%s-query", name)] = false
	}

	serviceAccounts := map[string]bool{}
	if hasOAuthProxy {
		serviceAccounts[fmt.Sprintf("%s-ui-proxy", name)] = false
	}

	configMaps := map[string]bool{}
	if hasConfigMap {
		configMaps[fmt.Sprintf("%s-ui-configuration", name)] = false
	}
	assertHasAllObjects(t, name, s, deployments, daemonsets, services, ingresses, routes, serviceAccounts, configMaps)
}

func TestSparkDependenciesStreaming(t *testing.T) {
	testSparkDependencies(t, func(jaeger *v1.Jaeger) S {
		return newStreamingStrategy(context.Background(), jaeger, &storage.ElasticsearchDeployment{Jaeger: jaeger})
	})
}

func TestEsIndexClenarStreaming(t *testing.T) {
	testEsIndexCleaner(t, func(jaeger *v1.Jaeger) S {
		return newStreamingStrategy(context.Background(), jaeger, &storage.ElasticsearchDeployment{Jaeger: jaeger})
	})
}

func TestAgentSidecarIsInjectedIntoQueryForStreaming(t *testing.T) {
	j := v1.NewJaeger(types.NamespacedName{Name: "my-instance"})
	c := newStreamingStrategy(context.Background(), j, &storage.ElasticsearchDeployment{Jaeger: j})
	for _, dep := range c.Deployments() {
		if strings.HasSuffix(dep.Name, "-query") {
			assert.Equal(t, 2, len(dep.Spec.Template.Spec.Containers))
			assert.Equal(t, "jaeger-agent", dep.Spec.Template.Spec.Containers[1].Name)
		}
	}
}

func TestAutoProvisionedKafkaInjectsIntoInstance(t *testing.T) {
	name := "my-instance"
	jaeger := v1.NewJaeger(types.NamespacedName{Name: name, Namespace: "project"})
	jaeger.Spec.Collector.Options = v1.NewOptions(map[string]interface{}{})
	jaeger.Spec.Ingester.Options = v1.NewOptions(map[string]interface{}{})
	manifest := S{typ: v1.DeploymentStrategyStreaming}

	// test
	autoProvisionKafka(context.Background(), jaeger, manifest)

	// verify
	assert.Equal(t, v1.AnnotationProvisionedKafkaValue, jaeger.Annotations[v1.AnnotationProvisionedKafkaKey])

	assert.Equal(t, "my-instance-kafka-bootstrap.project.svc.cluster.local:9093", jaeger.Spec.Collector.Options.Map()["kafka.producer.brokers"])
	assert.Contains(t, jaeger.Spec.Collector.Options.Map(), "kafka.producer.authentication")
	assert.Contains(t, jaeger.Spec.Collector.Options.Map(), "kafka.producer.tls.key")
	assert.Contains(t, jaeger.Spec.Collector.Options.Map(), "kafka.producer.tls.cert")
	assert.Contains(t, jaeger.Spec.Collector.Options.Map(), "kafka.producer.tls.ca")
	assert.NotContains(t, jaeger.Spec.Collector.Options.Map(), "kafka.consumer.brokers")

	assert.Equal(t, "my-instance-kafka-bootstrap.project.svc.cluster.local:9093", jaeger.Spec.Ingester.Options.Map()["kafka.consumer.brokers"])
	assert.Contains(t, jaeger.Spec.Ingester.Options.Map(), "kafka.consumer.authentication")
	assert.Contains(t, jaeger.Spec.Ingester.Options.Map(), "kafka.consumer.tls.key")
	assert.Contains(t, jaeger.Spec.Ingester.Options.Map(), "kafka.consumer.tls.cert")
	assert.Contains(t, jaeger.Spec.Ingester.Options.Map(), "kafka.consumer.tls.ca")
	assert.NotContains(t, jaeger.Spec.Ingester.Options.Map(), "kafka.producer.brokers")

	assert.Len(t, jaeger.Spec.Volumes, 2)
	assert.Len(t, jaeger.Spec.VolumeMounts, 2)
}

func TestReplaceVolume(t *testing.T) {
	// prepare
	instance := v1.NewJaeger(types.NamespacedName{Name: "my-instance", Namespace: "tenant-1"})
	instance.Spec.Volumes = []corev1.Volume{
		{
			Name: "kafkauser-my-instance",
			VolumeSource: corev1.VolumeSource{
				Secret: &corev1.SecretVolumeSource{
					SecretName: "secret-name-a",
				},
			},
		}, {
			Name: "kafkauser-my-instance-cluster-ca",
			VolumeSource: corev1.VolumeSource{
				Secret: &corev1.SecretVolumeSource{
					SecretName: "secret-name-b",
				},
			},
		}, {
			Name: "volume-c",
			VolumeSource: corev1.VolumeSource{
				Secret: &corev1.SecretVolumeSource{
					SecretName: "secret-name-c",
				},
			},
		},
	}

	// test
	ctx := context.Background()
	autoProvisionKafka(ctx, instance, newStreamingStrategy(ctx, instance, &storage.ElasticsearchDeployment{Jaeger: instance}))

	// verify
	assert.Len(t, instance.Spec.Volumes, 3)

	found := 0
	for _, v := range instance.Spec.Volumes {
		if v.Name == "kafkauser-my-instance-cluster-ca" {
			found = found + 1
			assert.Equal(t, "my-instance-cluster-ca-cert", v.VolumeSource.Secret.SecretName)
		}
		if v.Name == "kafkauser-my-instance" {
			found = found + 1
			assert.Equal(t, "my-instance", v.VolumeSource.Secret.SecretName)
		}
	}
	assert.Equal(t, 2, found)
}

func TestReplaceVolumeMount(t *testing.T) {
	// prepare
	instance := v1.NewJaeger(types.NamespacedName{Name: "my-instance", Namespace: "tenant-1"})
	instance.Spec.VolumeMounts = []corev1.VolumeMount{
		{
			Name:      "kafkauser-my-instance-cluster-ca",
			MountPath: "/var/path",
		}, {
			Name:      "kafkauser-my-instance",
			MountPath: "/var/path",
		}, {
			Name:      "volume-c",
			MountPath: "/var/path-c",
		},
	}

	// test
	ctx := context.Background()
	autoProvisionKafka(ctx, instance, newStreamingStrategy(ctx, instance, &storage.ElasticsearchDeployment{Jaeger: instance}))

	// verify
	assert.Len(t, instance.Spec.VolumeMounts, 3)
	found := 0
	for _, v := range instance.Spec.VolumeMounts {
		if v.Name == "kafkauser-my-instance-cluster-ca" || v.Name == "kafkauser-my-instance" {
			found = found + 1
			assert.True(t, strings.HasPrefix(v.MountPath, "/var/run/secrets"))
		}
	}
	assert.Equal(t, 2, found)
}

func TestAutoProvisionedKafkaAndElasticsearch(t *testing.T) {
	verdad := true
	one := int(1)
	jaeger := v1.NewJaeger(types.NamespacedName{Name: t.Name()})
	jaeger.Spec.Storage.Type = "elasticsearch"
	jaeger.Spec.Storage.EsIndexCleaner.Enabled = &verdad
	jaeger.Spec.Storage.EsIndexCleaner.NumberOfDays = &one
	jaeger.Spec.Storage.Options = v1.NewOptions(map[string]interface{}{"es.use-aliases": true})

	es := &storage.ElasticsearchDeployment{Jaeger: jaeger, CertScript: "../../scripts/cert_generation.sh"}
	err := es.CleanCerts()
	require.NoError(t, err)
	defer es.CleanCerts()
	c := newStreamingStrategy(context.Background(), jaeger, es)
	// there should be index-cleaner, rollover, lookback
	assert.Equal(t, 3, len(c.cronJobs))
	assertEsInjectSecretsStreaming(t, c.cronJobs[0].Spec.JobTemplate.Spec.Template.Spec)
	assertEsInjectSecretsStreaming(t, c.cronJobs[1].Spec.JobTemplate.Spec.Template.Spec)
	assertEsInjectSecretsStreaming(t, c.cronJobs[2].Spec.JobTemplate.Spec.Template.Spec)
}

func assertEsInjectSecretsStreaming(t *testing.T, p corev1.PodSpec) {
	// first two volumes are from the common spec
	assert.Equal(t, 3, len(p.Volumes))
	assert.Equal(t, "certs", p.Volumes[2].Name)
	assert.Equal(t, "certs", p.Containers[0].VolumeMounts[2].Name)
	envs := map[string]corev1.EnvVar{}
	for _, e := range p.Containers[0].Env {
		envs[e.Name] = e
	}
	assert.Contains(t, envs, "ES_TLS")
	assert.Contains(t, envs, "ES_TLS_CA")
	assert.Contains(t, envs, "ES_TLS_KEY")
	assert.Contains(t, envs, "ES_TLS_CERT")
}
