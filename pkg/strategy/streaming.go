package strategy

import (
	"fmt"
	"strings"

	"github.com/spf13/viper"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"

	"github.com/jaegertracing/jaeger-operator/pkg/account"
	v1 "github.com/jaegertracing/jaeger-operator/pkg/apis/jaegertracing/v1"
	crb "github.com/jaegertracing/jaeger-operator/pkg/clusterrolebinding"
	"github.com/jaegertracing/jaeger-operator/pkg/config/sampling"
	configmap "github.com/jaegertracing/jaeger-operator/pkg/config/ui"
	"github.com/jaegertracing/jaeger-operator/pkg/cronjob"
	"github.com/jaegertracing/jaeger-operator/pkg/deployment"
	"github.com/jaegertracing/jaeger-operator/pkg/ingress"
	"github.com/jaegertracing/jaeger-operator/pkg/inject"
	"github.com/jaegertracing/jaeger-operator/pkg/kafka"
	"github.com/jaegertracing/jaeger-operator/pkg/route"
	"github.com/jaegertracing/jaeger-operator/pkg/storage"
)

func newStreamingStrategy(jaeger *v1.Jaeger) S {
	manifest := S{typ: v1.DeploymentStrategyStreaming}
	collector := deployment.NewCollector(jaeger)
	query := deployment.NewQuery(jaeger)
	agent := deployment.NewAgent(jaeger)
	ingester := deployment.NewIngester(jaeger)

	// add all service accounts
	for _, acc := range account.Get(jaeger) {
		manifest.accounts = append(manifest.accounts, *acc)
	}

	// add all cluster role bindings
	manifest.clusterRoleBindings = crb.Get(jaeger)

	// add the config map
	if cm := configmap.NewUIConfig(jaeger).Get(); cm != nil {
		manifest.configMaps = append(manifest.configMaps, *cm)
	}

	// add the Sampling config map
	if cm := sampling.NewConfig(jaeger).Get(); cm != nil {
		manifest.configMaps = append(manifest.configMaps, *cm)
	}

	_, pfound := asOptionsMap(jaeger.Spec.Collector.Options.Map())["kafka.producer.brokers"]
	_, cfound := asOptionsMap(jaeger.Spec.Ingester.Options.Map())["kafka.consumer.brokers"]
	provisioned := jaeger.Annotations[v1.AnnotationProvisionedKafkaKey] == v1.AnnotationProvisionedKafkaValue

	// we provision a Kafka when no brokers have been set, or, when we are not in the first run,
	// when we know we've been the ones placing the broker information in the configuration
	if (!pfound && !cfound) || provisioned {
		manifest = autoProvisionKafka(jaeger, manifest)
	}

	// add the deployments
	manifest.deployments = []appsv1.Deployment{*collector.Get(), *inject.Sidecar(jaeger, inject.OAuthProxy(jaeger, query.Get()))}

	if d := ingester.Get(); d != nil {
		manifest.deployments = append(manifest.deployments, *d)
	}

	// add the daemonsets
	if ds := agent.Get(); ds != nil {
		manifest.daemonSets = []appsv1.DaemonSet{*ds}
	}

	// add the services
	for _, svc := range collector.Services() {
		manifest.services = append(manifest.services, *svc)
	}

	for _, svc := range query.Services() {
		manifest.services = append(manifest.services, *svc)
	}

	// add the routes/ingresses
	if viper.GetString("platform") == v1.FlagPlatformOpenShift {
		if q := route.NewQueryRoute(jaeger).Get(); nil != q {
			manifest.routes = append(manifest.routes, *q)
		}
	} else {
		if q := ingress.NewQueryIngress(jaeger).Get(); nil != q {
			manifest.ingresses = append(manifest.ingresses, *q)
		}
	}

	if isBoolTrue(jaeger.Spec.Storage.Dependencies.Enabled) {
		if cronjob.SupportedStorage(jaeger.Spec.Storage.Type) {
			manifest.cronJobs = append(manifest.cronJobs, *cronjob.CreateSparkDependencies(jaeger))
		} else {
			jaeger.Logger().WithField("type", jaeger.Spec.Storage.Type).Warn("Skipping spark dependencies job due to unsupported storage.")
		}
	}

	if isBoolTrue(jaeger.Spec.Storage.EsIndexCleaner.Enabled) {
		if strings.EqualFold(jaeger.Spec.Storage.Type, "elasticsearch") {
			manifest.cronJobs = append(manifest.cronJobs, *cronjob.CreateEsIndexCleaner(jaeger))
		} else {
			jaeger.Logger().WithField("type", jaeger.Spec.Storage.Type).Warn("Skipping Elasticsearch index cleaner job due to unsupported storage.")
		}
	}

	manifest.dependencies = storage.Dependencies(jaeger)

	return manifest
}

func autoProvisionKafka(jaeger *v1.Jaeger, manifest S) S {
	if jaeger.Annotations == nil {
		jaeger.Annotations = map[string]string{}
	}
	// mark that we auto provisioned a kafka for this instance
	jaeger.Annotations[v1.AnnotationProvisionedKafkaKey] = v1.AnnotationProvisionedKafkaValue

	k := kafka.Persistent(jaeger)
	ku := kafka.User(jaeger)
	manifest.kafkas = append(manifest.kafkas, k)
	manifest.kafkaUsers = append(manifest.kafkaUsers, ku)

	// these are the in-container paths, available to the Jaeger containers (ingester/collector)
	clusterCAPath := fmt.Sprintf("/var/run/secrets/%s-cluster-ca", jaeger.Name)
	clientCertPath := fmt.Sprintf("/var/run/secrets/%s", ku.Name)

	// this is the volume containing the client TLS details, like the cert and key
	kuVolume := corev1.Volume{
		Name: ku.Name,
		VolumeSource: corev1.VolumeSource{
			Secret: &corev1.SecretVolumeSource{
				SecretName: ku.Name,
			},
		},
	}
	// this is the volume containing the CA cluster cert
	kuCAVolume := corev1.Volume{
		Name: fmt.Sprintf("%s-cluster-ca", jaeger.Name), // the cluster name is the jaeger name
		VolumeSource: corev1.VolumeSource{
			Secret: &corev1.SecretVolumeSource{
				SecretName: fmt.Sprintf("%s-cluster-ca-cert", jaeger.Name),
			},
		},
	}
	jaeger.Spec.Volumes = replaceVolume(jaeger.Spec.Volumes, kuVolume, kuCAVolume)

	// and finally, the mount paths to have the secrets in the container file system
	kuVolumeMount := corev1.VolumeMount{
		Name:      ku.Name,
		MountPath: clientCertPath,
	}
	kuCAVolumeMount := corev1.VolumeMount{
		Name:      fmt.Sprintf("%s-cluster-ca", jaeger.Name), // the cluster name is the jaeger name
		MountPath: clusterCAPath,
	}
	jaeger.Spec.VolumeMounts = replaceVolumeMount(jaeger.Spec.VolumeMounts, kuVolumeMount, kuCAVolumeMount)

	brokers := fmt.Sprintf("%s-kafka-bootstrap.kafka.svc.cluster.local:9093", k.Name)

	collectorOpts := asOptionsMap(jaeger.Spec.Collector.Options.Map())
	ingesterOpts := asOptionsMap(jaeger.Spec.Ingester.Options.Map())

	collectorOpts["kafka.producer.brokers"] = brokers
	collectorOpts["kafka.producer.authentication"] = "tls"
	collectorOpts["kafka.producer.tls.ca"] = fmt.Sprintf("%s/ca.crt", clusterCAPath)
	collectorOpts["kafka.producer.tls.cert"] = fmt.Sprintf("%s/user.crt", clientCertPath)
	collectorOpts["kafka.producer.tls.key"] = fmt.Sprintf("%s/user.key", clientCertPath)
	if _, found := collectorOpts["kafka.producer.topic"]; !found {
		collectorOpts["kafka.producer.topic"] = "jaeger-spans"
	}

	ingesterOpts["kafka.consumer.brokers"] = brokers
	ingesterOpts["kafka.consumer.authentication"] = "tls"
	ingesterOpts["kafka.consumer.tls.ca"] = fmt.Sprintf("%s/ca.crt", clusterCAPath)
	ingesterOpts["kafka.consumer.tls.cert"] = fmt.Sprintf("%s/user.crt", clientCertPath)
	ingesterOpts["kafka.consumer.tls.key"] = fmt.Sprintf("%s/user.key", clientCertPath)
	if _, found := collectorOpts["kafka.consumer.topic"]; !found {
		ingesterOpts["kafka.consumer.topic"] = "jaeger-spans"
	}

	jaeger.Spec.Collector.Options = v1.NewOptions(collectorOpts)
	jaeger.Spec.Ingester.Options = v1.NewOptions(ingesterOpts)

	return manifest
}

func asOptionsMap(in map[string]string) map[string]interface{} {
	out := make(map[string]interface{})
	for k, v := range in {
		out[k] = v
	}
	return out
}

func replaceVolume(volumes []corev1.Volume, name ...corev1.Volume) []corev1.Volume {
	out := []corev1.Volume{}

	for _, v := range volumes {
		add := true

		for _, n := range name {
			if n.Name == v.Name {
				// we have an existing volume with the same name one of the new volumes
				// skip adding the old one to the output
				add = false
				break
			}
		}

		if add {
			out = append(out, v)
		}
	}

	// now, we add the new volumes
	for _, n := range name {
		out = append(out, n)
	}

	return out
}

func replaceVolumeMount(volumes []corev1.VolumeMount, name ...corev1.VolumeMount) []corev1.VolumeMount {
	out := []corev1.VolumeMount{}

	for _, v := range volumes {
		add := true

		for _, n := range name {
			if n.Name == v.Name {
				// we have an existing volume with the same name one of the new volumes
				// skip adding the old one to the output
				add = false
				break
			}
		}

		if add {
			out = append(out, v)
		}
	}

	// now, we add the new volumes
	for _, n := range name {
		out = append(out, n)
	}

	return out
}
