package deployment

import (
	"fmt"
	"sort"
	"strconv"
	"strings"

	"github.com/spf13/viper"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"

	"github.com/jaegertracing/jaeger-operator/pkg/account"
	"github.com/jaegertracing/jaeger-operator/pkg/apis/jaegertracing/v1"
	"github.com/jaegertracing/jaeger-operator/pkg/config/sampling"
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
	if jaeger.Spec.Collector.Replicas == nil || *jaeger.Spec.Collector.Replicas < 0 {
		replicaSize := int32(1)
		if jaeger.Spec.Collector.Size > 0 {
			jaeger.Logger().Warn("The 'size' property for the collector is deprecated. Use 'replicas' instead.")
			replicaSize = int32(jaeger.Spec.Collector.Size)
		}

		jaeger.Spec.Collector.Replicas = &replicaSize
	}

	if jaeger.Spec.Collector.Image == "" {
		jaeger.Spec.Collector.Image = fmt.Sprintf("%s:%s", viper.GetString("jaeger-collector-image"), viper.GetString("jaeger-version"))
	}

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
	// If strategy is "streaming", then change storage type
	// to Kafka, and the storage options will be used in the Ingester instead
	if strings.EqualFold(c.jaeger.Spec.Strategy, "streaming") {
		storageType = "kafka"
	}
	options := allArgs(c.jaeger.Spec.Collector.Options,
		c.jaeger.Spec.Storage.Options.Filter(storage.OptionsPrefix(storageType)))

	sampling.Update(c.jaeger, commonSpec, &options)

	// ensure we have a consistent order of the arguments
	// see https://github.com/jaegertracing/jaeger-operator/issues/334
	sort.Strings(options)

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
			Replicas: c.jaeger.Spec.Collector.Replicas,
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
						Image: c.jaeger.Spec.Collector.Image,
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

func (c *Collector) labels() map[string]string {
	return map[string]string{
		"app":                          "jaeger", // kept for backwards compatibility, remove by version 2.0
		"app.kubernetes.io/name":       c.name(),
		"app.kubernetes.io/instance":   c.jaeger.Name,
		"app.kubernetes.io/component":  "collector",
		"app.kubernetes.io/part-of":    "jaeger",
		"app.kubernetes.io/managed-by": "jaeger-operator", // should we qualify this with the operator's namespace?

		// the 'version' label is out for now for two reasons:
		// 1. https://github.com/jaegertracing/jaeger-operator/issues/166
		// 2. these labels are also used as selectors, and as such, need to be consistent... this
		// might be a problem once we support updating the jaeger version
	}
}

func (c *Collector) name() string {
	return fmt.Sprintf("%s-collector", c.jaeger.Name)
}
