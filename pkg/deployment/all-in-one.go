package deployment

import (
	"fmt"

	"github.com/sirupsen/logrus"
	"github.com/spf13/viper"
	appsv1 "k8s.io/api/apps/v1"
	"k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"

	"github.com/jaegertracing/jaeger-operator/pkg/apis/io/v1alpha1"
	"github.com/jaegertracing/jaeger-operator/pkg/config/sampling"
	"github.com/jaegertracing/jaeger-operator/pkg/config/ui"
	"github.com/jaegertracing/jaeger-operator/pkg/service"
	"github.com/jaegertracing/jaeger-operator/pkg/storage"
	"github.com/jaegertracing/jaeger-operator/pkg/util"
)

// AllInOne builds pods for jaegertracing/all-in-one
type AllInOne struct {
	jaeger *v1alpha1.Jaeger
}

// NewAllInOne builds a new AllInOne struct based on the given spec
func NewAllInOne(jaeger *v1alpha1.Jaeger) *AllInOne {
	if jaeger.Spec.AllInOne.Image == "" {
		jaeger.Spec.AllInOne.Image = fmt.Sprintf("%s:%s", viper.GetString("jaeger-all-in-one-image"), viper.GetString("jaeger-version"))
	}

	return &AllInOne{jaeger: jaeger}
}

// Get returns a pod for the current all-in-one configuration
func (a *AllInOne) Get() *appsv1.Deployment {
	logrus.Debug("Assembling an all-in-one deployment")
	labels := a.labels()
	trueVar := true

	baseCommonSpec := v1alpha1.JaegerCommonSpec{
		Annotations: map[string]string{
			"prometheus.io/scrape":    "true",
			"prometheus.io/port":      "16686",
			"sidecar.istio.io/inject": "false",
		},
	}

	commonSpec := util.Merge([]v1alpha1.JaegerCommonSpec{a.jaeger.Spec.AllInOne.JaegerCommonSpec, a.jaeger.Spec.JaegerCommonSpec, baseCommonSpec})

	options := allArgs(a.jaeger.Spec.AllInOne.Options,
		a.jaeger.Spec.Storage.Options.Filter(storage.OptionsPrefix(a.jaeger.Spec.Storage.Type)))

	configmap.Update(a.jaeger, commonSpec, &options)
	sampling.Update(a.jaeger, commonSpec, &options)

	var envFromSource []v1.EnvFromSource
	if len(a.jaeger.Spec.Storage.SecretName) > 0 {
		envFromSource = append(envFromSource, v1.EnvFromSource{
			SecretRef: &v1.SecretEnvSource{
				LocalObjectReference: v1.LocalObjectReference{
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
			Name:      a.jaeger.Name,
			Namespace: a.jaeger.Namespace,
			Labels:    labels,
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
			Template: v1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels:      labels,
					Annotations: commonSpec.Annotations,
				},
				Spec: v1.PodSpec{
					Containers: []v1.Container{{
						Image: a.jaeger.Spec.AllInOne.Image,
						Name:  "jaeger",
						Args:  options,
						Env: []v1.EnvVar{
							v1.EnvVar{
								Name:  "SPAN_STORAGE_TYPE",
								Value: a.jaeger.Spec.Storage.Type,
							},
							v1.EnvVar{
								Name:  "COLLECTOR_ZIPKIN_HTTP_PORT",
								Value: "9411",
							},
						},
						VolumeMounts: commonSpec.VolumeMounts,
						EnvFrom:      envFromSource,
						Ports: []v1.ContainerPort{
							{
								ContainerPort: 5775,
								Name:          "zk-compact-trft", // max 15 chars!
								Protocol:      v1.ProtocolUDP,
							},
							{
								ContainerPort: 5778,
								Name:          "config-rest",
							},
							{
								ContainerPort: 6831,
								Name:          "jg-compact-trft",
								Protocol:      v1.ProtocolUDP,
							},
							{
								ContainerPort: 6832,
								Name:          "jg-binary-trft",
								Protocol:      v1.ProtocolUDP,
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
						},
						ReadinessProbe: &v1.Probe{
							Handler: v1.Handler{
								HTTPGet: &v1.HTTPGetAction{
									Path: "/",
									Port: intstr.FromInt(14269),
								},
							},
							InitialDelaySeconds: 1,
						},
						Resources: commonSpec.Resources,
					}},
					Volumes: commonSpec.Volumes,
				},
			},
		},
	}
}

// Services returns a list of services to be deployed along with the all-in-one deployment
func (a *AllInOne) Services() []*v1.Service {
	labels := a.labels()
	return []*v1.Service{
		service.NewCollectorService(a.jaeger, labels),
		service.NewQueryService(a.jaeger, labels),
		service.NewAgentService(a.jaeger, labels),
	}
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
