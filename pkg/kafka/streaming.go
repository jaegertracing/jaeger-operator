package kafka

import (
	"fmt"

	"github.com/spf13/viper"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	v1 "github.com/jaegertracing/jaeger-operator/pkg/apis/jaegertracing/v1"
	"github.com/jaegertracing/jaeger-operator/pkg/apis/kafka/v1beta2"
	"github.com/jaegertracing/jaeger-operator/pkg/util"
)

// Persistent returns the custom resource for a persistent Kafka
// Reference: https://github.com/strimzi/strimzi-kafka-operator/blob/master/examples/kafka/kafka-persistent.yaml
func Persistent(jaeger *v1.Jaeger) v1beta2.Kafka {
	var replicas, replFactor, minIst, storage uint
	if viper.GetBool("kafka-provisioning-minimal") {
		jaeger.Logger().Warn("usage of kafka-provisioning-minimal is not supported")
		replicas = 1
		replFactor = 1
		minIst = 1
		storage = 10
	} else {
		replicas = 3
		replFactor = 3
		minIst = 2
		storage = 100
	}

	trueVar := true
	return v1beta2.Kafka{
		ObjectMeta: metav1.ObjectMeta{
			Name:      jaeger.Name,
			Namespace: jaeger.Namespace,
			Labels:    util.Labels(jaeger.Name, "kafka", *jaeger),
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
		Spec: v1beta2.KafkaSpec{
			v1.NewFreeForm(map[string]interface{}{
				"kafka": map[string]interface{}{
					"replicas": uint(replicas),
					"listeners": []map[string]interface{}{
						{
							"name": "plain",
							"port": 9092,
							"type": "internal",
							"tls":  false,
						},
						{
							"name": "tls",
							"port": 9093,
							"type": "internal",
							"tls":  true,
						},
					},
					"config": map[string]interface{}{
						"offsets.topic.replication.factor":         replFactor,
						"transaction.state.log.replication.factor": replFactor,
						"transaction.state.log.min.isr":            minIst,
						"log.message.format.version":               "2.3",
					},
					"storage": map[string]interface{}{
						"type": "jbod",
						"volumes": []map[string]interface{}{{
							"id":          0,
							"type":        "persistent-claim",
							"size":        fmt.Sprintf("%dGi", storage),
							"deleteClaim": false,
						}},
					},
				},
				"zookeeper": map[string]interface{}{
					"replicas": uint(replicas),
					"storage": map[string]interface{}{
						"type":        "persistent-claim",
						"size":        fmt.Sprintf("%dGi", storage),
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
func User(jaeger *v1.Jaeger) v1beta2.KafkaUser {
	trueVar := true

	labels := util.Labels(jaeger.Name, "kafkauser", *jaeger)

	// based on this label, the Strimzi operator will create the TLS secrets for
	// this user to access the target cluster
	labels["strimzi.io/cluster"] = jaeger.Name

	return v1beta2.KafkaUser{
		ObjectMeta: metav1.ObjectMeta{
			Name:      jaeger.Name,
			Namespace: jaeger.Namespace,
			Labels:    labels,
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
		Spec: v1beta2.KafkaUserSpec{
			v1.NewFreeForm(map[string]interface{}{
				"authentication": map[string]interface{}{
					"type": "tls",
				},
			}),
		},
	}
}
