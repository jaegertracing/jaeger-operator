package deployment

import (
	"fmt"
	"sort"
	"strconv"

	appsv1 "k8s.io/api/apps/v1"
	autoscalingv2beta2 "k8s.io/api/autoscaling/v2beta2"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"

	"github.com/jaegertracing/jaeger-operator/pkg/account"
	v1 "github.com/jaegertracing/jaeger-operator/pkg/apis/jaegertracing/v1"
	"github.com/jaegertracing/jaeger-operator/pkg/config/sampling"
	"github.com/jaegertracing/jaeger-operator/pkg/service"
	"github.com/jaegertracing/jaeger-operator/pkg/storage"
	"github.com/jaegertracing/jaeger-operator/pkg/util"
)

const (
	// we need to have an upper bound, and 100 seems like a "good" max value
	defaultMaxReplicas = int32(100)

	// for both memory and cpu
	defaultAvgUtilization = int32(90)
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
	c.jaeger.Logger().Debug("assembling a collector deployment")

	labels := c.labels()
	trueVar := true

	args := append(c.jaeger.Spec.Collector.Options.ToArgs())

	adminPort := util.GetPort("--admin-http-port=", args, 14269)

	baseCommonSpec := v1.JaegerCommonSpec{
		Annotations: map[string]string{
			"prometheus.io/scrape":    "true",
			"prometheus.io/port":      strconv.Itoa(int(adminPort)),
			"sidecar.istio.io/inject": "false",
			"linkerd.io/inject":       "disabled",
		},
		Labels: labels,
	}

	commonSpec := util.Merge([]v1.JaegerCommonSpec{c.jaeger.Spec.Collector.JaegerCommonSpec, c.jaeger.Spec.JaegerCommonSpec, baseCommonSpec})

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
		storageType = "kafka"
	}
	options := allArgs(c.jaeger.Spec.Collector.Options,
		c.jaeger.Spec.Storage.Options.Filter(storage.OptionsPrefix(storageType)))

	sampling.Update(c.jaeger, commonSpec, &options)

	// ensure we have a consistent order of the arguments
	// see https://github.com/jaegertracing/jaeger-operator/issues/334
	sort.Strings(options)

	replicaSize := c.jaeger.Spec.Collector.Replicas
	if replicaSize == nil || *replicaSize < 0 {
		s := int32(1)
		replicaSize = &s
	}

	return &appsv1.Deployment{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "apps/v1",
			Kind:       "Deployment",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:        c.name(),
			Namespace:   c.jaeger.Namespace,
			Labels:      commonSpec.Labels,
			Annotations: commonSpec.Annotations,
			OwnerReferences: []metav1.OwnerReference{
				metav1.OwnerReference{
					APIVersion: c.jaeger.APIVersion,
					Kind:       c.jaeger.Kind,
					Name:       c.jaeger.Name,
					UID:        c.jaeger.UID,
					Controller: &trueVar,
				},
			},
		},
		Spec: appsv1.DeploymentSpec{
			Replicas: replicaSize,
			Selector: &metav1.LabelSelector{
				MatchLabels: labels,
			},
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels:      commonSpec.Labels,
					Annotations: commonSpec.Annotations,
				},
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{{
						Image: util.ImageName(c.jaeger.Spec.Collector.Image, "jaeger-collector-image"),
						Name:  "jaeger-collector",
						Args:  options,
						Env: []corev1.EnvVar{
							corev1.EnvVar{
								Name:  "SPAN_STORAGE_TYPE",
								Value: storageType,
							},
							corev1.EnvVar{
								Name:  "COLLECTOR_ZIPKIN_HTTP_PORT",
								Value: "9411",
							},
						},
						VolumeMounts: commonSpec.VolumeMounts,
						EnvFrom:      envFromSource,
						Ports: []corev1.ContainerPort{
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
						},
						LivenessProbe: &corev1.Probe{
							Handler: corev1.Handler{
								HTTPGet: &corev1.HTTPGetAction{
									Path: "/",
									Port: intstr.FromInt(int(adminPort)),
								},
							},
							InitialDelaySeconds: 5,
							PeriodSeconds:       15,
							FailureThreshold:    5,
						},
						ReadinessProbe: &corev1.Probe{
							Handler: corev1.Handler{
								HTTPGet: &corev1.HTTPGetAction{
									Path: "/",
									Port: intstr.FromInt(int(adminPort)),
								},
							},
							InitialDelaySeconds: 1,
						},
						Resources: commonSpec.Resources,
					}},
					Volumes:            commonSpec.Volumes,
					ServiceAccountName: account.JaegerServiceAccountFor(c.jaeger, account.CollectorComponent),
					Affinity:           commonSpec.Affinity,
					Tolerations:        commonSpec.Tolerations,
					SecurityContext:    commonSpec.SecurityContext,
				},
			},
		},
	}
}

// Services returns a list of services to be deployed along with the all-in-one deployment
func (c *Collector) Services() []*corev1.Service {
	return service.NewCollectorServices(c.jaeger, c.labels())
}

// Autoscalers returns a list of HPAs based on this collector
func (c *Collector) Autoscalers() []autoscalingv2beta2.HorizontalPodAutoscaler {
	// fixed number of replicas is explicitly set, do not auto scale
	if c.jaeger.Spec.Collector.Replicas != nil {
		return []autoscalingv2beta2.HorizontalPodAutoscaler{}
	}

	// explicitly disabled, do not auto scale
	if c.jaeger.Spec.Collector.Autoscale != nil && *c.jaeger.Spec.Collector.Autoscale == false {
		return []autoscalingv2beta2.HorizontalPodAutoscaler{}
	}

	maxReplicas := int32(-1) // unset, or invalid value

	if nil != c.jaeger.Spec.Collector.MaxReplicas {
		maxReplicas = *c.jaeger.Spec.Collector.MaxReplicas
	}
	if maxReplicas < 0 {
		maxReplicas = defaultMaxReplicas
	}

	labels := c.labels()
	labels["app.kubernetes.io/component"] = "hpa-collector"
	baseCommonSpec := v1.JaegerCommonSpec{
		Labels: labels,
	}

	avgUtilization := defaultAvgUtilization
	trueVar := true
	commonSpec := util.Merge([]v1.JaegerCommonSpec{c.jaeger.Spec.Collector.JaegerCommonSpec, c.jaeger.Spec.JaegerCommonSpec, baseCommonSpec})

	// scale up when either CPU or memory is above 90%
	return []autoscalingv2beta2.HorizontalPodAutoscaler{{
		ObjectMeta: metav1.ObjectMeta{
			Name:        c.name(),
			Namespace:   c.jaeger.Namespace,
			Labels:      commonSpec.Labels,
			Annotations: commonSpec.Annotations,
			OwnerReferences: []metav1.OwnerReference{
				metav1.OwnerReference{
					APIVersion: c.jaeger.APIVersion,
					Kind:       c.jaeger.Kind,
					Name:       c.jaeger.Name,
					UID:        c.jaeger.UID,
					Controller: &trueVar,
				},
			},
		},
		Spec: autoscalingv2beta2.HorizontalPodAutoscalerSpec{
			ScaleTargetRef: autoscalingv2beta2.CrossVersionObjectReference{
				APIVersion: "apps/v1",
				Kind:       "Deployment",
				Name:       c.name(),
			},
			MinReplicas: c.jaeger.Spec.Collector.MinReplicas,
			MaxReplicas: maxReplicas,
			Metrics: []autoscalingv2beta2.MetricSpec{
				{
					Type: autoscalingv2beta2.ResourceMetricSourceType,
					Resource: &autoscalingv2beta2.ResourceMetricSource{
						Name: corev1.ResourceCPU,
						Target: autoscalingv2beta2.MetricTarget{
							Type:               autoscalingv2beta2.UtilizationMetricType,
							AverageUtilization: &avgUtilization,
						},
					},
				},
				{
					Type: autoscalingv2beta2.ResourceMetricSourceType,
					Resource: &autoscalingv2beta2.ResourceMetricSource{
						Name: corev1.ResourceMemory,
						Target: autoscalingv2beta2.MetricTarget{
							Type:               autoscalingv2beta2.UtilizationMetricType,
							AverageUtilization: &avgUtilization,
						},
					},
				},
			},
		},
	}}
}

func (c *Collector) labels() map[string]string {
	return util.Labels(c.name(), "collector", *c.jaeger)
}

func (c *Collector) name() string {
	return fmt.Sprintf("%s-collector", c.jaeger.Name)
}
