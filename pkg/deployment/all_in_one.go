package deployment

import (
	"fmt"
	"sort"
	"strconv"

	"github.com/operator-framework/operator-lib/proxy"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"

	v1 "github.com/jaegertracing/jaeger-operator/apis/v1"
	"github.com/jaegertracing/jaeger-operator/pkg/account"
	"github.com/jaegertracing/jaeger-operator/pkg/autodetect"
	"github.com/jaegertracing/jaeger-operator/pkg/config/ca"
	"github.com/jaegertracing/jaeger-operator/pkg/config/sampling"
	"github.com/jaegertracing/jaeger-operator/pkg/config/tls"
	configmap "github.com/jaegertracing/jaeger-operator/pkg/config/ui"
	"github.com/jaegertracing/jaeger-operator/pkg/service"
	"github.com/jaegertracing/jaeger-operator/pkg/storage"
	"github.com/jaegertracing/jaeger-operator/pkg/util"
)

// AllInOne builds pods for jaegertracing/all-in-one
type AllInOne struct {
	jaeger *v1.Jaeger
}

// NewAllInOne builds a new AllInOne struct based on the given spec
func NewAllInOne(jaeger *v1.Jaeger) *AllInOne {
	return &AllInOne{jaeger: jaeger}
}

// Get returns a pod for the current all-in-one configuration
func (a *AllInOne) Get() *appsv1.Deployment {
	a.jaeger.Logger().V(-1).Info("Assembling an all-in-one deployment")
	trueVar := true
	falseVar := false

	args := a.jaeger.Spec.AllInOne.Options.ToArgs()

	adminPort := util.GetAdminPort(args, 14269)

	jaegerDisabled := false
	if a.jaeger.Spec.AllInOne.TracingEnabled != nil && !*a.jaeger.Spec.AllInOne.TracingEnabled {
		jaegerDisabled = true
	}

	baseCommonSpec := v1.JaegerCommonSpec{
		Annotations: map[string]string{
			"prometheus.io/scrape": "true",
			"prometheus.io/port":   strconv.Itoa(int(adminPort)),
			"linkerd.io/inject":    "disabled",
		},
		Labels: a.labels(),
	}

	commonSpec := util.Merge([]v1.JaegerCommonSpec{a.jaeger.Spec.AllInOne.JaegerCommonSpec, a.jaeger.Spec.JaegerCommonSpec, baseCommonSpec})
	_, ok := commonSpec.Annotations["sidecar.istio.io/inject"]
	if !ok {
		commonSpec.Annotations["sidecar.istio.io/inject"] = "false"
	}

	options := util.AllArgs(a.jaeger.Spec.AllInOne.Options,
		a.jaeger.Spec.Storage.Options.Filter(a.jaeger.Spec.Storage.Type.OptionsPrefix()))

	configmap.Update(a.jaeger, commonSpec, &options)
	sampling.Update(a.jaeger, commonSpec, &options)

	// If tls is not explicitly set, update jaeger CR with the tls flags according to the platform
	if len(util.FindItem("--collector.grpc.tls.enabled=", options)) == 0 {
		tls.Update(a.jaeger, commonSpec, &options)
	}

	ca.Update(a.jaeger, commonSpec)
	ca.AddServiceCA(a.jaeger, commonSpec)
	storage.UpdateGRPCPlugin(a.jaeger, commonSpec)

	// Enable tls by default for openshift platform
	// even though the agent is in the same process as the collector, they communicate via gRPC, and the collector has TLS enabled,
	// as it might receive connections from external agents
	if autodetect.OperatorConfiguration.GetPlatform() == autodetect.OpenShiftPlatform {
		if len(util.FindItem("--reporter.grpc.host-port=", options)) == 0 &&
			len(util.FindItem("--reporter.grpc.tls.enabled=", options)) == 0 {
			options = append(options, "--reporter.grpc.tls.enabled=true")
			options = append(options, fmt.Sprintf("--reporter.grpc.tls.ca=%s", ca.ServiceCAPath))
			options = append(options, fmt.Sprintf("--reporter.grpc.tls.server-name=%s.%s.svc.cluster.local", service.GetNameForHeadlessCollectorService(a.jaeger), a.jaeger.Namespace))
		}
	}

	// ensure we have a consistent order of the arguments
	// see https://github.com/jaegertracing/jaeger-operator/issues/334
	sort.Strings(options)

	var envFromSource []corev1.EnvFromSource
	if len(a.jaeger.Spec.Storage.SecretName) > 0 {
		envFromSource = append(envFromSource, corev1.EnvFromSource{
			SecretRef: &corev1.SecretEnvSource{
				LocalObjectReference: corev1.LocalObjectReference{
					Name: a.jaeger.Spec.Storage.SecretName,
				},
			},
		})
	}

	strategy := appsv1.DeploymentStrategy{
		Type: appsv1.RecreateDeploymentStrategyType,
	}

	if a.jaeger.Spec.AllInOne.Strategy != nil {
		strategy = *a.jaeger.Spec.AllInOne.Strategy
	}

	priorityClassName := ""
	if a.jaeger.Spec.AllInOne.PriorityClassName != "" {
		priorityClassName = a.jaeger.Spec.AllInOne.PriorityClassName
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

	if a.jaeger.Spec.AllInOne.LivenessProbe != nil {
		livenessProbe = a.jaeger.Spec.AllInOne.LivenessProbe
	}

	envVars := []corev1.EnvVar{
		{
			Name:  "SPAN_STORAGE_TYPE",
			Value: string(a.jaeger.Spec.Storage.Type),
		},
		{
			Name:  "METRICS_STORAGE_TYPE",
			Value: string(a.jaeger.Spec.AllInOne.MetricsStorage.Type),
		},
		{
			Name:  "COLLECTOR_ZIPKIN_HOST_PORT",
			Value: ":9411",
		},
		{
			Name:  "JAEGER_DISABLED",
			Value: strconv.FormatBool(jaegerDisabled),
		},
	}

	if a.jaeger.Spec.AllInOne.MetricsStorage.Type == "prometheus" && a.jaeger.Spec.AllInOne.MetricsStorage.ServerUrl != "" {
		envVars = append(envVars, corev1.EnvVar{
			Name:  "PROMETHEUS_SERVER_URL",
			Value: a.jaeger.Spec.AllInOne.MetricsStorage.ServerUrl,
		})
	}

	envVars = append(envVars, proxy.ReadProxyVarsFromEnv()...)

	ports := []corev1.ContainerPort{
		{
			ContainerPort: 5775,
			Name:          "zk-compact-trft", // max 15 chars!
			Protocol:      corev1.ProtocolUDP,
		},
		{
			ContainerPort: 5778,
			Name:          "config-rest",
		},
		{
			ContainerPort: 6831,
			Name:          "jg-compact-trft",
			Protocol:      corev1.ProtocolUDP,
		},
		{
			ContainerPort: 6832,
			Name:          "jg-binary-trft",
			Protocol:      corev1.ProtocolUDP,
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
			ContainerPort: 16685,
			Name:          "grpc-query",
		},
		{
			ContainerPort: 16686,
			Name:          "query",
		},
		{
			ContainerPort: adminPort,
			Name:          "admin-http",
		},
		{
			ContainerPort: 14250,
			Name:          "grpc",
		},
	}

	return &appsv1.Deployment{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "apps/v1",
			Kind:       "Deployment",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:        a.jaeger.Name,
			Namespace:   a.jaeger.Namespace,
			Labels:      commonSpec.Labels,
			Annotations: baseCommonSpec.Annotations,
			OwnerReferences: []metav1.OwnerReference{{
				APIVersion: a.jaeger.APIVersion,
				Kind:       a.jaeger.Kind,
				Name:       a.jaeger.Name,
				UID:        a.jaeger.UID,
				Controller: &trueVar,
			}},
		},
		Spec: appsv1.DeploymentSpec{
			Selector: &metav1.LabelSelector{
				MatchLabels: commonSpec.Labels,
			},
			Strategy: strategy,
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels:      commonSpec.Labels,
					Annotations: commonSpec.Annotations,
				},
				Spec: corev1.PodSpec{
					ImagePullSecrets: commonSpec.ImagePullSecrets,
					Containers: []corev1.Container{{
						Image:         util.ImageName(a.jaeger.Spec.AllInOne.Image, "jaeger-all-in-one-image"),
						Name:          "jaeger",
						Args:          options,
						Env:           append(envVars, getOTLPEnvVars(options)...),
						VolumeMounts:  commonSpec.VolumeMounts,
						EnvFrom:       envFromSource,
						Ports:         append(ports, getOTLPContainePorts(options)...),
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
						ImagePullPolicy: commonSpec.ImagePullPolicy,
						SecurityContext: commonSpec.ContainerSecurityContext,
					}},
					PriorityClassName:  priorityClassName,
					Volumes:            commonSpec.Volumes,
					ServiceAccountName: account.JaegerServiceAccountFor(a.jaeger, account.AllInOneComponent),
					Affinity:           commonSpec.Affinity,
					Tolerations:        commonSpec.Tolerations,
					SecurityContext:    commonSpec.SecurityContext,
					EnableServiceLinks: &falseVar,
					InitContainers:     storage.GetGRPCPluginInitContainers(a.jaeger, commonSpec),
				},
			},
		},
	}
}

// Services returns a list of services to be deployed along with the all-in-one deployment
func (a *AllInOne) Services() []*corev1.Service {
	// merge defined labels with default labels
	spec := util.Merge([]v1.JaegerCommonSpec{a.jaeger.Spec.AllInOne.JaegerCommonSpec, a.jaeger.Spec.JaegerCommonSpec, {Labels: a.labels()}})
	labels := spec.Labels

	return append(service.NewCollectorServices(a.jaeger, labels),
		service.NewQueryService(a.jaeger, labels),
		service.NewAgentService(a.jaeger, labels),
	)
}

func (a *AllInOne) labels() map[string]string {
	return util.Labels(a.name(), "all-in-one", *a.jaeger)
}

func (a *AllInOne) name() string {
	return a.jaeger.Name
}
