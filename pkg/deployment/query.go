package deployment

import (
	"fmt"
	"sort"
	"strconv"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"

	"github.com/jaegertracing/jaeger-operator/pkg/account"
	v1 "github.com/jaegertracing/jaeger-operator/pkg/apis/jaegertracing/v1"
	"github.com/jaegertracing/jaeger-operator/pkg/config/ca"
	configmap "github.com/jaegertracing/jaeger-operator/pkg/config/ui"
	"github.com/jaegertracing/jaeger-operator/pkg/service"
	"github.com/jaegertracing/jaeger-operator/pkg/util"
)

// Query builds pods for jaegertracing/jaeger-query
type Query struct {
	jaeger *v1.Jaeger
}

// NewQuery builds a new Query struct based on the given spec
func NewQuery(jaeger *v1.Jaeger) *Query {
	return &Query{jaeger: jaeger}
}

// Get returns a deployment specification for the current instance
func (q *Query) Get() *appsv1.Deployment {
	q.jaeger.Logger().Debug("Assembling a query deployment")
	labels := q.labels()
	trueVar := true
	falseVar := false

	args := append(q.jaeger.Spec.Query.Options.ToArgs())

	adminPort := util.GetAdminPort(args, 16687)

	baseCommonSpec := v1.JaegerCommonSpec{
		Annotations: map[string]string{
			"prometheus.io/scrape":    "true",
			"prometheus.io/port":      strconv.Itoa(int(adminPort)),
			"sidecar.istio.io/inject": "false",
			"linkerd.io/inject":       "disabled",
		},
		Labels: labels,
	}

	jaegerDisabled := false
	if q.jaeger.Spec.Query.TracingEnabled != nil && *q.jaeger.Spec.Query.TracingEnabled == false {
		jaegerDisabled = true
	} else {
		// note that we are explicitly using a string here, not the value from `inject.Annotation`
		// this has two reasons:
		// 1) as it is, it would cause a circular dependency, so, we'd have to extract that constant to somewhere else
		// 2) this specific string is part of the "public API" of the operator: we should not change
		// it at will. So, we leave this configured just like any other application would
		baseCommonSpec.Annotations["sidecar.jaegertracing.io/inject"] = q.jaeger.Name
	}

	commonSpec := util.Merge([]v1.JaegerCommonSpec{q.jaeger.Spec.Query.JaegerCommonSpec, q.jaeger.Spec.JaegerCommonSpec, baseCommonSpec})

	options := allArgs(q.jaeger.Spec.Query.Options,
		q.jaeger.Spec.Storage.Options.Filter(q.jaeger.Spec.Storage.Type.OptionsPrefix()))

	configmap.Update(q.jaeger, commonSpec, &options)
	ca.Update(q.jaeger, commonSpec)

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

	priorityClassName := ""
	if q.jaeger.Spec.Query.PriorityClassName != "" {
		priorityClassName = q.jaeger.Spec.Query.PriorityClassName
	}

	strategy := appsv1.DeploymentStrategy{
		Type: appsv1.RecreateDeploymentStrategyType,
	}

	if q.jaeger.Spec.Query.Strategy != nil {
		strategy = *q.jaeger.Spec.Query.Strategy
	}

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
			OwnerReferences: []metav1.OwnerReference{{
				APIVersion: q.jaeger.APIVersion,
				Kind:       q.jaeger.Kind,
				Name:       q.jaeger.Name,
				UID:        q.jaeger.UID,
				Controller: &trueVar,
			}},
		},
		Spec: appsv1.DeploymentSpec{
			Replicas: q.jaeger.Spec.Query.Replicas,
			Selector: &metav1.LabelSelector{
				MatchLabels: labels,
			},
			Strategy: strategy,
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels:      commonSpec.Labels,
					Annotations: commonSpec.Annotations,
				},
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{{
						Image: util.ImageName(q.jaeger.Spec.Query.Image, "jaeger-query-image"),
						Name:  "jaeger-query",
						Args:  options,
						Env: []corev1.EnvVar{
							{
								Name:  "SPAN_STORAGE_TYPE",
								Value: string(q.jaeger.Spec.Storage.Type),
							},
							{
								Name:  "JAEGER_DISABLED",
								Value: strconv.FormatBool(jaegerDisabled),
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
						LivenessProbe: &corev1.Probe{
							Handler: corev1.Handler{
								HTTPGet: &corev1.HTTPGetAction{
									Path: "/",
									Port: intstr.FromInt(int(adminPort)),
								},
							},
							InitialDelaySeconds: 5,
							PeriodSeconds:       15,
							FailureThreshold:    5,
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
					PriorityClassName:  priorityClassName,
					Volumes:            commonSpec.Volumes,
					ServiceAccountName: account.JaegerServiceAccountFor(q.jaeger, account.QueryComponent),
					Affinity:           commonSpec.Affinity,
					Tolerations:        commonSpec.Tolerations,
					SecurityContext:    commonSpec.SecurityContext,
					EnableServiceLinks: &falseVar,
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
	return util.Labels(q.name(), "query", *q.jaeger)
}

func (q *Query) name() string {
	return fmt.Sprintf("%s-query", q.jaeger.Name)
}
