package deployment

import (
	"fmt"
	"strings"

	"github.com/sirupsen/logrus"
	"github.com/spf13/viper"
	appsv1 "k8s.io/api/apps/v1"
	"k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"

	"github.com/jaegertracing/jaeger-operator/pkg/apis/io/v1alpha1"
	"github.com/jaegertracing/jaeger-operator/pkg/storage"
	"github.com/jaegertracing/jaeger-operator/pkg/util"
)

// Ingester builds pods for jaegertracing/jaeger-ingester
type Ingester struct {
	jaeger *v1alpha1.Jaeger
}

// NewIngester builds a new Ingester struct based on the given spec
func NewIngester(jaeger *v1alpha1.Jaeger) *Ingester {
	if jaeger.Spec.Ingester.Size <= 0 {
		jaeger.Spec.Ingester.Size = 1
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

	logrus.Debugf("Assembling a ingester deployment for %v", i.jaeger)

	labels := i.labels()
	trueVar := true
	replicas := int32(i.jaeger.Spec.Ingester.Size)

	baseCommonSpec := v1alpha1.JaegerCommonSpec{
		Annotations: map[string]string{
			"prometheus.io/scrape":    "true",
			"prometheus.io/port":      "14271",
			"sidecar.istio.io/inject": "false",
		},
	}

	commonSpec := util.Merge([]v1alpha1.JaegerCommonSpec{i.jaeger.Spec.Ingester.JaegerCommonSpec, i.jaeger.Spec.JaegerCommonSpec, baseCommonSpec})

	var envFromSource []v1.EnvFromSource
	if len(i.jaeger.Spec.Storage.SecretName) > 0 {
		envFromSource = append(envFromSource, v1.EnvFromSource{
			SecretRef: &v1.SecretEnvSource{
				LocalObjectReference: v1.LocalObjectReference{
					Name: i.jaeger.Spec.Storage.SecretName,
				},
			},
		})
	}

	options := allArgs(i.jaeger.Spec.Ingester.Options,
		i.jaeger.Spec.Storage.Options.Filter(storage.OptionsPrefix(i.jaeger.Spec.Storage.Type)),
		i.jaeger.Spec.Storage.Options.Filter("kafka"))

	return &appsv1.Deployment{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "apps/v1",
			Kind:       "Deployment",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      i.name(),
			Namespace: i.jaeger.Namespace,
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
			Replicas: &replicas,
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
						Image: i.jaeger.Spec.Ingester.Image,
						Name:  "jaeger-ingester",
						Args:  options,
						Env: []v1.EnvVar{
							v1.EnvVar{
								Name:  "SPAN_STORAGE_TYPE",
								Value: i.jaeger.Spec.Storage.Type,
							},
						},
						VolumeMounts: commonSpec.VolumeMounts,
						EnvFrom:      envFromSource,
						Ports: []v1.ContainerPort{
							{
								ContainerPort: 14270,
								Name:          "hc-http",
							},
							{
								ContainerPort: 14271,
								Name:          "metrics-http",
							},
						},
						ReadinessProbe: &v1.Probe{
							Handler: v1.Handler{
								HTTPGet: &v1.HTTPGetAction{
									Path: "/",
									Port: intstr.FromInt(14270),
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

// Services returns a list of services to be deployed along with the ingester deployment
func (i *Ingester) Services() []*v1.Service {
	// Return empty list, as a service is not required for this deployment, which also
	// simplifies switching between different strategies.
	return []*v1.Service{}
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
