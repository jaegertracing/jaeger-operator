package strategy

import (
	"context"
	"fmt"
	"strings"

	batchv1beta1 "k8s.io/api/batch/v1beta1"

	"github.com/spf13/viper"
	"go.opentelemetry.io/otel/global"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"

	"github.com/jaegertracing/jaeger-operator/pkg/account"
	v1 "github.com/jaegertracing/jaeger-operator/pkg/apis/jaegertracing/v1"
	crb "github.com/jaegertracing/jaeger-operator/pkg/clusterrolebinding"
	"github.com/jaegertracing/jaeger-operator/pkg/config/ca"
	"github.com/jaegertracing/jaeger-operator/pkg/config/otelconfig"
	"github.com/jaegertracing/jaeger-operator/pkg/config/sampling"
	configmap "github.com/jaegertracing/jaeger-operator/pkg/config/ui"
	"github.com/jaegertracing/jaeger-operator/pkg/cronjob"
	"github.com/jaegertracing/jaeger-operator/pkg/deployment"
	"github.com/jaegertracing/jaeger-operator/pkg/ingress"
	"github.com/jaegertracing/jaeger-operator/pkg/inject"
	"github.com/jaegertracing/jaeger-operator/pkg/kafka"
	"github.com/jaegertracing/jaeger-operator/pkg/route"
	"github.com/jaegertracing/jaeger-operator/pkg/storage"
	"github.com/jaegertracing/jaeger-operator/pkg/util"
)

func newStreamingStrategy(ctx context.Context, jaeger *v1.Jaeger) S {
	tracer := global.TraceProvider().GetTracer(v1.ReconciliationTracer)
	ctx, span := tracer.Start(ctx, "newStreamingStrategy")
	defer span.End()

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

	// add the optional OpenShift trusted CA config map
	if cm := ca.GetTrustedCABundle(jaeger); cm != nil {
		manifest.configMaps = append(manifest.configMaps, *cm)
	}

	if cm := otelconfig.Get(jaeger); len(cm) > 0 {
		manifest.configMaps = append(manifest.configMaps, cm...)
	}

	_, pfound := jaeger.Spec.Collector.Options.GenericMap()["kafka.producer.brokers"]
	_, cfound := jaeger.Spec.Ingester.Options.GenericMap()["kafka.consumer.brokers"]
	provisioned := jaeger.Annotations[v1.AnnotationProvisionedKafkaKey] == v1.AnnotationProvisionedKafkaValue

	// we provision a Kafka when no brokers have been set, or, when we are not in the first run,
	// when we know we've been the ones placing the broker information in the configuration
	if (!pfound && !cfound) || provisioned {
		jaeger.Logger().Info("Provisioning Kafka, this might take a while...")
		manifest = autoProvisionKafka(ctx, jaeger, manifest)
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

	// add autoscalers
	manifest.horizontalPodAutoscalers = append(collector.Autoscalers(), ingester.Autoscalers()...)

	if isBoolTrue(jaeger.Spec.Storage.Dependencies.Enabled) {
		if cronjob.SupportedStorage(jaeger.Spec.Storage.Type) {
			manifest.cronJobs = append(manifest.cronJobs, *cronjob.CreateSparkDependencies(jaeger))
		} else {
			jaeger.Logger().WithField("type", jaeger.Spec.Storage.Type).Warn("Skipping spark dependencies job due to unsupported storage.")
		}
	}

	var indexCleaner *batchv1beta1.CronJob
	if isBoolTrue(jaeger.Spec.Storage.EsIndexCleaner.Enabled) {
		if strings.EqualFold(jaeger.Spec.Storage.Type, "elasticsearch") {
			indexCleaner = cronjob.CreateEsIndexCleaner(jaeger)
		} else {
			jaeger.Logger().WithField("type", jaeger.Spec.Storage.Type).Warn("Skipping Elasticsearch index cleaner job due to unsupported storage.")
		}
	}

	var esRollover []batchv1beta1.CronJob
	if storage.EnableRollover(jaeger.Spec.Storage) {
		esRollover = cronjob.CreateRollover(jaeger)
	}

	// prepare the deployments, which may get changed by the elasticsearch routine
	cDep := collector.Get()
	queryDep := inject.Sidecar(jaeger, inject.OAuthProxy(jaeger, query.Get()))
	var ingesterDep *appsv1.Deployment
	if d := ingester.Get(); d != nil {
		ingesterDep = d
	}
	manifest.dependencies = storage.Dependencies(jaeger)

	// assembles the pieces for an elasticsearch self-provisioned deployment via the elasticsearch operator
	if storage.ShouldDeployElasticsearch(jaeger.Spec.Storage) {
		var jobs []*corev1.PodSpec
		for i := range manifest.dependencies {
			jobs = append(jobs, &manifest.dependencies[i].Spec.Template.Spec)
		}
		if indexCleaner != nil {
			jobs = append(jobs, &indexCleaner.Spec.JobTemplate.Spec.Template.Spec)
		}
		for i := range esRollover {
			jobs = append(jobs, &esRollover[i].Spec.JobTemplate.Spec.Template.Spec)
		}
		deps := []*appsv1.Deployment{queryDep}
		if ingesterDep != nil {
			deps = append(deps, ingesterDep)
		}
		autoProvisionElasticsearch(&manifest, jaeger, jobs, deps)
	}
	manifest.deployments = []appsv1.Deployment{*cDep, *queryDep}
	if ingesterDep != nil {
		manifest.deployments = append(manifest.deployments, *ingesterDep)
	}

	// the index cleaner ES job, which may have been changed by the ES self-provisioning routine
	if indexCleaner != nil {
		manifest.cronJobs = append(manifest.cronJobs, *indexCleaner)
	}
	if len(esRollover) > 0 {
		manifest.cronJobs = append(manifest.cronJobs, esRollover...)
	}

	return manifest
}

func autoProvisionKafka(ctx context.Context, jaeger *v1.Jaeger, manifest S) S {
	tracer := global.TraceProvider().GetTracer(v1.ReconciliationTracer)
	ctx, span := tracer.Start(ctx, "autoProvisionKafka")
	defer span.End()

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

	// store the new volumes/volume mounts in a common spec, later to be merged with the instance's common spec
	commonSpec := v1.JaegerCommonSpec{}

	// this is the volume containing the client TLS details, like the cert and key
	kuVolume := corev1.Volume{
		Name: fmt.Sprintf("kafkauser-%s", ku.Name),
		VolumeSource: corev1.VolumeSource{
			Secret: &corev1.SecretVolumeSource{
				SecretName: ku.Name,
			},
		},
	}
	// this is the volume containing the CA cluster cert
	kuCAVolume := corev1.Volume{
		Name: fmt.Sprintf("kafkauser-%s-cluster-ca", jaeger.Name), // the cluster name is the jaeger name
		VolumeSource: corev1.VolumeSource{
			Secret: &corev1.SecretVolumeSource{
				SecretName: fmt.Sprintf("%s-cluster-ca-cert", jaeger.Name),
			},
		},
	}
	commonSpec.Volumes = append(commonSpec.Volumes, kuVolume, kuCAVolume)

	// and finally, the mount paths to have the secrets in the container file system
	kuVolumeMount := corev1.VolumeMount{
		Name:      fmt.Sprintf("kafkauser-%s", ku.Name),
		MountPath: clientCertPath,
	}
	kuCAVolumeMount := corev1.VolumeMount{
		Name:      fmt.Sprintf("kafkauser-%s-cluster-ca", jaeger.Name), // the cluster name is the jaeger name
		MountPath: clusterCAPath,
	}
	commonSpec.VolumeMounts = append(commonSpec.VolumeMounts, kuVolumeMount, kuCAVolumeMount)

	brokers := fmt.Sprintf("%s-kafka-bootstrap.%s.svc.cluster.local:9093", k.Name, k.Namespace)

	collectorOpts := jaeger.Spec.Collector.Options.GenericMap()
	ingesterOpts := jaeger.Spec.Ingester.Options.GenericMap()

	collectorOpts["kafka.producer.brokers"] = brokers
	collectorOpts["kafka.producer.authentication"] = "tls"
	collectorOpts["kafka.producer.tls.enabled"] = "true"
	collectorOpts["kafka.producer.tls.ca"] = fmt.Sprintf("%s/ca.crt", clusterCAPath)
	collectorOpts["kafka.producer.tls.cert"] = fmt.Sprintf("%s/user.crt", clientCertPath)
	collectorOpts["kafka.producer.tls.key"] = fmt.Sprintf("%s/user.key", clientCertPath)

	ingesterOpts["kafka.consumer.brokers"] = brokers
	ingesterOpts["kafka.consumer.authentication"] = "tls"
	ingesterOpts["kafka.consumer.tls.enabled"] = "true"
	ingesterOpts["kafka.consumer.tls.ca"] = fmt.Sprintf("%s/ca.crt", clusterCAPath)
	ingesterOpts["kafka.consumer.tls.cert"] = fmt.Sprintf("%s/user.crt", clientCertPath)
	ingesterOpts["kafka.consumer.tls.key"] = fmt.Sprintf("%s/user.key", clientCertPath)

	jaeger.Spec.Collector.Options = v1.NewOptions(collectorOpts)
	jaeger.Spec.Ingester.Options = v1.NewOptions(ingesterOpts)
	jaeger.Spec.JaegerCommonSpec = *util.Merge([]v1.JaegerCommonSpec{commonSpec, jaeger.Spec.JaegerCommonSpec})

	return manifest
}
