package deployment

import (
	"fmt"
	"sort"
	"strconv"

	"github.com/spf13/viper"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"

	"github.com/jaegertracing/jaeger-operator/pkg/account"
	v1 "github.com/jaegertracing/jaeger-operator/pkg/apis/jaegertracing/v1"
	"github.com/jaegertracing/jaeger-operator/pkg/config/ca"
	"github.com/jaegertracing/jaeger-operator/pkg/config/sampling"
	"github.com/jaegertracing/jaeger-operator/pkg/config/tls"
	configmap "github.com/jaegertracing/jaeger-operator/pkg/config/ui"
	"github.com/jaegertracing/jaeger-operator/pkg/service"
	"github.com/jaegertracing/jaeger-operator/pkg/util"
)

// AllInOne builds pods for jaegertracing/all-in-one
type AllInOne struct {
	jaeger *v1.Jaeger
}

// NewAllInOne builds a new AllInOne struct based on the given spec
func NewAllInOne(jaeger *v1.Jaeger) *AllInOne {
	return &AllInOne{jaeger: jaeger}
}

// Get returns a pod for the current all-in-one configuration
func (a *AllInOne) Get() *appsv1.Deployment {
	a.jaeger.Logger().Debug("Assembling an all-in-one deployment")
	trueVar := true
	falseVar := false

	args := append(a.jaeger.Spec.AllInOne.Options.ToArgs())

	adminPort := util.GetAdminPort(args, 14269)

	jaegerDisabled := false
	if a.jaeger.Spec.AllInOne.TracingEnabled != nil && *a.jaeger.Spec.AllInOne.TracingEnabled == false {
		jaegerDisabled = true
	}

	baseCommonSpec := v1.JaegerCommonSpec{
		Annotations: map[string]string{
			"prometheus.io/scrape":    "true",
			"prometheus.io/port":      strconv.Itoa(int(adminPort)),
			"sidecar.istio.io/inject": "false",
			"linkerd.io/inject":       "disabled",
		},
		Labels: a.labels(),
	}

	commonSpec := util.Merge([]v1.JaegerCommonSpec{a.jaeger.Spec.AllInOne.JaegerCommonSpec, a.jaeger.Spec.JaegerCommonSpec, baseCommonSpec})

	options := allArgs(a.jaeger.Spec.AllInOne.Options,
		a.jaeger.Spec.Storage.Options.Filter(a.jaeger.Spec.Storage.Type.OptionsPrefix()))

	configmap.Update(a.jaeger, commonSpec, &options)
	sampling.Update(a.jaeger, commonSpec, &options)
	tls.Update(a.jaeger, commonSpec, &options)
	ca.Update(a.jaeger, commonSpec)
	ca.AddServiceCA(a.jaeger, commonSpec)

	// Enable tls by default for openshift platform
	// even though the agent is in the same process as the collector, they communicate via gRPC, and the collector has TLS enabled,
	// as it might receive connections from external agents
	if viper.GetString("platform") == v1.FlagPlatformOpenShift {
		if len(util.FindItem("--reporter.grpc.tls.enabled=true", options)) == 0 {
			options = append(options, "--reporter.grpc.tls.enabled=true")
			options = append(options, fmt.Sprintf("--reporter.grpc.tls.ca=%s", ca.ServiceCAPath))
			options = append(options, fmt.Sprintf("--reporter.grpc.tls.server-name=%s.%s.svc.cluster.local", service.GetNameForHeadlessCollectorService(a.jaeger), a.jaeger.Namespace))
		}
	}

	// ensure we have a consistent order of the arguments
	// see https://github.com/jaegertracing/jaeger-operator/issues/334
	sort.Strings(options)

	var envFromSource []corev1.EnvFromSource
	if len(a.jaeger.Spec.Storage.SecretName) > 0 {
		envFromSource = append(envFromSource, corev1.EnvFromSource{
			SecretRef: &corev1.SecretEnvSource{
				LocalObjectReference: corev1.LocalObjectReference{
					Name: a.jaeger.Spec.Storage.SecretName,
				},
			},
		})
	}

	strategy := appsv1.DeploymentStrategy{
		Type: appsv1.RecreateDeploymentStrategyType,
	}

	if a.jaeger.Spec.AllInOne.Strategy != nil {
		strategy = *a.jaeger.Spec.AllInOne.Strategy
	}

	return &appsv1.Deployment{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "apps/v1",
			Kind:       "Deployment",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:        a.jaeger.Name,
			Namespace:   a.jaeger.Namespace,
			Labels:      commonSpec.Labels,
			Annotations: commonSpec.Annotations,
			OwnerReferences: []metav1.OwnerReference{{
				APIVersion: a.jaeger.APIVersion,
				Kind:       a.jaeger.Kind,
				Name:       a.jaeger.Name,
				UID:        a.jaeger.UID,
				Controller: &trueVar,
			}},
		},
		Spec: appsv1.DeploymentSpec{
			Selector: &metav1.LabelSelector{
				MatchLabels: commonSpec.Labels,
			},
			Strategy: strategy,
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels:      commonSpec.Labels,
					Annotations: commonSpec.Annotations,
				},
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{{
						Image: util.ImageName(a.jaeger.Spec.AllInOne.Image, "jaeger-all-in-one-image"),
						Name:  "jaeger",
						Args:  options,
						Env: []corev1.EnvVar{
							{
								Name:  "SPAN_STORAGE_TYPE",
								Value: string(a.jaeger.Spec.Storage.Type),
							},
							{
								Name:  "COLLECTOR_ZIPKIN_HOST_PORT",
								Value: ":9411",
							},
							{
								Name:  "JAEGER_DISABLED",
								Value: strconv.FormatBool(jaegerDisabled),
							},
						},
						VolumeMounts: commonSpec.VolumeMounts,
						EnvFrom:      envFromSource,
						Ports: []corev1.ContainerPort{
							{
								ContainerPort: 5775,
								Name:          "zk-compact-trft", // max 15 chars!
								Protocol:      corev1.ProtocolUDP,
							},
							{
								ContainerPort: 5778,
								Name:          "config-rest",
							},
							{
								ContainerPort: 6831,
								Name:          "jg-compact-trft",
								Protocol:      corev1.ProtocolUDP,
							},
							{
								ContainerPort: 6832,
								Name:          "jg-binary-trft",
								Protocol:      corev1.ProtocolUDP,
							},
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
								ContainerPort: 16686,
								Name:          "query",
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
					ServiceAccountName: account.JaegerServiceAccountFor(a.jaeger, account.AllInOneComponent),
					Affinity:           commonSpec.Affinity,
					Tolerations:        commonSpec.Tolerations,
					SecurityContext:    commonSpec.SecurityContext,
					EnableServiceLinks: &falseVar,
				},
			},
		},
	}
}

// Services returns a list of services to be deployed along with the all-in-one deployment
func (a *AllInOne) Services() []*corev1.Service {
	// merge defined labels with default labels
	spec := util.Merge([]v1.JaegerCommonSpec{a.jaeger.Spec.AllInOne.JaegerCommonSpec, a.jaeger.Spec.JaegerCommonSpec, v1.JaegerCommonSpec{Labels: a.labels()}})
	labels := spec.Labels

	return append(service.NewCollectorServices(a.jaeger, labels),
		service.NewQueryService(a.jaeger, labels),
		service.NewAgentService(a.jaeger, labels),
	)
}

func (a *AllInOne) labels() map[string]string {
	return util.Labels(a.name(), "all-in-one", *a.jaeger)
}

func (a *AllInOne) name() string {
	return a.jaeger.Name
}
