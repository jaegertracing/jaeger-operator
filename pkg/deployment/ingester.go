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
	"github.com/jaegertracing/jaeger-operator/pkg/service"
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

	selector := i.selector()
	trueVar := true
	replicas := int32(i.jaeger.Spec.Ingester.Size)

	baseCommonSpec := v1alpha1.JaegerCommonSpec{
		Annotations: map[string]string{
			"prometheus.io/scrape":    "true",
			"prometheus.io/port":      "2345",
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
			Name:      fmt.Sprintf("%s-ingester", i.jaeger.Name),
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
				MatchLabels: selector,
			},
			Template: v1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels:      selector,
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
								ContainerPort: 2345,
								Name:          "ingester-http",
							},
						},
						ReadinessProbe: &v1.Probe{
							Handler: v1.Handler{
								HTTPGet: &v1.HTTPGetAction{
									Path: "/",
									Port: intstr.FromInt(2345),
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

// Services returns a list of services to be deployed along with the ingesterdeployment
func (i *Ingester) Services() []*v1.Service {
	services := []*v1.Service{}

	service := service.NewIngesterService(i.jaeger, i.selector())

	if service != nil {
		services = append(services, service)
	}

	return services
}

func (i *Ingester) selector() map[string]string {
	return map[string]string{"app": "jaeger", "jaeger": i.jaeger.Name, "jaeger-component": "ingester"}
}
