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
	"github.com/jaegertracing/jaeger-operator/pkg/storage"
	"github.com/jaegertracing/jaeger-operator/pkg/util"
)

// Ingester builds pods for jaegertracing/jaeger-ingester
type Ingester struct {
	jaeger *v1.Jaeger
}

// NewIngester builds a new Ingester struct based on the given spec
func NewIngester(jaeger *v1.Jaeger) *Ingester {
	if jaeger.Spec.Ingester.Replicas == nil || *jaeger.Spec.Ingester.Replicas < 0 {
		replicaSize := int32(1)
		if jaeger.Spec.Ingester.Size > 0 {
			jaeger.Logger().Warn("The 'size' property for the ingester is deprecated. Use 'replicas' instead.")
			replicaSize = int32(jaeger.Spec.Ingester.Size)
		}

		jaeger.Spec.Ingester.Replicas = &replicaSize
	}

	if jaeger.Spec.Ingester.Image == "" {
		jaeger.Spec.Ingester.Image = fmt.Sprintf("%s:%s", viper.GetString("jaeger-ingester-image"), viper.GetString("jaeger-version"))
	}

	return &Ingester{jaeger: jaeger}
}

// Get returns a ingester pod
func (i *Ingester) Get() *appsv1.Deployment {
	if !strings.EqualFold(i.jaeger.Spec.Strategy, "streaming") {
		return nil
	}

	i.jaeger.Logger().Debug("Assembling an ingester deployment")

	labels := i.labels()
	trueVar := true

	args := append(i.jaeger.Spec.Ingester.Options.ToArgs())

	adminPort := util.GetPort("--admin-http-port=", args, 14270)

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
		i.jaeger.Spec.Storage.Options.Filter(storage.OptionsPrefix(i.jaeger.Spec.Storage.Type)))

	// ensure we have a consistent order of the arguments
	// see https://github.com/jaegertracing/jaeger-operator/issues/334
	sort.Strings(options)

	return &appsv1.Deployment{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "apps/v1",
			Kind:       "Deployment",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      i.name(),
			Namespace: i.jaeger.Namespace,
			Labels:    commonSpec.Labels,
			OwnerReferences: []metav1.OwnerReference{
				metav1.OwnerReference{
					APIVersion: i.jaeger.APIVersion,
					Kind:       i.jaeger.Kind,
					Name:       i.jaeger.Name,
					UID:        i.jaeger.UID,
					Controller: &trueVar,
				},
			},
		},
		Spec: appsv1.DeploymentSpec{
			Replicas: i.jaeger.Spec.Ingester.Replicas,
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
						Image: i.jaeger.Spec.Ingester.Image,
						Name:  "jaeger-ingester",
						Args:  options,
						Env: []corev1.EnvVar{
							corev1.EnvVar{
								Name:  "SPAN_STORAGE_TYPE",
								Value: i.jaeger.Spec.Storage.Type,
							},
						},
						VolumeMounts: commonSpec.VolumeMounts,
						EnvFrom:      envFromSource,
						Ports: []corev1.ContainerPort{
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
					ServiceAccountName: account.JaegerServiceAccountFor(i.jaeger, account.IngesterComponent),
					Affinity:           commonSpec.Affinity,
					Tolerations:        commonSpec.Tolerations,
					SecurityContext:    commonSpec.SecurityContext,
				},
			},
		},
	}
}

func (i *Ingester) labels() map[string]string {
	return map[string]string{
		"app":                          "jaeger", // TODO(jpkroehling): see collector.go in this package
		"app.kubernetes.io/name":       i.name(),
		"app.kubernetes.io/instance":   i.jaeger.Name,
		"app.kubernetes.io/component":  "ingester",
		"app.kubernetes.io/part-of":    "jaeger",
		"app.kubernetes.io/managed-by": "jaeger-operator",
	}
}

func (i *Ingester) name() string {
	return fmt.Sprintf("%s-ingester", i.jaeger.Name)
}
