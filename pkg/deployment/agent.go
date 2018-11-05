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
)

// Agent builds pods for jaegertracing/jaeger-agent
type Agent struct {
	jaeger *v1alpha1.Jaeger
}

// NewAgent builds a new Agent struct based on the given spec
func NewAgent(jaeger *v1alpha1.Jaeger) *Agent {
	if jaeger.Spec.Agent.Image == "" {
		jaeger.Spec.Agent.Image = fmt.Sprintf("%s:%s", viper.GetString("jaeger-agent-image"), viper.GetString("jaeger-version"))
	}

	return &Agent{jaeger: jaeger}
}

// Get returns a Agent pod
func (a *Agent) Get() *appsv1.DaemonSet {
	if strings.ToLower(a.jaeger.Spec.Agent.Strategy) != "daemonset" {
		logrus.Debugf(
			"The Jaeger instance '%v' is using a Sidecar strategy for the Jaeger Agent. Skipping its DaemonSet deployment.",
			a.jaeger.Name,
		)
		return nil
	}

	args := append(a.jaeger.Spec.Agent.Options.ToArgs(), fmt.Sprintf("--collector.host-port=%s:14267", service.GetNameForCollectorService(a.jaeger)))
	trueVar := true
	selector := a.selector()
	annotations := map[string]string{
		"prometheus.io/scrape":    "true",
		"prometheus.io/port":      "5778",
		"sidecar.istio.io/inject": "false",
	}
	for k, v := range a.jaeger.Spec.Agent.Annotations {
		annotations[k] = v
	}

	return &appsv1.DaemonSet{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "apps/v1",
			Kind:       "DaemonSet",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      fmt.Sprintf("%s-agent-daemonset", a.jaeger.Name),
			Namespace: a.jaeger.Namespace,
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
		Spec: appsv1.DaemonSetSpec{
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
						Image: a.jaeger.Spec.Agent.Image,
						Name:  "jaeger-agent-daemonset",
						Args:  args,
						Ports: []v1.ContainerPort{
							{
								ContainerPort: 5775,
								HostPort:      5775,
								Name:          "zk-compact-trft",
								Protocol:      v1.ProtocolUDP,
							},
							{
								ContainerPort: 5778,
								HostPort:      5778,
								Name:          "config-rest",
							},
							{
								ContainerPort: 6831,
								HostPort:      6831,
								Name:          "jg-compact-trft",
								Protocol:      v1.ProtocolUDP,
							},
							{
								ContainerPort: 6832,
								HostPort:      6832,
								Name:          "jg-binary-trft",
								Protocol:      v1.ProtocolUDP,
							},
						},
						ReadinessProbe: &v1.Probe{
							Handler: v1.Handler{
								HTTPGet: &v1.HTTPGetAction{
									Path: "/metrics",
									Port: intstr.FromInt(5778),
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

func (a *Agent) selector() map[string]string {
	return map[string]string{"app": "jaeger", "jaeger": a.jaeger.Name, "jaeger-component": "agent-daemonset"}
}
