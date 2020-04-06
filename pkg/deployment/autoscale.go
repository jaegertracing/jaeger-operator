package deployment

import (
	autoscalingv2beta2 "k8s.io/api/autoscaling/v2beta2"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	v1 "github.com/jaegertracing/jaeger-operator/pkg/apis/jaegertracing/v1"
	"github.com/jaegertracing/jaeger-operator/pkg/util"
)

const (
	// we need to have an upper bound, and 100 seems like a "good" max value
	defaultMaxReplicas = int32(100)

	// for both memory and cpu
	defaultAvgUtilization = int32(90)
)

// Autoscalers returns a list of HPAs based on specs
func autoscalers(
	componentReplicas *int32,
	label string,
	componentName string,
	componentLabels map[string]string,
	autoScaleSpec v1.AutoScaleSpec,
	componentSpec v1.JaegerCommonSpec,
	jaeger *v1.Jaeger) []autoscalingv2beta2.HorizontalPodAutoscaler {

	// fixed number of replicas is explicitly set, do not auto scale
	if componentReplicas != nil {
		return []autoscalingv2beta2.HorizontalPodAutoscaler{}
	}

	// explicitly disabled, do not auto scale
	if autoScaleSpec.Autoscale != nil && *autoScaleSpec.Autoscale == false {
		return []autoscalingv2beta2.HorizontalPodAutoscaler{}
	}

	maxReplicas := int32(-1) // unset, or invalid value

	if nil != autoScaleSpec.MaxReplicas {
		maxReplicas = *autoScaleSpec.MaxReplicas
	}
	if maxReplicas < 0 {
		maxReplicas = defaultMaxReplicas
	}

	componentLabels["app.kubernetes.io/component"] = label
	baseCommonSpec := v1.JaegerCommonSpec{
		Labels: componentLabels,
	}

	avgUtilization := defaultAvgUtilization
	trueVar := true
	commonSpec := util.Merge([]v1.JaegerCommonSpec{componentSpec, jaeger.Spec.JaegerCommonSpec, baseCommonSpec})

	// scale up when either CPU or memory is above 90%
	return []autoscalingv2beta2.HorizontalPodAutoscaler{{
		ObjectMeta: metav1.ObjectMeta{
			Name:        componentName,
			Namespace:   jaeger.Namespace,
			Labels:      commonSpec.Labels,
			Annotations: commonSpec.Annotations,
			OwnerReferences: []metav1.OwnerReference{
				{
					APIVersion: jaeger.APIVersion,
					Kind:       jaeger.Kind,
					Name:       jaeger.Name,
					UID:        jaeger.UID,
					Controller: &trueVar,
				},
			},
		},
		Spec: autoscalingv2beta2.HorizontalPodAutoscalerSpec{
			ScaleTargetRef: autoscalingv2beta2.CrossVersionObjectReference{
				APIVersion: "apps/v1",
				Kind:       "Deployment",
				Name:       componentName,
			},
			MinReplicas: jaeger.Spec.Ingester.MinReplicas,
			MaxReplicas: maxReplicas,
			Metrics: []autoscalingv2beta2.MetricSpec{
				{
					Type: autoscalingv2beta2.ResourceMetricSourceType,
					Resource: &autoscalingv2beta2.ResourceMetricSource{
						Name: corev1.ResourceCPU,
						Target: autoscalingv2beta2.MetricTarget{
							Type:               autoscalingv2beta2.UtilizationMetricType,
							AverageUtilization: &avgUtilization,
						},
					},
				},
				{
					Type: autoscalingv2beta2.ResourceMetricSourceType,
					Resource: &autoscalingv2beta2.ResourceMetricSource{
						Name: corev1.ResourceMemory,
						Target: autoscalingv2beta2.MetricTarget{
							Type:               autoscalingv2beta2.UtilizationMetricType,
							AverageUtilization: &avgUtilization,
						},
					},
				},
			},
		},
	}}
}
