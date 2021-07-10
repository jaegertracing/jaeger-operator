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
	"github.com/jaegertracing/jaeger-operator/pkg/config/ca"
	"github.com/jaegertracing/jaeger-operator/pkg/util"
)

// Ingester builds pods for jaegertracing/jaeger-ingester
type Ingester struct {
	jaeger *v1.Jaeger
}

// NewIngester builds a new Ingester struct based on the given spec
func NewIngester(jaeger *v1.Jaeger) *Ingester {
	return &Ingester{jaeger: jaeger}
}

// Autoscalers returns a list of HPAs based on this ingester
func (i *Ingester) Autoscalers() []autoscalingv2beta2.HorizontalPodAutoscaler {
	return autoscalers(i)
}

// Get returns a ingester pod
func (i *Ingester) Get() *appsv1.Deployment {
	if i.jaeger.Spec.Strategy != v1.DeploymentStrategyStreaming {
		return nil
	}

	i.jaeger.Logger().Debug("Assembling an ingester deployment")

	labels := i.labels()
	trueVar := true
	falseVar := false

	args := append(i.jaeger.Spec.Ingester.Options.ToArgs())

	adminPort := util.GetAdminPort(args, 14270)

	baseCommonSpec := v1.JaegerCommonSpec{
		Annotations: map[string]string{
			"prometheus.io/scrape":    "true",
			"prometheus.io/port":      strconv.Itoa(int(adminPort)),
			"sidecar.istio.io/inject": "false",
			"linkerd.io/inject":       "disabled",
		},
		Labels: labels,
	}

	commonSpec := util.Merge([]v1.JaegerCommonSpec{i.jaeger.Spec.Ingester.JaegerCommonSpec, i.jaeger.Spec.JaegerCommonSpec, baseCommonSpec})

	var envFromSource []corev1.EnvFromSource
	if len(i.jaeger.Spec.Storage.SecretName) > 0 {
		envFromSource = append(envFromSource, corev1.EnvFromSource{
			SecretRef: &corev1.SecretEnvSource{
				LocalObjectReference: corev1.LocalObjectReference{
					Name: i.jaeger.Spec.Storage.SecretName,
				},
			},
		})
	}

	options := allArgs(i.jaeger.Spec.Ingester.Options,
		i.jaeger.Spec.Storage.Options.Filter(i.jaeger.Spec.Storage.Type.OptionsPrefix()))

	ca.Update(i.jaeger, commonSpec)

	// ensure we have a consistent order of the arguments
	// see https://github.com/jaegertracing/jaeger-operator/issues/334
	sort.Strings(options)

	strategy := appsv1.DeploymentStrategy{
		Type: appsv1.RecreateDeploymentStrategyType,
	}

	if i.jaeger.Spec.Ingester.Strategy != nil {
		strategy = *i.jaeger.Spec.Ingester.Strategy
	}

	return &appsv1.Deployment{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "apps/v1",
			Kind:       "Deployment",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      i.name(),
			Namespace: i.jaeger.Namespace,
			Labels:    commonSpec.Labels,
			OwnerReferences: []metav1.OwnerReference{{
				APIVersion: i.jaeger.APIVersion,
				Kind:       i.jaeger.Kind,
				Name:       i.jaeger.Name,
				UID:        i.jaeger.UID,
				Controller: &trueVar,
			}},
		},
		Spec: appsv1.DeploymentSpec{
			Replicas: i.jaeger.Spec.Ingester.Replicas,
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
					Containers: []corev1.Container{{
						Image: util.ImageName(i.jaeger.Spec.Ingester.Image, "jaeger-ingester-image"),
						Name:  "jaeger-ingester",
						Args:  options,
						Env: []corev1.EnvVar{{
							Name:  "SPAN_STORAGE_TYPE",
							Value: string(i.jaeger.Spec.Storage.Type),
						}},
						VolumeMounts: commonSpec.VolumeMounts,
						EnvFrom:      envFromSource,
						Ports: []corev1.ContainerPort{
							{
								ContainerPort: adminPort,
								Name:          "admin-http",
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
					ServiceAccountName: account.JaegerServiceAccountFor(i.jaeger, account.IngesterComponent),
					Affinity:           commonSpec.Affinity,
					Tolerations:        commonSpec.Tolerations,
					SecurityContext:    commonSpec.SecurityContext,
					EnableServiceLinks: &falseVar,
				},
			},
		},
	}
}

func (i *Ingester) labels() map[string]string {
	return util.Labels(i.name(), "ingester", *i.jaeger)
}

func (i *Ingester) hpaLabels() map[string]string {
	labels := i.labels()
	labels["app.kubernetes.io/component"] = "hpa-ingester"
	return labels
}

func (i *Ingester) name() string {
	return fmt.Sprintf("%s-ingester", i.jaeger.Name)
}

func (i *Ingester) commonSpec() v1.JaegerCommonSpec {
	return i.jaeger.Spec.Ingester.JaegerCommonSpec
}

func (i *Ingester) autoscalingSpec() v1.AutoScaleSpec {
	return i.jaeger.Spec.Ingester.AutoScaleSpec
}

func (i *Ingester) jaegerInstance() *v1.Jaeger {
	return i.jaeger
}

func (i *Ingester) replicas() *int32 {
	return i.jaeger.Spec.Ingester.Replicas
}
