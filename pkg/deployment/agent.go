package deployment

import (
	"fmt"
	"strings"

	"github.com/sirupsen/logrus"
	appsv1 "k8s.io/api/apps/v1"
	"k8s.io/api/core/v1"

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
		jaeger.Spec.Agent.Image = "jaegertracing/jaeger-agent:1.6" // TODO: externalize this
	}

	return &Agent{jaeger: jaeger}
}

// Get returns a Agent pod
func (a *Agent) Get() *appsv1.Deployment {
	if strings.ToLower(a.jaeger.Spec.Agent.Strategy) != "daemonset" {
		logrus.Infof(
			"The Jaeger instance '%v' is using a Sidecar strategy for the Jaeger Agent. Skipping its DaemonSet deployment.",
			a.jaeger.ObjectMeta.Name,
		)
		return nil
	}

	logrus.Infof("DaemonSet deployments aren't supported yet. Jaeger instance: '%v'", a.jaeger.ObjectMeta.Name)
	return nil
}

// InjectSidecar adds a new container to the deployment, containing Jaeger's agent
func (a *Agent) InjectSidecar(dep appsv1.Deployment) *appsv1.Deployment {
	sidecar := v1.Container{
		Image: a.jaeger.Spec.Agent.Image,
		Name:  "jaeger-agent",
		Args:  []string{fmt.Sprintf("--collector.host-port=%s:14267", service.GetNameForCollectorService(a.jaeger))},
		Ports: []v1.ContainerPort{
			{
				ContainerPort: 5775,
				Name:          "zk-compact-trft",
			},
			{
				ContainerPort: 5778,
				Name:          "config-rest",
			},
			{
				ContainerPort: 6831,
				Name:          "jg-compact-trft",
			},
			{
				ContainerPort: 6832,
				Name:          "jg-binary-trft",
			},
		},
	}

	dep.Spec.Template.Spec.Containers = append(dep.Spec.Template.Spec.Containers, sidecar)
	return &dep
}
