package inject

import (
	"fmt"
	"reflect"
	"sort"
	"strings"

	"github.com/jaegertracing/jaeger-operator/pkg/config/ca"
	"github.com/jaegertracing/jaeger-operator/pkg/config/otelconfig"

	log "github.com/sirupsen/logrus"
	"github.com/spf13/viper"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"

	v1 "github.com/jaegertracing/jaeger-operator/pkg/apis/jaegertracing/v1"
	"github.com/jaegertracing/jaeger-operator/pkg/deployment"
	"github.com/jaegertracing/jaeger-operator/pkg/service"
	"github.com/jaegertracing/jaeger-operator/pkg/util"
)

var (
	// Annotation is the annotation name to look for when deciding whether or not to inject
	Annotation = "sidecar.jaegertracing.io/inject"
	// Label is the label name the operator put on injected deployments.
	Label = "sidecar.jaegertracing.io/injected"
	// AnnotationLegacy holds the annotation name we had in the past, which we keep for backwards compatibility
	AnnotationLegacy = "inject-jaeger-agent"
	// PrometheusDefaultAnnotations is a map containing annotations for prometheus to be inserted at sidecar in case it doesn't have any
	PrometheusDefaultAnnotations = map[string]string{
		"prometheus.io/scrape": "true",
		"prometheus.io/port":   "14271",
	}
)

const (
	envVarTags        = "JAEGER_TAGS"
	envVarServiceName = "JAEGER_SERVICE_NAME"
	envVarPropagation = "JAEGER_PROPAGATION"
	envVarPodName     = "POD_NAME"
	envVarHostIP      = "HOST_IP"
)

// Sidecar adds a new container to the deployment, connecting to the given jaeger instance
func Sidecar(jaeger *v1.Jaeger, dep *appsv1.Deployment) *appsv1.Deployment {
	deployment.NewAgent(jaeger) // we need some initialization from that, but we don't actually need the agent's instance here
	logFields := jaeger.Logger().WithField("deployment", dep.Name)

	if jaeger == nil {
		logFields.Trace("no Jaeger instance found, skipping sidecar injection")
		return dep
	}

	if val, ok := dep.Labels[Label]; ok && val != jaeger.Name {
		logFields.Trace("deployment is assigned to a different Jaeger instance, skipping sidecar injection")
		return dep
	}
	decorate(dep)
	hasAgent, agentContainerIndex := HasJaegerAgent(dep)
	logFields.Debug("injecting sidecar")
	if hasAgent { // This is an update
		dep.Spec.Template.Spec.Containers[agentContainerIndex] = container(jaeger, dep)
	} else {
		dep.Spec.Template.Spec.Containers = append(dep.Spec.Template.Spec.Containers, container(jaeger, dep))

	}

	jaegerName := util.Truncate(jaeger.Name, 63)

	if dep.Labels == nil {
		dep.Labels = map[string]string{Label: jaegerName}
	} else {
		dep.Labels[Label] = jaegerName
	}

	return dep
}

// Needed determines whether a pod needs to get a sidecar injected or not
func Needed(dep *appsv1.Deployment, ns *corev1.Namespace) bool {
	_, depExist := dep.Annotations[Annotation]
	_, nsExist := ns.Annotations[Annotation]
	if !depExist && !nsExist {
		log.WithFields(log.Fields{
			"namespace":  dep.Namespace,
			"deployment": dep.Name,
		}).Trace("annotation not present, not injecting")
		return false
	}
	// do not inject jaeger due to port collision
	// do not inject if deployment's Annotation value is false
	if dep.Labels["app"] == "jaeger" {
		return false
	}

	hasAgent, _ := HasJaegerAgent(dep)

	if hasAgent {
		// has already a sidecar injected and managed by the operator
		// return true because could require an update.
		_, hasLabel := dep.Labels[Label]
		return hasLabel
	}
	// If no agent but has annotations
	return true
}

// Select a suitable Jaeger from the JaegerList for the given Pod, or nil of none is suitable
func Select(target *appsv1.Deployment, ns *corev1.Namespace, availableJaegerPods *v1.JaegerList) *v1.Jaeger {
	jaegerNameDep := target.Annotations[Annotation]
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
		instancesInNamespace := getJaegerFromNamespace(target.Namespace, availableJaegerPods)
		if len(instancesInNamespace) == 1 {
			jaeger := instancesInNamespace[0]
			return jaeger
		}
		// At this point, we have more than one instance that could be used to inject
		// we should just not inject, as it's not clear which one should be used.
	}
	return nil
}

func getJaegerFromNamespace(namespace string, jaegers *v1.JaegerList) []*v1.Jaeger {
	var instances []*v1.Jaeger
	for _, p := range jaegers.Items {
		if p.Namespace == namespace {
			// matched the namespace!
			instances = append(instances, &p)
		}
	}
	return instances
}

func getJaeger(name string, jaegers *v1.JaegerList) *v1.Jaeger {
	for _, p := range jaegers.Items {
		if p.Name == name {
			// matched the name!
			return &p
		}
	}
	return nil
}

func container(jaeger *v1.Jaeger, dep *appsv1.Deployment) corev1.Container {
	args := append(jaeger.Spec.Agent.Options.ToArgs())

	// we only add the grpc host if we are adding the reporter type and there's no explicit value yet
	if len(util.FindItem("--reporter.grpc.host-port=", args)) == 0 {
		args = append(args, fmt.Sprintf("--reporter.grpc.host-port=dns:///%s.%s.svc:14250", service.GetNameForHeadlessCollectorService(jaeger), jaeger.Namespace))
	}

	// Enable tls by default for openshift platform
	if viper.GetString("platform") == v1.FlagPlatformOpenShift {
		if len(util.FindItem("--reporter.grpc.tls.enabled=true", args)) == 0 {
			args = append(args, "--reporter.grpc.tls.enabled=true")
			args = append(args, fmt.Sprintf("--reporter.grpc.tls.ca=%s", ca.ServiceCAPath))
		}
	}

	zkCompactTrft := util.GetPort("--processor.zipkin-compact.server-host-port=", args, 5775)
	configRest := util.GetPort("--http-server.host-port=", args, 5778)
	jgCompactTrft := util.GetPort("--processor.jaeger-compact.server-host-port=", args, 6831)
	jgBinaryTrft := util.GetPort("--processor.jaeger-binary.server-host-port=", args, 6832)
	adminPort := util.GetPort("--admin-http-port=", args, 14271)

	if len(util.FindItem("--jaeger.tags=", args)) == 0 {
		agentTags := fmt.Sprintf("%s=%s,%s=%s,%s=%s,%s=%s,%s=%s",
			"cluster", "undefined", // this value isn't currently available
			"deployment.name", dep.Name,
			"pod.namespace", dep.Namespace,
			"pod.name", fmt.Sprintf("${%s:}", envVarPodName),
			"host.ip", fmt.Sprintf("${%s:}", envVarHostIP),
		)

		if len(dep.Spec.Template.Spec.Containers) == 1 {
			agentTags = fmt.Sprintf("%s,%s=%s", agentTags,
				"container.name", dep.Spec.Template.Spec.Containers[0].Name,
			)
		}

		args = append(args, fmt.Sprintf(`--jaeger.tags=%s`, agentTags))
	}

	commonSpec := util.Merge([]v1.JaegerCommonSpec{jaeger.Spec.Agent.JaegerCommonSpec, jaeger.Spec.JaegerCommonSpec})

	// Use only the agent common spec for volumes and mounts.
	// We don't want to mount all Jaeger internal volumes into user's deployments
	volumesAndMountsSpec := &jaeger.Spec.Agent.JaegerCommonSpec
	otelConf, err := jaeger.Spec.Agent.Config.GetMap()
	if err != nil {
		jaeger.Logger().WithField("error", err).
			WithField("component", "agent").
			Errorf("Could not parse OTEL config, config map will not be created")
	} else if otelconfig.ShouldCreate(jaeger, jaeger.Spec.Agent.Options, otelConf) {
		otelconfig.Update(jaeger, "agent", volumesAndMountsSpec, &args)
	}

	ca.Update(jaeger, volumesAndMountsSpec)
	ca.AddServiceCA(jaeger, volumesAndMountsSpec)

	// ensure we have a consistent order of the arguments
	// see https://github.com/jaegertracing/jaeger-operator/issues/334
	sort.Strings(args)

	dep.Spec.Template.Spec.ImagePullSecrets = util.RemoveDuplicatedImagePullSecrets(append(dep.Spec.Template.Spec.ImagePullSecrets, jaeger.Spec.Agent.ImagePullSecrets...))
	dep.Spec.Template.Spec.Volumes = util.RemoveDuplicatedVolumes(append(dep.Spec.Template.Spec.Volumes, volumesAndMountsSpec.Volumes...))
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
		Resources:    commonSpec.Resources,
		VolumeMounts: volumesAndMountsSpec.VolumeMounts,
	}
}

func decorate(dep *appsv1.Deployment) {
	app, found := dep.Spec.Template.Labels["app.kubernetes.io/instance"]
	if !found {
		app, found = dep.Spec.Template.Labels["app.kubernetes.io/name"]
	}
	if !found {
		app, found = dep.Spec.Template.Labels["app"]
	}
	if found {
		// Append the namespace to the app name. Using the DNS style "<app>.<namespace>""
		// which also matches with the style used in Istio.
		if len(dep.Namespace) > 0 {
			app += "." + dep.Namespace
		} else {
			app += ".default"
		}
		for i := 0; i < len(dep.Spec.Template.Spec.Containers); i++ {
			if !hasEnv(envVarServiceName, dep.Spec.Template.Spec.Containers[i].Env) {
				dep.Spec.Template.Spec.Containers[i].Env = append(dep.Spec.Template.Spec.Containers[i].Env, corev1.EnvVar{
					Name:  envVarServiceName,
					Value: app,
				})
			}
			if !hasEnv(envVarPropagation, dep.Spec.Template.Spec.Containers[i].Env) {
				dep.Spec.Template.Spec.Containers[i].Env = append(dep.Spec.Template.Spec.Containers[i].Env, corev1.EnvVar{
					Name:  envVarPropagation,
					Value: "jaeger,b3",
				})
			}
		}
	}
	for key, value := range PrometheusDefaultAnnotations {
		_, ok := dep.Annotations[key]
		if !ok {
			dep.Annotations[key] = value
		}
	}

}

func hasEnv(name string, vars []corev1.EnvVar) bool {
	for i := 0; i < len(vars); i++ {
		if vars[i].Name == name {
			return true
		}
	}
	return false
}

// CleanSidecar of  deployments  associated with the jaeger instance.
func CleanSidecar(instanceName string, deployment *appsv1.Deployment) {
	delete(deployment.Labels, Label)
	for c := 0; c < len(deployment.Spec.Template.Spec.Containers); c++ {
		if deployment.Spec.Template.Spec.Containers[c].Name == "jaeger-agent" {
			// delete jaeger-agent container
			deployment.Spec.Template.Spec.Containers = append(deployment.Spec.Template.Spec.Containers[:c], deployment.Spec.Template.Spec.Containers[c+1:]...)
			break
		}
	}
	if viper.GetString("platform") == v1.FlagPlatformOpenShift {
		names := map[string]bool{
			ca.TrustedCANameFromString(instanceName): true,
			ca.ServiceCANameFromString(instanceName): true,
		}
		// Remove the managed volumes, if present
		for v := 0; v < len(deployment.Spec.Template.Spec.Volumes); v++ {
			if _, ok := names[deployment.Spec.Template.Spec.Volumes[v].Name]; ok {
				// delete managed volume
				deployment.Spec.Template.Spec.Volumes = append(deployment.Spec.Template.Spec.Volumes[:v], deployment.Spec.Template.Spec.Volumes[v+1:]...)
				v--
			}
		}
	}
}

// HasJaegerAgent checks whether deployment has Jaeger Agent container
func HasJaegerAgent(dep *appsv1.Deployment) (bool, int) {
	// this pod is annotated, it should have a sidecar
	// but does it already have one?
	for i, container := range dep.Spec.Template.Spec.Containers {
		if container.Name == "jaeger-agent" { // we don't labels/annotations on containers, so, we rely on its name
			return true, i
		}
	}
	return false, -1
}

// EqualSidecar check if two deployments sidecar are equal
func EqualSidecar(dep, oldDep *appsv1.Deployment) bool {
	depHasAgent, depAgentIndex := HasJaegerAgent(dep)
	oldDepHasAgent, oldDepIndex := HasJaegerAgent(oldDep)
	if depHasAgent != oldDepHasAgent {
		return false
	}
	depContainer := dep.Spec.Template.Spec.Containers[depAgentIndex]
	oldDepContainer := oldDep.Spec.Template.Spec.Containers[oldDepIndex]
	return reflect.DeepEqual(depContainer, oldDepContainer)
}
