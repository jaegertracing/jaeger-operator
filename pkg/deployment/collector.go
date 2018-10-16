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
	"github.com/jaegertracing/jaeger-operator/pkg/service"
)

// Collector builds pods for jaegertracing/jaeger-collector
type Collector struct {
	jaeger *v1alpha1.Jaeger
}

// NewCollector builds a new Collector struct based on the given spec
func NewCollector(jaeger *v1alpha1.Jaeger) *Collector {
	if jaeger.Spec.Collector.Size <= 0 {
		jaeger.Spec.Collector.Size = 1
	}

	if jaeger.Spec.Collector.Image == "" {
		jaeger.Spec.Collector.Image = fmt.Sprintf("%s:%s", viper.GetString("jaeger-collector-image"), viper.GetString("jaeger-version"))
	}

	return &Collector{jaeger: jaeger}
}

// Get returns a collector pod
func (c *Collector) Get() *appsv1.Deployment {
	logrus.Debugf("Assembling a collector deployment for %v", c.jaeger)

	selector := c.selector()
	trueVar := true
	replicas := int32(c.jaeger.Spec.Collector.Size)
	annotations := map[string]string{
		"prometheus.io/scrape": "true",
		"prometheus.io/port":   "14268",
	}

	return &appsv1.Deployment{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "apps/v1",
			Kind:       "Deployment",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      fmt.Sprintf("%s-collector", c.jaeger.Name),
			Namespace: c.jaeger.Namespace,
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
			Replicas: &replicas,
			Selector: &metav1.LabelSelector{
				MatchLabels: selector,
			},
			Template: v1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels:      selector,
					Annotations: annotations,
				},
				Spec: v1.PodSpec{
					Containers: []v1.Container{{
						Image: c.jaeger.Spec.Collector.Image,
						Name:  "jaeger-collector",
						Args:  allArgs(c.jaeger.Spec.Collector.Options, c.jaeger.Spec.Storage.Options),
						Env: []v1.EnvVar{
							v1.EnvVar{
								Name:  "SPAN_STORAGE_TYPE",
								Value: c.jaeger.Spec.Storage.Type,
							},
							v1.EnvVar{
								Name:  "COLLECTOR_ZIPKIN_HTTP_PORT",
								Value: "9411",
							},
						},
						Ports: []v1.ContainerPort{
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
					}},
				},
			},
		},
	}
}

// Services returns a list of services to be deployed along with the all-in-one deployment
func (c *Collector) Services() []*v1.Service {
	selector := c.selector()
	return []*v1.Service{
		service.NewCollectorService(c.jaeger, selector),
		service.NewZipkinService(c.jaeger, selector),
	}
}

func (c *Collector) selector() map[string]string {
	return map[string]string{"app": "jaeger", "jaeger": c.jaeger.Name, "jaeger-component": "collector"}
}
