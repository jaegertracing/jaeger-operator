package deployment

import (
	"fmt"
	"sort"
	"strings"

	"github.com/spf13/viper"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"

	"github.com/jaegertracing/jaeger-operator/pkg/apis/jaegertracing/v1"
	"github.com/jaegertracing/jaeger-operator/pkg/service"
	"github.com/jaegertracing/jaeger-operator/pkg/util"
)

// Agent builds pods for jaegertracing/jaeger-agent
type Agent struct {
	jaeger *v1.Jaeger
}

// NewAgent builds a new Agent struct based on the given spec
func NewAgent(jaeger *v1.Jaeger) *Agent {
	if jaeger.Spec.Agent.Image == "" {
		jaeger.Spec.Agent.Image = fmt.Sprintf("%s:%s", viper.GetString("jaeger-agent-image"), viper.GetString("jaeger-version"))
	}

	return &Agent{jaeger: jaeger}
}

// Get returns a Agent pod
func (a *Agent) Get() *appsv1.DaemonSet {
	if !strings.EqualFold(a.jaeger.Spec.Agent.Strategy, "daemonset") {
		a.jaeger.Logger().WithField("strategy", a.jaeger.Spec.Agent.Strategy).Debug("skipping agent daemonset")
		return nil
	}

	args := append(a.jaeger.Spec.Agent.Options.ToArgs(),
		"--reporter.type=grpc",
		fmt.Sprintf("--reporter.grpc.host-port=%s:14250", service.GetNameForCollectorService(a.jaeger)))
	trueVar := true
	labels := a.labels()

	baseCommonSpec := v1.JaegerCommonSpec{
		Annotations: map[string]string{
			"prometheus.io/scrape":    "true",
			"prometheus.io/port":      "5778",
			"sidecar.istio.io/inject": "false",
		},
	}

	commonSpec := util.Merge([]v1.JaegerCommonSpec{a.jaeger.Spec.Agent.JaegerCommonSpec, a.jaeger.Spec.JaegerCommonSpec, baseCommonSpec})

	// ensure we have a consistent order of the arguments
	// see https://github.com/jaegertracing/jaeger-operator/issues/334
	sort.Strings(args)

	return &appsv1.DaemonSet{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "apps/v1",
			Kind:       "DaemonSet",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      fmt.Sprintf("%s-agent-daemonset", a.jaeger.Name),
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
		Spec: appsv1.DaemonSetSpec{
			Selector: &metav1.LabelSelector{
				MatchLabels: labels,
			},
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels:      labels,
					Annotations: commonSpec.Annotations,
				},
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{{
						Image: a.jaeger.Spec.Agent.Image,
						Name:  "jaeger-agent-daemonset",
						Args:  args,
						Ports: []corev1.ContainerPort{
							{
								ContainerPort: 5775,
								HostPort:      5775,
								Name:          "zk-compact-trft",
								Protocol:      corev1.ProtocolUDP,
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
								Protocol:      corev1.ProtocolUDP,
							},
							{
								ContainerPort: 6832,
								HostPort:      6832,
								Name:          "jg-binary-trft",
								Protocol:      corev1.ProtocolUDP,
							},
						},
						ReadinessProbe: &corev1.Probe{
							Handler: corev1.Handler{
								HTTPGet: &corev1.HTTPGetAction{
									Path: "/metrics",
									Port: intstr.FromInt(5778),
								},
							},
							InitialDelaySeconds: 1,
						},
						Resources: commonSpec.Resources,
					}},
				},
			},
		},
	}
}

func (a *Agent) labels() map[string]string {
	return map[string]string{
		"app":                          "jaeger", // TODO(jpkroehling): see collector.go in this package
		"app.kubernetes.io/name":       a.name(),
		"app.kubernetes.io/instance":   a.jaeger.Name,
		"app.kubernetes.io/component":  "agent",
		"app.kubernetes.io/part-of":    "jaeger",
		"app.kubernetes.io/managed-by": "jaeger-operator",
	}
}

func (a *Agent) name() string {
	return fmt.Sprintf("%s-agent", a.jaeger.Name)
}
