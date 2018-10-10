package deployment

import (
	"fmt"

	"github.com/sirupsen/logrus"
	"github.com/spf13/viper"
	appsv1 "k8s.io/api/apps/v1"
	"k8s.io/api/core/v1"
	"k8s.io/api/extensions/v1beta1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"

	"github.com/jaegertracing/jaeger-operator/pkg/apis/io/v1alpha1"
	"github.com/jaegertracing/jaeger-operator/pkg/ingress"
	"github.com/jaegertracing/jaeger-operator/pkg/service"
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

	jaeger.Spec.Agent.Annotations = jaeger.Spec.AllInOne.Annotations
	jaeger.Spec.Collector.Annotations = jaeger.Spec.AllInOne.Annotations
	jaeger.Spec.Query.Annotations = jaeger.Spec.AllInOne.Annotations

	jaeger.Spec.Agent.Labels = jaeger.Spec.AllInOne.Labels
	jaeger.Spec.Collector.Labels = jaeger.Spec.AllInOne.Labels
	jaeger.Spec.Query.Labels = jaeger.Spec.AllInOne.Labels

	return &AllInOne{jaeger: jaeger}
}

// Get returns a pod for the current all-in-one configuration
func (a *AllInOne) Get() *appsv1.Deployment {
	logrus.Debug("Assembling an all-in-one deployment")
	selector := a.selector()
	trueVar := true

	labels := map[string]string{}
	for k, v := range a.jaeger.Spec.AllInOne.Labels {
		labels[k] = v
	}

	// and we append the selectors that we need to be there in a controlled way
	for k, v := range selector {
		labels[k] = v
	}

	annotations := map[string]string{
		"prometheus.io/scrape": "true",
		"prometheus.io/port":   "16686",
	}
	for k, v := range a.jaeger.Spec.AllInOne.Annotations {
		annotations[k] = v
	}

	return &appsv1.Deployment{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "apps/v1",
			Kind:       "Deployment",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:        a.jaeger.Name,
			Namespace:   a.jaeger.Namespace,
			Labels:      labels,
			Annotations: annotations,
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
				MatchLabels: selector,
			},
			Template: v1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels:      labels,
					Annotations: annotations,
				},
				Spec: v1.PodSpec{
					Containers: []v1.Container{{
						Image: a.jaeger.Spec.AllInOne.Image,
						Name:  "jaeger",
						Args:  allArgs(a.jaeger.Spec.AllInOne.Options, a.jaeger.Spec.Storage.Options),
						Env: []v1.EnvVar{
							v1.EnvVar{
								Name:  "SPAN_STORAGE_TYPE",
								Value: a.jaeger.Spec.Storage.Type,
							},
						},
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
							InitialDelaySeconds: 5,
						},
					}},
				},
			},
		},
	}
}

// Services returns a list of services to be deployed along with the all-in-one deployment
func (a *AllInOne) Services() []*v1.Service {
	selector := a.selector()
	return []*v1.Service{
		service.NewCollectorService(a.jaeger, selector),
		service.NewQueryService(a.jaeger, selector),
		service.NewAgentService(a.jaeger, selector),
		service.NewZipkinService(a.jaeger, selector),
	}
}

// Ingresses returns a list of ingress rules to be deployed along with the all-in-one deployment
func (a *AllInOne) Ingresses() []*v1beta1.Ingress {
	if a.jaeger.Spec.AllInOne.Ingress.Enabled == nil || *a.jaeger.Spec.AllInOne.Ingress.Enabled == true {
		return []*v1beta1.Ingress{
			ingress.NewQueryIngress(a.jaeger),
		}
	}

	return []*v1beta1.Ingress{}
}

func (a *AllInOne) selector() map[string]string {
	return map[string]string{"app": "jaeger", "jaeger": a.jaeger.Name}
}
