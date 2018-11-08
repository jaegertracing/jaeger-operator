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
	"github.com/jaegertracing/jaeger-operator/pkg/storage"
	"github.com/jaegertracing/jaeger-operator/pkg/util"
)

// Query builds pods for jaegertracing/jaeger-query
type Query struct {
	jaeger *v1alpha1.Jaeger
}

// NewQuery builds a new Query struct based on the given spec
func NewQuery(jaeger *v1alpha1.Jaeger) *Query {
	if jaeger.Spec.Query.Size <= 0 {
		jaeger.Spec.Query.Size = 1
	}

	if jaeger.Spec.Query.Image == "" {
		jaeger.Spec.Query.Image = fmt.Sprintf("%s:%s", viper.GetString("jaeger-query-image"), viper.GetString("jaeger-version"))
	}

	return &Query{jaeger: jaeger}
}

// Get returns a deployment specification for the current instance
func (q *Query) Get() *appsv1.Deployment {
	logrus.Debug("Assembling a query deployment")
	selector := q.selector()
	trueVar := true
	replicas := int32(q.jaeger.Spec.Query.Size)
	annotations := map[string]string{
		"prometheus.io/scrape":    "true",
		"prometheus.io/port":      "16686",
		"sidecar.istio.io/inject": "false",

		// note that we are explicitly using a string here, not the value from `inject.Annotation`
		// this has two reasons:
		// 1) as it is, it would cause a circular dependency, so, we'd have to extract that constant to somewhere else
		// 2) this specific string is part of the "public API" of the operator: we should not change
		// it at will. So, we leave this configured just like any other application would
		"inject-jaeger-agent": q.jaeger.Name,
	}
	for k, v := range q.jaeger.Spec.Annotations {
		annotations[k] = v
	}
	for k, v := range q.jaeger.Spec.Query.Annotations {
		annotations[k] = v
	}

	return &appsv1.Deployment{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "apps/v1",
			Kind:       "Deployment",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      fmt.Sprintf("%s-query", q.jaeger.Name),
			Namespace: q.jaeger.Namespace,
			OwnerReferences: []metav1.OwnerReference{
				metav1.OwnerReference{
					APIVersion: q.jaeger.APIVersion,
					Kind:       q.jaeger.Kind,
					Name:       q.jaeger.Name,
					UID:        q.jaeger.UID,
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
						Image: q.jaeger.Spec.Query.Image,
						Name:  "jaeger-query",
						Args: allArgs(q.jaeger.Spec.Query.Options,
							q.jaeger.Spec.Storage.Options.Filter(storage.OptionsPrefix(q.jaeger.Spec.Storage.Type))),
						Env: []v1.EnvVar{
							v1.EnvVar{
								Name:  "SPAN_STORAGE_TYPE",
								Value: q.jaeger.Spec.Storage.Type,
							},
						},
						VolumeMounts: util.RemoveDuplicatedVolumeMounts(append(q.jaeger.Spec.Query.VolumeMounts, q.jaeger.Spec.VolumeMounts...)),
						Ports: []v1.ContainerPort{
							{
								ContainerPort: 16686,
								Name:          "query",
							},
						},
						ReadinessProbe: &v1.Probe{
							Handler: v1.Handler{
								HTTPGet: &v1.HTTPGetAction{
									Path: "/",
									Port: intstr.FromInt(16687),
								},
							},
							InitialDelaySeconds: 1,
						},
					},
					},
					Volumes: util.RemoveDuplicatedVolumes(append(q.jaeger.Spec.Query.Volumes, q.jaeger.Spec.Volumes...)),
				},
			},
		},
	}
}

// Services returns a list of services to be deployed along with the query deployment
func (q *Query) Services() []*v1.Service {
	selector := q.selector()
	return []*v1.Service{
		service.NewQueryService(q.jaeger, selector),
	}
}

func (q *Query) selector() map[string]string {
	return map[string]string{"app": "jaeger", "jaeger": q.jaeger.Name, "jaeger-component": "query"}
}
