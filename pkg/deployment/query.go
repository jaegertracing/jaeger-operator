package deployment

import (
	"fmt"
	"sort"
	"strconv"

	"github.com/spf13/viper"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"

	"github.com/jaegertracing/jaeger-operator/pkg/account"
	"github.com/jaegertracing/jaeger-operator/pkg/apis/jaegertracing/v1"
	"github.com/jaegertracing/jaeger-operator/pkg/config/ui"
	"github.com/jaegertracing/jaeger-operator/pkg/service"
	"github.com/jaegertracing/jaeger-operator/pkg/storage"
	"github.com/jaegertracing/jaeger-operator/pkg/util"
)

// Query builds pods for jaegertracing/jaeger-query
type Query struct {
	jaeger *v1.Jaeger
}

// NewQuery builds a new Query struct based on the given spec
func NewQuery(jaeger *v1.Jaeger) *Query {
	if jaeger.Spec.Query.Replicas == nil || *jaeger.Spec.Query.Replicas < 0 {
		replicaSize := int32(1)
		if jaeger.Spec.Query.Size > 0 {
			jaeger.Logger().Warn("The 'size' property for the query is deprecated. Use 'replicas' instead.")
			replicaSize = int32(jaeger.Spec.Query.Size)
		}

		jaeger.Spec.Query.Replicas = &replicaSize
	}

	if jaeger.Spec.Query.Image == "" {
		jaeger.Spec.Query.Image = fmt.Sprintf("%s:%s", viper.GetString("jaeger-query-image"), viper.GetString("jaeger-version"))
	}

	return &Query{jaeger: jaeger}
}

// Get returns a deployment specification for the current instance
func (q *Query) Get() *appsv1.Deployment {
	q.jaeger.Logger().Debug("Assembling a query deployment")
	labels := q.labels()
	trueVar := true

	args := append(q.jaeger.Spec.Query.Options.ToArgs())

	adminPort := util.GetPort("--admin-http-port=", args, 16687)

	baseCommonSpec := v1.JaegerCommonSpec{
		Annotations: map[string]string{
			"prometheus.io/scrape":    "true",
			"prometheus.io/port":      strconv.Itoa(int(adminPort)),
			"sidecar.istio.io/inject": "false",
			"linkerd.io/inject":       "disabled",

			// note that we are explicitly using a string here, not the value from `inject.Annotation`
			// this has two reasons:
			// 1) as it is, it would cause a circular dependency, so, we'd have to extract that constant to somewhere else
			// 2) this specific string is part of the "public API" of the operator: we should not change
			// it at will. So, we leave this configured just like any other application would
			"sidecar.jaegertracing.io/inject": q.jaeger.Name,
		},
		Labels: labels,
	}

	commonSpec := util.Merge([]v1.JaegerCommonSpec{q.jaeger.Spec.Query.JaegerCommonSpec, q.jaeger.Spec.JaegerCommonSpec, baseCommonSpec})

	options := allArgs(q.jaeger.Spec.Query.Options,
		q.jaeger.Spec.Storage.Options.Filter(storage.OptionsPrefix(q.jaeger.Spec.Storage.Type)))

	configmap.Update(q.jaeger, commonSpec, &options)
	var envFromSource []corev1.EnvFromSource
	if len(q.jaeger.Spec.Storage.SecretName) > 0 {
		envFromSource = append(envFromSource, corev1.EnvFromSource{
			SecretRef: &corev1.SecretEnvSource{
				LocalObjectReference: corev1.LocalObjectReference{
					Name: q.jaeger.Spec.Storage.SecretName,
				},
			},
		})
	}

	// ensure we have a consistent order of the arguments
	// see https://github.com/jaegertracing/jaeger-operator/issues/334
	sort.Strings(options)

	return &appsv1.Deployment{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "apps/v1",
			Kind:       "Deployment",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:        fmt.Sprintf("%s-query", q.jaeger.Name),
			Namespace:   q.jaeger.Namespace,
			Labels:      commonSpec.Labels,
			Annotations: commonSpec.Annotations,
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
			Replicas: q.jaeger.Spec.Query.Replicas,
			Selector: &metav1.LabelSelector{
				MatchLabels: labels,
			},
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels:      commonSpec.Labels,
					Annotations: commonSpec.Annotations,
				},
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{{
						Image: q.jaeger.Spec.Query.Image,
						Name:  "jaeger-query",
						Args:  options,
						Env: []corev1.EnvVar{
							corev1.EnvVar{
								Name:  "SPAN_STORAGE_TYPE",
								Value: q.jaeger.Spec.Storage.Type,
							},
						},
						VolumeMounts: commonSpec.VolumeMounts,
						EnvFrom:      envFromSource,
						Ports: []corev1.ContainerPort{
							{
								ContainerPort: 16686,
								Name:          "query",
							},
							{
								ContainerPort: adminPort,
								Name:          "admin-http",
							},
						},
						ReadinessProbe: &corev1.Probe{
							Handler: corev1.Handler{
								HTTPGet: &corev1.HTTPGetAction{
									Path: "/",
									Port: intstr.FromInt(int(adminPort)),
								},
							},
							InitialDelaySeconds: 1,
						},
						Resources: commonSpec.Resources,
					}},
					Volumes:            commonSpec.Volumes,
					ServiceAccountName: account.JaegerServiceAccountFor(q.jaeger, account.QueryComponent),
					Affinity:           commonSpec.Affinity,
					Tolerations:        commonSpec.Tolerations,
					SecurityContext:    commonSpec.SecurityContext,
				},
			},
		},
	}
}

// Services returns a list of services to be deployed along with the query deployment
func (q *Query) Services() []*corev1.Service {
	labels := q.labels()
	return []*corev1.Service{
		service.NewQueryService(q.jaeger, labels),
	}
}

func (q *Query) labels() map[string]string {
	return map[string]string{
		"app":                          "jaeger", // TODO(jpkroehling): see collector.go in this package
		"app.kubernetes.io/name":       q.name(),
		"app.kubernetes.io/instance":   q.jaeger.Name,
		"app.kubernetes.io/component":  "query",
		"app.kubernetes.io/part-of":    "jaeger",
		"app.kubernetes.io/managed-by": "jaeger-operator",
	}
}

func (q *Query) name() string {
	return fmt.Sprintf("%s-query", q.jaeger.Name)
}
