package deployment

import (
	autoscalingv2 "k8s.io/api/autoscaling/v2"
	autoscalingv2beta2 "k8s.io/api/autoscaling/v2beta2"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"

	"github.com/spf13/viper"

	v1 "github.com/jaegertracing/jaeger-operator/apis/v1"
	"github.com/jaegertracing/jaeger-operator/pkg/util"
)

const (
	// we need to have an upper bound, and 100 seems like a "good" max value
	defaultMaxReplicas = int32(100)

	// for both memory and cpu
	defaultAvgUtilization = int32(90)
)

type component interface {
	name() string
	hpaLabels() map[string]string
	replicas() *int32
	commonSpec() v1.JaegerCommonSpec
	autoscalingSpec() v1.AutoScaleSpec
	jaegerInstance() *v1.Jaeger
}

// Autoscalers returns a list of HPAs based on specs
func autoscalers(component component) []runtime.Object {
	// fixed number of replicas is explicitly set, do not auto scale
	if component.replicas() != nil {
		return []runtime.Object{}
	}

	autoScaleSpec := component.autoscalingSpec()

	// explicitly disabled, do not auto scale
	if autoScaleSpec.Autoscale != nil && !*autoScaleSpec.Autoscale {
		return []runtime.Object{}
	}

	maxReplicas := int32(-1) // unset, or invalid value

	if nil != autoScaleSpec.MaxReplicas {
		maxReplicas = *autoScaleSpec.MaxReplicas
	}
	if maxReplicas < 0 {
		maxReplicas = defaultMaxReplicas
	}

	baseCommonSpec := v1.JaegerCommonSpec{
		Labels: component.hpaLabels(),
	}

	avgUtilization := defaultAvgUtilization
	trueVar := true
	jaeger := component.jaegerInstance()
	commonSpec := util.Merge([]v1.JaegerCommonSpec{component.commonSpec(), jaeger.Spec.JaegerCommonSpec, baseCommonSpec})

	// scale up when either CPU or memory is above 90%
	var result []runtime.Object

	autoscalingVersion := viper.GetString(v1.FlagAutoscalingVersion)

	if autoscalingVersion == v1.FlagAutoscalingVersionV2Beta2 {
		autoscaler := autoscalingv2beta2.HorizontalPodAutoscaler{
			TypeMeta: metav1.TypeMeta{
				Kind:       "HorizontalPodAutoscaler",
				APIVersion: autoscalingVersion,
			},
			ObjectMeta: metav1.ObjectMeta{
				Name:        component.name(),
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
					Name:       component.name(),
				},
				MinReplicas: autoScaleSpec.MinReplicas,
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
		}
		result = append(result, &autoscaler)
	} else {
		autoscaler := autoscalingv2.HorizontalPodAutoscaler{
			TypeMeta: metav1.TypeMeta{
				Kind:       "HorizontalPodAutoscaler",
				APIVersion: autoscalingVersion,
			},
			ObjectMeta: metav1.ObjectMeta{
				Name:        component.name(),
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
			Spec: autoscalingv2.HorizontalPodAutoscalerSpec{
				ScaleTargetRef: autoscalingv2.CrossVersionObjectReference{
					APIVersion: "apps/v1",
					Kind:       "Deployment",
					Name:       component.name(),
				},
				MinReplicas: autoScaleSpec.MinReplicas,
				MaxReplicas: maxReplicas,
				Metrics: []autoscalingv2.MetricSpec{
					{
						Type: autoscalingv2.ResourceMetricSourceType,
						Resource: &autoscalingv2.ResourceMetricSource{
							Name: corev1.ResourceCPU,
							Target: autoscalingv2.MetricTarget{
								Type:               autoscalingv2.UtilizationMetricType,
								AverageUtilization: &avgUtilization,
							},
						},
					},
					{
						Type: autoscalingv2.ResourceMetricSourceType,
						Resource: &autoscalingv2.ResourceMetricSource{
							Name: corev1.ResourceMemory,
							Target: autoscalingv2.MetricTarget{
								Type:               autoscalingv2.UtilizationMetricType,
								AverageUtilization: &avgUtilization,
							},
						},
					},
				},
			},
		}
		result = append(result, &autoscaler)
	}
	return result
}
