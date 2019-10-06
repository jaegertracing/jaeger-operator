package inject

import (
	"fmt"
	"sort"
	"strings"

	log "github.com/sirupsen/logrus"
	"github.com/spf13/viper"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"

	v1 "github.com/jaegertracing/jaeger-operator/pkg/apis/jaegertracing/v1"
	"github.com/jaegertracing/jaeger-operator/pkg/deployment"
	"github.com/jaegertracing/jaeger-operator/pkg/service"
	"github.com/jaegertracing/jaeger-operator/pkg/util"
	"github.com/jaegertracing/jaeger-operator/pkg/version"
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
		"prometheus.io/port":   "5778",
	}
)

const (
	envVarTags        = "JAEGER_TAGS"
	envVarServiceName = "JAEGER_SERVICE_NAME"
	envVarPropagation = "JAEGER_PROPAGATION"
	envVarPodName     = "POD_NAME"
)

// Sidecar adds a new container to the deployment, connecting to the given jaeger instance
func Sidecar(jaeger *v1.Jaeger, dep *appsv1.Deployment) *appsv1.Deployment {
	deployment.NewAgent(jaeger) // we need some initialization from that, but we don't actually need the agent's instance here
	logFields := jaeger.Logger().WithField("deployment", dep.Name)

	if jaeger == nil || (dep.Annotations[Annotation] != jaeger.Name && dep.Annotations[AnnotationLegacy] != jaeger.Name) {
		logFields.Debug("skipping sidecar injection")
	} else {
		decorate(dep)
		logFields.Debug("injecting sidecar")
		dep.Spec.Template.Spec.Containers = append(dep.Spec.Template.Spec.Containers, container(jaeger, dep))
		// Add label to deployment
		logFields.Debug("adding label to deployment")

		if dep.Labels == nil {
			dep.Labels = map[string]string{Label: jaeger.Name}
		} else {
			dep.Labels[Label] = jaeger.Name
		}
	}

	return dep
}

// Needed determines whether a pod needs to get a sidecar injected or not
func Needed(dep *appsv1.Deployment) bool {
	if dep.Annotations[Annotation] == "" {
		log.WithFields(log.Fields{
			"namespace":  dep.Namespace,
			"deployment": dep.Name,
		}).Debug("annotation not present, not injecting")
		return false
	}

	// this pod is annotated, it should have a sidecar
	// but does it already have one?
	for _, container := range dep.Spec.Template.Spec.Containers {
		if container.Name == "jaeger-agent" { // we don't labels/annotations on containers, so, we rely on its name
			return false
		}
	}

	return true
}

// Select a suitable Jaeger from the JaegerList for the given Pod, or nil of none is suitable
func Select(target *appsv1.Deployment, availableJaegerPods *v1.JaegerList) *v1.Jaeger {
	jaegerName := target.Annotations[Annotation]
	if strings.EqualFold(jaegerName, "true") && len(availableJaegerPods.Items) == 1 {
		// if there's only *one* jaeger within this namespace, then that's what
		// we'll use -- otherwise, we should just not inject, as it's not clear which
		// jaeger instance to use!
		// first, we make sure we normalize the name:
		jaeger := &availableJaegerPods.Items[0]
		target.Annotations[Annotation] = jaeger.Name
		return jaeger
	}

	for _, p := range availableJaegerPods.Items {
		if p.Name == jaegerName {
			// matched the name!
			return &p
		}
	}
	return nil
}

func container(jaeger *v1.Jaeger, dep *appsv1.Deployment) corev1.Container {
	args := append(jaeger.Spec.Agent.Options.ToArgs())

	if len(util.FindItem("--reporter.type=", args)) == 0 {
		args = append(args, "--reporter.type=grpc")

		// we only add the grpc host if we are adding the reporter type and there's no explicit value yet
		if len(util.FindItem("--reporter.grpc.host-port=", args)) == 0 {
			args = append(args, fmt.Sprintf("--reporter.grpc.host-port=dns:///%s.%s:14250", service.GetNameForHeadlessCollectorService(jaeger), jaeger.Namespace))
		}
	}

	zkCompactTrft := util.GetPort("--processor.zipkin-compact.server-host-port=", args, 5775)
	configRest := util.GetPort("--http-server.host-port=", args, 5778)
	jgCompactTrft := util.GetPort("--processor.jaeger-compact.server-host-port=", args, 6831)
	jgBinaryTrft := util.GetPort("--processor.jaeger-binary.server-host-port=", args, 6832)

	if len(util.FindItem("--jaeger.tags=", args)) == 0 {
		agentTags := fmt.Sprintf("%s=%s,%s=%s,%s=%s,%s=%s",
			"cluster", "undefined", // this value isn't currently available
			"deployment.name", dep.Name,
			"pod.namespace", dep.Namespace,
			"pod.name", fmt.Sprintf("${%s:}", envVarPodName),
		)

		if len(dep.Spec.Template.Spec.Containers) == 1 {
			agentTags = fmt.Sprintf("%s,%s=%s", agentTags,
				"container.name", dep.Spec.Template.Spec.Containers[0].Name,
			)

			args = append(args, fmt.Sprintf(`--jaeger.tags=%s`, agentTags))
		}
	}

	commonSpec := util.Merge([]v1.JaegerCommonSpec{jaeger.Spec.Agent.JaegerCommonSpec, jaeger.Spec.JaegerCommonSpec})

	image := jaeger.Spec.Agent.Image
	if image == "" {
		image = fmt.Sprintf("%s:%s", viper.GetString("jaeger-agent-image"), version.Get().Jaeger)
	}

	// ensure we have a consistent order of the arguments
	// see https://github.com/jaegertracing/jaeger-operator/issues/334
	sort.Strings(args)

	return corev1.Container{
		Image: image,
		Name:  "jaeger-agent",
		Args:  args,
		Env: []corev1.EnvVar{{
			Name: envVarPodName,
			ValueFrom: &corev1.EnvVarSource{
				FieldRef: &corev1.ObjectFieldSelector{
					FieldPath: "metadata.name",
				},
			},
		}},
		Ports: []corev1.ContainerPort{
			{
				ContainerPort: zkCompactTrft,
				Name:          "zk-compact-trft",
			},
			{
				ContainerPort: configRest,
				Name:          "config-rest",
			},
			{
				ContainerPort: jgCompactTrft,
				Name:          "jg-compact-trft",
			},
			{
				ContainerPort: jgBinaryTrft,
				Name:          "jg-binary-trft",
			},
		},
		Resources: commonSpec.Resources,
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
func CleanSidecar(deployment *appsv1.Deployment) {
	delete(deployment.Labels, Label)
	for c := 0; c < len(deployment.Spec.Template.Spec.Containers); c++ {
		if deployment.Spec.Template.Spec.Containers[c].Name == "jaeger-agent" {
			// delete jaeger-agent container
			deployment.Spec.Template.Spec.Containers = append(deployment.Spec.Template.Spec.Containers[:c], deployment.Spec.Template.Spec.Containers[c+1:]...)
			break
		}
	}
}
