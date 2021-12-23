package inject

import (
	"fmt"
	"sort"
	"strings"

	log "github.com/sirupsen/logrus"
	"github.com/spf13/viper"
	corev1 "k8s.io/api/core/v1"

	v1 "github.com/jaegertracing/jaeger-operator/apis/v1"
	"github.com/jaegertracing/jaeger-operator/pkg/config/ca"
	"github.com/jaegertracing/jaeger-operator/pkg/deployment"
	"github.com/jaegertracing/jaeger-operator/pkg/service"
	"github.com/jaegertracing/jaeger-operator/pkg/util"
)

// SidecarPod adds a new container to the pod, connecting to the given jaeger instance
func SidecarPod(jaeger *v1.Jaeger, pod *corev1.Pod) *corev1.Pod {
	deployment.NewAgent(jaeger) // we need some initialization from that, but we don't actually need the agent's instance here
	logFields := jaeger.Logger().WithField("pod", pod.Name)

	if jaeger == nil {
		logFields.Trace("no Jaeger instance found, skipping sidecar injection")
		return pod
	}

	if val, ok := pod.Labels[Label]; ok && val != jaeger.Name {
		logFields.Trace("pod is assigned to a different Jaeger instance, skipping sidecar injection")
		return pod
	}
	decoratePod(pod)
	hasAgent, agentContainerIndex := HasJaegerAgent(pod.Spec.Containers)
	logFields.Debug("injecting sidecar")
	if hasAgent { // This is an update
		pod.Spec.Containers[agentContainerIndex] = containerPod(jaeger, pod, agentContainerIndex)
	} else {
		pod.Spec.Containers = append(pod.Spec.Containers, containerPod(jaeger, pod, -1))
	}

	jaegerName := util.Truncate(jaeger.Name, 63)

	if pod.Labels == nil {
		pod.Labels = map[string]string{Label: jaegerName}
	} else {
		pod.Labels[Label] = jaegerName
	}

	return pod
}

// PodNeeded determines whether a pod needs to get a sidecar injected or not
// For pod injection, we only inject if and only if
// 1. no label "sidecar.jaegertracing.io/injected"
// 2. no container named "jaeger-agent"
// the fulfillment of the above conditions normally imply the pod has been taken over the pod controller
func PodNeeded(pod *corev1.Pod, ns *corev1.Namespace) bool {
	if !desiredPod(pod, ns) {
		return false
	}

	// do not inject jaeger due to port collision
	// do not inject if pod's Annotation value is false
	if pod.Labels["app"] == "jaeger" && pod.Labels["app.kubernetes.io/component"] != "query" {
		return false
	}

	// A simple test
	// Skip if hasLabel is true which means the agent should have been injected
	if _, hasLabel := pod.Labels[Label]; hasLabel {
		return false
	}

	// A detailed check, whether there is a jaeger-agent container being injected
	hasAgent, _ := HasJaegerAgent(pod.Spec.Containers)

	if hasAgent {
		return false
	}

	// If no agent at all but has annotations!
	return true
}

// SelectForPod a suitable Jaeger from the JaegerList for the given Pod, or nil of none is suitable
func SelectForPod(pod *corev1.Pod, ns *corev1.Namespace, availableJaegerPods *v1.JaegerList) *v1.Jaeger {
	jaegerNameDep := pod.Labels[PodLabel]
	jaegerNameNs := ns.Annotations[Annotation]

	if jaegerNameDep != "" && !strings.EqualFold(jaegerNameDep, "true") {
		// name on the deployment has precedence
		if jaeger := getJaeger(jaegerNameDep, availableJaegerPods); jaeger != nil {
			return jaeger
		}
		return nil
	}
	if jaeger := getJaeger(jaegerNameNs, availableJaegerPods); jaeger != nil {
		return jaeger
	}

	if strings.EqualFold(jaegerNameDep, "true") || strings.EqualFold(jaegerNameNs, "true") {
		// If there is only *one* available instance in all watched namespaces
		// then that's what we'll use
		if len(availableJaegerPods.Items) == 1 {
			jaeger := &availableJaegerPods.Items[0]
			return jaeger
		}
		// If there is more than one available instance in all watched namespaces
		// then we should find if there is only *one* on the same namespace
		// if that is the case. we should use it.
		instancesInNamespace := getJaegerFromNamespace(pod.GetNamespace(), availableJaegerPods)
		if len(instancesInNamespace) == 1 {
			jaeger := instancesInNamespace[0]
			return jaeger
		}
		// At this point, we have more than one instance that could be used to inject
		// we should just not inject, as it's not clear which one should be used.
	}
	return nil
}

// desiredPod determines whether a sidecar is desired, based on the annotation from both the pod and the namespace
func desiredPod(pod *corev1.Pod, ns *corev1.Namespace) bool {
	logger := log.WithFields(log.Fields{
		"namespace": pod.Namespace,
		"pod":       pod.Name, // resource name
	})
	appLabelValue, appExist := pod.Labels[PodLabel]
	nsInjectionLabelValue, nsExist := ns.Labels[NamespaceLabel]

	// TODO: support annotation like istio

	if appExist && !strings.EqualFold(appLabelValue, "false") {
		logger.Debug("annotation present on pod")
		return true
	}

	if nsExist && strings.EqualFold(nsInjectionLabelValue, "enabled") {
		logger.Debug("injection label present on namespace")
		return true
	}

	return false
}

func decoratePod(pod *corev1.Pod) {
	app, found := pod.Labels["app.kubernetes.io/instance"]
	if !found {
		app, found = pod.Labels["app.kubernetes.io/name"]
	}
	if !found {
		app, found = pod.Labels["app"]
	}
	if found {
		// Append the namespace to the app name. Using the DNS style "<app>.<namespace>""
		// which also matches with the style used in Istio.
		if len(pod.Namespace) > 0 {
			app += "." + pod.Namespace
		} else {
			app += ".default"
		}
		for i := 0; i < len(pod.Spec.Containers); i++ {
			if !hasEnv(envVarServiceName, pod.Spec.Containers[i].Env) {
				pod.Spec.Containers[i].Env = append(pod.Spec.Containers[i].Env, corev1.EnvVar{
					Name:  envVarServiceName,
					Value: app,
				})
			}
			if !hasEnv(envVarPropagation, pod.Spec.Containers[i].Env) {
				pod.Spec.Containers[i].Env = append(pod.Spec.Containers[i].Env, corev1.EnvVar{
					Name:  envVarPropagation,
					Value: "jaeger,b3,w3c",
				})
			}
		}
	}
	if pod.Annotations == nil {
		pod.Annotations = make(map[string]string, len(PrometheusDefaultAnnotations))
	}
	for key, value := range PrometheusDefaultAnnotations {
		_, ok := pod.Annotations[key]
		if !ok {
			pod.Annotations[key] = value
		}
	}
}

func containerPod(jaeger *v1.Jaeger, pod *corev1.Pod, agentIdx int) corev1.Container {
	args := append(jaeger.Spec.Agent.Options.ToArgs())

	// we only add the grpc host if we are adding the reporter type and there's no explicit value yet
	if len(util.FindItem("--reporter.grpc.host-port=", args)) == 0 {
		args = append(args, fmt.Sprintf("--reporter.grpc.host-port=dns:///%s.%s.svc:14250", service.GetNameForHeadlessCollectorService(jaeger), jaeger.Namespace))
	}

	// Enable tls by default for openshift platform
	if viper.GetString("platform") == v1.FlagPlatformOpenShift {
		if len(util.FindItem("--reporter.grpc.tls.enabled=", args)) == 0 {
			args = append(args, "--reporter.grpc.tls.enabled=true")
			args = append(args, fmt.Sprintf("--reporter.grpc.tls.ca=%s", ca.ServiceCAPath))
		}
	}

	zkCompactTrft := util.GetPort("--processor.zipkin-compact.server-host-port=", args, 5775)
	configRest := util.GetPort("--http-server.host-port=", args, 5778)
	jgCompactTrft := util.GetPort("--processor.jaeger-compact.server-host-port=", args, 6831)
	jgBinaryTrft := util.GetPort("--processor.jaeger-binary.server-host-port=", args, 6832)
	adminPort := util.GetAdminPort(args, 14271)

	if len(util.FindItem("--agent.tags=", args)) == 0 {
		defaultAgentTagsMap := make(map[string]string)
		defaultAgentTagsMap["cluster"] = "undefined" // this value isn't currently available
		defaultAgentTagsMap["pod.namespace"] = pod.Namespace
		defaultAgentTagsMap["pod.name"] = pod.Name
		defaultAgentTagsMap["host.ip"] = fmt.Sprintf("${%s:}", envVarHostIP)

		defaultContainerName := getContainerName(pod.Spec.Containers, agentIdx)

		// if we can deduce the container name from the PodSpec
		if defaultContainerName != "" {
			defaultAgentTagsMap["container.name"] = defaultContainerName
		}

		if agentIdx > -1 {
			existingAgentTags := parseAgentTags(pod.Spec.Containers[agentIdx].Args)
			// merge two maps
			for key, value := range defaultAgentTagsMap {
				existingAgentTags[key] = value
			}
			args = append(args, fmt.Sprintf(`--agent.tags=%s`, joinTags(existingAgentTags)))
		} else {
			args = append(args, fmt.Sprintf(`--agent.tags=%s`, joinTags(defaultAgentTagsMap)))
		}

	}

	commonSpec := util.Merge([]v1.JaegerCommonSpec{jaeger.Spec.Agent.JaegerCommonSpec, jaeger.Spec.JaegerCommonSpec})

	// Use only the agent common spec for volumes and mounts.
	// We don't want to mount all Jaeger internal volumes into user's pods
	volumesAndMountsSpec := jaeger.Spec.Agent.JaegerCommonSpec
	ca.Update(jaeger, &volumesAndMountsSpec)
	ca.AddServiceCA(jaeger, &volumesAndMountsSpec)

	// ensure we have a consistent order of the arguments
	// see https://github.com/jaegertracing/jaeger-operator/issues/334
	sort.Strings(args)

	pod.Spec.ImagePullSecrets = util.RemoveDuplicatedImagePullSecrets(append(pod.Spec.ImagePullSecrets, jaeger.Spec.Agent.ImagePullSecrets...))
	pod.Spec.Volumes = util.RemoveDuplicatedVolumes(append(pod.Spec.Volumes, volumesAndMountsSpec.Volumes...))
	return corev1.Container{
		Image: util.ImageName(jaeger.Spec.Agent.Image, "jaeger-agent-image"),
		Name:  "jaeger-agent",
		Args:  args,
		Env: []corev1.EnvVar{
			{
				Name: envVarPodName,
				ValueFrom: &corev1.EnvVarSource{
					FieldRef: &corev1.ObjectFieldSelector{
						FieldPath: "metadata.name",
					},
				},
			},
			{
				Name: envVarHostIP,
				ValueFrom: &corev1.EnvVarSource{
					FieldRef: &corev1.ObjectFieldSelector{
						FieldPath: "status.hostIP",
					},
				},
			},
		},
		Ports: []corev1.ContainerPort{
			{
				ContainerPort: zkCompactTrft,
				Name:          "zk-compact-trft",
				Protocol:      corev1.ProtocolUDP,
			},
			{
				ContainerPort: configRest,
				Name:          "config-rest",
			},
			{
				ContainerPort: jgCompactTrft,
				Name:          "jg-compact-trft",
				Protocol:      corev1.ProtocolUDP,
			},
			{
				ContainerPort: jgBinaryTrft,
				Name:          "jg-binary-trft",
				Protocol:      corev1.ProtocolUDP,
			},
			{
				ContainerPort: adminPort,
				Name:          "admin-http",
			},
		},
		Resources:       commonSpec.Resources,
		SecurityContext: jaeger.Spec.Agent.SidecarSecurityContext,
		VolumeMounts:    volumesAndMountsSpec.VolumeMounts,
	}
}
