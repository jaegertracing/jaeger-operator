package deployment

import (
	"fmt"
	"sort"
	"strconv"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/util/intstr"

	v1 "github.com/jaegertracing/jaeger-operator/apis/v1"
	"github.com/jaegertracing/jaeger-operator/pkg/account"
	"github.com/jaegertracing/jaeger-operator/pkg/config/ca"
	"github.com/jaegertracing/jaeger-operator/pkg/config/sampling"
	"github.com/jaegertracing/jaeger-operator/pkg/config/tls"
	"github.com/jaegertracing/jaeger-operator/pkg/service"
	"github.com/jaegertracing/jaeger-operator/pkg/storage"
	"github.com/jaegertracing/jaeger-operator/pkg/util"
)

// Collector builds pods for jaegertracing/jaeger-collector
type Collector struct {
	jaeger *v1.Jaeger
}

// NewCollector builds a new Collector struct based on the given spec
func NewCollector(jaeger *v1.Jaeger) *Collector {
	return &Collector{jaeger: jaeger}
}

// Get returns a collector pod
func (c *Collector) Get() *appsv1.Deployment {
	c.jaeger.Logger().V(-1).Info("assembling a collector deployment")

	labels := c.labels()
	trueVar := true
	falseVar := false

	args := c.jaeger.Spec.Collector.Options.ToArgs()

	adminPort := util.GetAdminPort(args, 14269)

	baseCommonSpec := v1.JaegerCommonSpec{
		Annotations: map[string]string{
			"prometheus.io/scrape": "true",
			"prometheus.io/port":   strconv.Itoa(int(adminPort)),
			"linkerd.io/inject":    "disabled",
		},
		Labels: labels,
	}

	commonSpec := util.Merge([]v1.JaegerCommonSpec{c.jaeger.Spec.Collector.JaegerCommonSpec, c.jaeger.Spec.JaegerCommonSpec, baseCommonSpec})
	_, ok := commonSpec.Annotations["sidecar.istio.io/inject"]
	if !ok {
		commonSpec.Annotations["sidecar.istio.io/inject"] = "false"
	}

	var envFromSource []corev1.EnvFromSource
	if len(c.jaeger.Spec.Storage.SecretName) > 0 {
		envFromSource = append(envFromSource, corev1.EnvFromSource{
			SecretRef: &corev1.SecretEnvSource{
				LocalObjectReference: corev1.LocalObjectReference{
					Name: c.jaeger.Spec.Storage.SecretName,
				},
			},
		})
	}

	storageType := c.jaeger.Spec.Storage.Type
	// If strategy is DeploymentStrategyStreaming, then change storage type
	// to Kafka, and the storage options will be used in the Ingester instead
	if c.jaeger.Spec.Strategy == v1.DeploymentStrategyStreaming {
		storageType = v1.JaegerKafkaStorage
		if len(c.jaeger.Spec.Collector.KafkaSecretName) > 0 {
			envFromSource = append(envFromSource, corev1.EnvFromSource{
				SecretRef: &corev1.SecretEnvSource{
					LocalObjectReference: corev1.LocalObjectReference{
						Name: c.jaeger.Spec.Collector.KafkaSecretName,
					},
				},
			})
		}
	}
	options := util.AllArgs(c.jaeger.Spec.Collector.Options,
		c.jaeger.Spec.Storage.Options.Filter(storageType.OptionsPrefix()))

	sampling.Update(c.jaeger, commonSpec, &options)
	if len(util.FindItem("--collector.grpc.tls.enabled=", args)) == 0 {
		tls.Update(c.jaeger, commonSpec, &options)
	}
	ca.Update(c.jaeger, commonSpec)
	storage.UpdateGRPCPlugin(c.jaeger, commonSpec)

	// ensure we have a consistent order of the arguments
	// see https://github.com/jaegertracing/jaeger-operator/issues/334
	sort.Strings(options)

	priorityClassName := ""
	if c.jaeger.Spec.Collector.PriorityClassName != "" {
		priorityClassName = c.jaeger.Spec.Collector.PriorityClassName
	}

	strategy := appsv1.DeploymentStrategy{
		Type: appsv1.RecreateDeploymentStrategyType,
	}

	if c.jaeger.Spec.Collector.Strategy != nil {
		strategy = *c.jaeger.Spec.Collector.Strategy
	}

	livenessProbe := &corev1.Probe{
		ProbeHandler: corev1.ProbeHandler{
			HTTPGet: &corev1.HTTPGetAction{
				Path: "/",
				Port: intstr.FromInt(int(adminPort)),
			},
		},
		InitialDelaySeconds: 5,
		PeriodSeconds:       15,
		FailureThreshold:    5,
	}

	if c.jaeger.Spec.Collector.LivenessProbe != nil {
		livenessProbe = c.jaeger.Spec.Collector.LivenessProbe
	}

	envVars := []corev1.EnvVar{
		{
			Name:  "SPAN_STORAGE_TYPE",
			Value: string(storageType),
		},
		{
			Name:  "COLLECTOR_ZIPKIN_HOST_PORT",
			Value: ":9411",
		},
	}

	ports := []corev1.ContainerPort{
		{
			ContainerPort: 9411,
			Name:          "zipkin",
		},
		{
			ContainerPort: 14267,
			Name:          "c-tchan-trft", // for collector
		},
		{
			ContainerPort: 14268,
			Name:          "c-binary-trft",
		},
		{
			ContainerPort: adminPort,
			Name:          "admin-http",
		},
		{
			ContainerPort: 14250,
			Name:          "grpc",
		},
	}

	var nodeSelector map[string]string
	if c.jaeger.Spec.Collector.NodeSelector != nil {
		nodeSelector = c.jaeger.Spec.Collector.NodeSelector
	}

	deployment := &appsv1.Deployment{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "apps/v1",
			Kind:       "Deployment",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:        c.name(),
			Namespace:   c.jaeger.Namespace,
			Labels:      commonSpec.Labels,
			Annotations: baseCommonSpec.Annotations,
			OwnerReferences: []metav1.OwnerReference{{
				APIVersion: c.jaeger.APIVersion,
				Kind:       c.jaeger.Kind,
				Name:       c.jaeger.Name,
				UID:        c.jaeger.UID,
				Controller: &trueVar,
			}},
		},
		Spec: appsv1.DeploymentSpec{
			Replicas: c.jaeger.Spec.Collector.Replicas,
			Selector: &metav1.LabelSelector{
				MatchLabels: labels,
			},
			Strategy: strategy,
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels:      commonSpec.Labels,
					Annotations: commonSpec.Annotations,
				},
				Spec: corev1.PodSpec{
					ImagePullSecrets: c.jaeger.Spec.ImagePullSecrets,
					Containers: []corev1.Container{{
						Image:         util.ImageName(c.jaeger.Spec.Collector.Image, "jaeger-collector-image"),
						Name:          "jaeger-collector",
						Args:          options,
						Env:           append(envVars, getOTLPEnvVars(options)...),
						VolumeMounts:  commonSpec.VolumeMounts,
						EnvFrom:       envFromSource,
						Ports:         append(ports, getOTLPContainePorts(options)...),
						LivenessProbe: livenessProbe,
						ReadinessProbe: &corev1.Probe{
							ProbeHandler: corev1.ProbeHandler{
								HTTPGet: &corev1.HTTPGetAction{
									Path: "/",
									Port: intstr.FromInt(int(adminPort)),
								},
							},
							InitialDelaySeconds: 1,
						},
						Lifecycle:       c.jaeger.Spec.Collector.Lifecycle,
						Resources:       commonSpec.Resources,
						ImagePullPolicy: commonSpec.ImagePullPolicy,
						SecurityContext: commonSpec.ContainerSecurityContext,
					}},
					PriorityClassName:             priorityClassName,
					Volumes:                       commonSpec.Volumes,
					ServiceAccountName:            account.JaegerServiceAccountFor(c.jaeger, account.CollectorComponent),
					Affinity:                      commonSpec.Affinity,
					Tolerations:                   commonSpec.Tolerations,
					SecurityContext:               commonSpec.SecurityContext,
					EnableServiceLinks:            &falseVar,
					InitContainers:                storage.GetGRPCPluginInitContainers(c.jaeger, commonSpec),
					TerminationGracePeriodSeconds: c.jaeger.Spec.Collector.TerminationGracePeriodSeconds,
				},
			},
		},
	}
	if nodeSelector != nil {
		deployment.Spec.Template.Spec.NodeSelector = nodeSelector
	}
	return deployment
}

// Services returns a list of services to be deployed along with the all-in-one deployment
func (c *Collector) Services() []*corev1.Service {
	return service.NewCollectorServices(c.jaeger, c.labels())
}

// Autoscalers returns a list of HPAs based on this collector
func (c *Collector) Autoscalers() []runtime.Object {
	return autoscalers(c)
}

func (c *Collector) labels() map[string]string {
	return util.Labels(c.name(), "collector", *c.jaeger)
}

func (c *Collector) hpaLabels() map[string]string {
	labels := c.labels()
	labels["app.kubernetes.io/component"] = "hpa-collector"
	return labels
}

func (c *Collector) name() string {
	return fmt.Sprintf("%s-collector", c.jaeger.Name)
}

func (c *Collector) commonSpec() v1.JaegerCommonSpec {
	return c.jaeger.Spec.Collector.JaegerCommonSpec
}

func (c *Collector) autoscalingSpec() v1.AutoScaleSpec {
	return c.jaeger.Spec.Collector.AutoScaleSpec
}

func (c *Collector) jaegerInstance() *v1.Jaeger {
	return c.jaeger
}

func (c *Collector) replicas() *int32 {
	return c.jaeger.Spec.Collector.Replicas
}
