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
	"github.com/jaegertracing/jaeger-operator/pkg/apis/jaegertracing/v1"
	"github.com/jaegertracing/jaeger-operator/pkg/config/sampling"
	"github.com/jaegertracing/jaeger-operator/pkg/config/ui"
	"github.com/jaegertracing/jaeger-operator/pkg/service"
	"github.com/jaegertracing/jaeger-operator/pkg/storage"
	"github.com/jaegertracing/jaeger-operator/pkg/util"
)

// AllInOne builds pods for jaegertracing/all-in-one
type AllInOne struct {
	jaeger *v1.Jaeger
}

// NewAllInOne builds a new AllInOne struct based on the given spec
func NewAllInOne(jaeger *v1.Jaeger) *AllInOne {
	if jaeger.Spec.AllInOne.Image == "" {
		jaeger.Spec.AllInOne.Image = fmt.Sprintf("%s:%s", viper.GetString("jaeger-all-in-one-image"), viper.GetString("jaeger-version"))
	}

	return &AllInOne{jaeger: jaeger}
}

// Get returns a pod for the current all-in-one configuration
func (a *AllInOne) Get() *appsv1.Deployment {
	a.jaeger.Logger().Debug("Assembling an all-in-one deployment")
	labels := a.labels()
	trueVar := true

	args := append(a.jaeger.Spec.AllInOne.Options.ToArgs())

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

	commonSpec := util.Merge([]v1.JaegerCommonSpec{a.jaeger.Spec.AllInOne.JaegerCommonSpec, a.jaeger.Spec.JaegerCommonSpec, baseCommonSpec})

	options := allArgs(a.jaeger.Spec.AllInOne.Options,
		a.jaeger.Spec.Storage.Options.Filter(storage.OptionsPrefix(a.jaeger.Spec.Storage.Type)))

	configmap.Update(a.jaeger, commonSpec, &options)
	sampling.Update(a.jaeger, commonSpec, &options)

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
			OwnerReferences: []metav1.OwnerReference{
				metav1.OwnerReference{
					APIVersion: a.jaeger.APIVersion,
					Kind:       a.jaeger.Kind,
					Name:       a.jaeger.Name,
					UID:        a.jaeger.UID,
					Controller: &trueVar,
				},
			},
		},
		Spec: appsv1.DeploymentSpec{
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
						Image: a.jaeger.Spec.AllInOne.Image,
						Name:  "jaeger",
						Args:  options,
						Env: []corev1.EnvVar{
							corev1.EnvVar{
								Name:  "SPAN_STORAGE_TYPE",
								Value: a.jaeger.Spec.Storage.Type,
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
				},
			},
		},
	}
}

// Services returns a list of services to be deployed along with the all-in-one deployment
func (a *AllInOne) Services() []*corev1.Service {
	labels := a.labels()
	return append(service.NewCollectorServices(a.jaeger, labels),
		service.NewQueryService(a.jaeger, labels),
		service.NewAgentService(a.jaeger, labels),
	)
}

func (a *AllInOne) labels() map[string]string {
	return map[string]string{
		"app":                          "jaeger", // TODO(jpkroehling): see collector.go in this package
		"app.kubernetes.io/name":       a.name(),
		"app.kubernetes.io/instance":   a.jaeger.Name,
		"app.kubernetes.io/component":  "all-in-one",
		"app.kubernetes.io/part-of":    "jaeger",
		"app.kubernetes.io/managed-by": "jaeger-operator",
	}
}

func (a *AllInOne) name() string {
	return a.jaeger.Name
}
