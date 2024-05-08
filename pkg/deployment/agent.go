package deployment

import (
	"fmt"
	"sort"
	"strconv"
	"strings"

	"github.com/operator-framework/operator-lib/proxy"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"

	v1 "github.com/jaegertracing/jaeger-operator/apis/v1"
	"github.com/jaegertracing/jaeger-operator/pkg/account"
	"github.com/jaegertracing/jaeger-operator/pkg/autodetect"
	"github.com/jaegertracing/jaeger-operator/pkg/config/ca"
	"github.com/jaegertracing/jaeger-operator/pkg/service"
	"github.com/jaegertracing/jaeger-operator/pkg/util"
)

// Agent builds pods for jaegertracing/jaeger-agent
type Agent struct {
	jaeger *v1.Jaeger
}

// NewAgent builds a new Agent struct based on the given spec
func NewAgent(jaeger *v1.Jaeger) *Agent {
	return &Agent{jaeger: jaeger}
}

// Get returns a Agent pod
func (a *Agent) Get() *appsv1.DaemonSet {
	if !strings.EqualFold(a.jaeger.Spec.Agent.Strategy, "daemonset") {
		a.jaeger.Logger().V(-1).Info(
			"skipping agent daemonset",
			"strategy", a.jaeger.Spec.Agent.Strategy,
		)
		return nil
	}

	args := a.jaeger.Spec.Agent.Options.ToArgs()

	// we only add the grpc host if we are adding the reporter type and there's no explicit value yet
	if len(util.FindItem("--reporter.grpc.host-port=", args)) == 0 {
		args = append(args, fmt.Sprintf("--reporter.grpc.host-port=dns:///%s.%s:14250", service.GetNameForHeadlessCollectorService(a.jaeger), a.jaeger.Namespace))
	}

	// Enable tls by default for openshift platform
	if autodetect.OperatorConfiguration.GetPlatform() == autodetect.OpenShiftPlatform {
		if len(util.FindItem("--reporter.grpc.tls.enabled=", args)) == 0 {
			args = append(args, "--reporter.grpc.tls.enabled=true")
			args = append(args, fmt.Sprintf("--reporter.grpc.tls.ca=%s", ca.ServiceCAPath))
			args = append(args, fmt.Sprintf("--reporter.grpc.tls.server-name=%s.%s.svc.cluster.local", service.GetNameForHeadlessCollectorService(a.jaeger), a.jaeger.Namespace))
		}
	}

	zkCompactTrft := util.GetPort("--processor.zipkin-compact.server-host-port=", args, 5775)
	configRest := util.GetPort("--http-server.host-port=", args, 5778)
	jgCompactTrft := util.GetPort("--processor.jaeger-compact.server-host-port=", args, 6831)
	jgBinaryTrft := util.GetPort("--processor.jaeger-binary.server-host-port=", args, 6832)
	adminPort := util.GetAdminPort(args, 14271)

	trueVar := true
	falseVar := false
	labels := util.Labels(a.name(), "agent", *a.jaeger)

	baseCommonSpec := v1.JaegerCommonSpec{
		Annotations: map[string]string{
			"prometheus.io/scrape": "true",
			"prometheus.io/port":   strconv.Itoa(int(adminPort)),
			"linkerd.io/inject":    "disabled",
		},
		Labels: labels,
	}

	commonSpec := util.Merge([]v1.JaegerCommonSpec{a.jaeger.Spec.Agent.JaegerCommonSpec, a.jaeger.Spec.JaegerCommonSpec, baseCommonSpec})
	_, ok := commonSpec.Annotations["sidecar.istio.io/inject"]
	if !ok {
		commonSpec.Annotations["sidecar.istio.io/inject"] = "false"
	}

	ca.Update(a.jaeger, commonSpec)
	ca.AddServiceCA(a.jaeger, commonSpec)

	// ensure we have a consistent order of the arguments
	// see https://github.com/jaegertracing/jaeger-operator/issues/334
	sort.Strings(args)

	hostNetwork := false
	dnsPolicy := a.jaeger.Spec.Agent.DNSPolicy
	if a.jaeger.Spec.Agent.HostNetwork != nil {
		hostNetwork = *a.jaeger.Spec.Agent.HostNetwork
		if dnsPolicy == "" {
			dnsPolicy = corev1.DNSClusterFirstWithHostNet
		}
	}

	priorityClassName := ""
	if a.jaeger.Spec.Agent.PriorityClassName != "" {
		priorityClassName = a.jaeger.Spec.Agent.PriorityClassName
	}

	livenessProbe := &corev1.Probe{
		ProbeHandler: corev1.ProbeHandler{
			HTTPGet: &corev1.HTTPGetAction{
				Path: "/",
				Port: intstr.FromInt(int(adminPort)),
			},
		},
		InitialDelaySeconds: 5,
		PeriodSeconds:       15,
		FailureThreshold:    5,
	}
	if a.jaeger.Spec.Agent.LivenessProbe != nil {
		livenessProbe = a.jaeger.Spec.Agent.LivenessProbe
	}

	return &appsv1.DaemonSet{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "apps/v1",
			Kind:       "DaemonSet",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      fmt.Sprintf("%s-agent-daemonset", a.jaeger.Name),
			Namespace: a.jaeger.Namespace,
			Labels:    commonSpec.Labels,
			OwnerReferences: []metav1.OwnerReference{{
				APIVersion: a.jaeger.APIVersion,
				Kind:       a.jaeger.Kind,
				Name:       a.jaeger.Name,
				UID:        a.jaeger.UID,
				Controller: &trueVar,
			}},
		},
		Spec: appsv1.DaemonSetSpec{
			Selector: &metav1.LabelSelector{
				MatchLabels: commonSpec.Labels,
			},
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels:      commonSpec.Labels,
					Annotations: commonSpec.Annotations,
				},
				Spec: corev1.PodSpec{
					ImagePullSecrets: a.jaeger.Spec.Agent.ImagePullSecrets,
					Containers: []corev1.Container{{
						Image: util.ImageName(a.jaeger.Spec.Agent.Image, "jaeger-agent-image"),
						Name:  "jaeger-agent-daemonset",
						Args:  args,
						Env:   proxy.ReadProxyVarsFromEnv(),
						Ports: []corev1.ContainerPort{
							{
								ContainerPort: zkCompactTrft,
								HostPort:      zkCompactTrft,
								Name:          "zk-compact-trft",
								Protocol:      corev1.ProtocolUDP,
							},
							{
								ContainerPort: configRest,
								HostPort:      configRest,
								Name:          "config-rest",
							},
							{
								ContainerPort: jgCompactTrft,
								HostPort:      jgCompactTrft,
								Name:          "jg-compact-trft",
								Protocol:      corev1.ProtocolUDP,
							},
							{
								ContainerPort: jgBinaryTrft,
								HostPort:      jgBinaryTrft,
								Name:          "jg-binary-trft",
								Protocol:      corev1.ProtocolUDP,
							},
							{
								ContainerPort: adminPort,
								HostPort:      adminPort,
								Name:          "admin-http",
							},
						},
						LivenessProbe: livenessProbe,
						ReadinessProbe: &corev1.Probe{
							ProbeHandler: corev1.ProbeHandler{
								HTTPGet: &corev1.HTTPGetAction{
									Path: "/",
									Port: intstr.FromInt(int(adminPort)),
								},
							},
							InitialDelaySeconds: 1,
						},
						Resources:       commonSpec.Resources,
						VolumeMounts:    commonSpec.VolumeMounts,
						ImagePullPolicy: commonSpec.ImagePullPolicy,
						SecurityContext: commonSpec.ContainerSecurityContext,
					}},
					DNSPolicy:          dnsPolicy,
					HostNetwork:        hostNetwork,
					PriorityClassName:  priorityClassName,
					Volumes:            commonSpec.Volumes,
					Affinity:           commonSpec.Affinity,
					Tolerations:        commonSpec.Tolerations,
					SecurityContext:    commonSpec.SecurityContext,
					ServiceAccountName: account.JaegerServiceAccountFor(a.jaeger, account.AgentComponent),
					EnableServiceLinks: &falseVar,
				},
			},
		},
	}
}

func (a *Agent) name() string {
	return fmt.Sprintf("%s-agent", a.jaeger.Name)
}
