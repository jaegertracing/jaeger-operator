package kafka

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	v1 "github.com/jaegertracing/jaeger-operator/pkg/apis/jaegertracing/v1"
	"github.com/jaegertracing/jaeger-operator/pkg/apis/kafka/v1beta1"
)

// Persistent returns the custom resource for a persistent Kafka
// Reference: https://github.com/strimzi/strimzi-kafka-operator/blob/master/examples/kafka/kafka-persistent.yaml
func Persistent(jaeger *v1.Jaeger) v1beta1.Kafka {
	trueVar := true
	return v1beta1.Kafka{
		ObjectMeta: metav1.ObjectMeta{
			Name:      jaeger.Name,
			Namespace: jaeger.Namespace,
			Labels: map[string]string{
				"app":                         "jaeger",
				"app.kubernetes.io/name":      jaeger.Name,
				"app.kubernetes.io/instance":  jaeger.Name,
				"app.kubernetes.io/component": "kafkauser",
				"app.kubernetes.io/part-of":   "jaeger",

				// workaround for https://github.com/strimzi/strimzi-kafka-operator/issues/2107
				"app.kubernetes.io/managed---by": "jaeger-operator",
			},
			OwnerReferences: []metav1.OwnerReference{
				metav1.OwnerReference{
					APIVersion: jaeger.APIVersion,
					Kind:       jaeger.Kind,
					Name:       jaeger.Name,
					UID:        jaeger.UID,
					Controller: &trueVar,
				},
			},
		},
		Spec: v1beta1.KafkaSpec{
			v1.NewFreeForm(map[string]interface{}{
				"kafka": map[string]interface{}{
					"version":  "2.3.0",
					"replicas": 3,
					"listeners": map[string]interface{}{
						"plain": map[string]interface{}{},
						"tls":   map[string]interface{}{},
					},
					"config": map[string]interface{}{
						"offsets.topic.replication.factor":         3,
						"transaction.state.log.replication.factor": 3,
						"transaction.state.log.min.isr":            2,
						"log.message.format.version":               "2.3",
					},
					"storage": map[string]interface{}{
						"type": "jbod",
						"volumes": []map[string]interface{}{{
							"id":          0,
							"type":        "persistent-claim",
							"size":        "100Gi",
							"deleteClaim": false,
						}},
					},
				},
				"zookeeper": map[string]interface{}{
					"replicas": 3,
					"storage": map[string]interface{}{
						"type":        "persistent-claim",
						"size":        "100Gi",
						"deleteClaim": false,
					},
				},
				"entityOperator": map[string]interface{}{
					"topicOperator": map[string]interface{}{},
					"userOperator":  map[string]interface{}{},
				},
			}),
		},
	}
}

// User returns a custom resource for a Kafka user. The Kafka Operator will then create a secret with the
// credentials for this user
func User(jaeger *v1.Jaeger) v1beta1.KafkaUser {
	trueVar := true
	return v1beta1.KafkaUser{
		ObjectMeta: metav1.ObjectMeta{
			Name:      jaeger.Name,
			Namespace: jaeger.Namespace,
			Labels: map[string]string{
				"app":                         "jaeger",
				"app.kubernetes.io/name":      jaeger.Name,
				"app.kubernetes.io/instance":  jaeger.Name,
				"app.kubernetes.io/component": "kafkauser",
				"app.kubernetes.io/part-of":   "jaeger",
				"strimzi.io/cluster":          jaeger.Name,

				// workaround for https://github.com/strimzi/strimzi-kafka-operator/issues/2107
				"app.kubernetes.io/managed---by": "jaeger-operator",
			},
			OwnerReferences: []metav1.OwnerReference{
				metav1.OwnerReference{
					APIVersion: jaeger.APIVersion,
					Kind:       jaeger.Kind,
					Name:       jaeger.Name,
					UID:        jaeger.UID,
					Controller: &trueVar,
				},
			},
		},
		Spec: v1beta1.KafkaUserSpec{
			v1.NewFreeForm(map[string]interface{}{
				"authentication": map[string]interface{}{
					"type": "tls",
				},
			}),
		},
	}
}
