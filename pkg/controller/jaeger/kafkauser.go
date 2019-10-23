package jaeger

import (
	"context"
	"errors"
	"strings"
	"time"

	log "github.com/sirupsen/logrus"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/wait"
	"sigs.k8s.io/controller-runtime/pkg/client"

	v1 "github.com/jaegertracing/jaeger-operator/pkg/apis/jaegertracing/v1"
	kafkav1beta1 "github.com/jaegertracing/jaeger-operator/pkg/apis/kafka/v1beta1"
	"github.com/jaegertracing/jaeger-operator/pkg/inventory"
)

var (
	// ErrKafkaUserRemoved is returned when a kafka user existed but has been removed
	ErrKafkaUserRemoved = errors.New("kafka user has been removed")
)

func (r *ReconcileJaeger) applyKafkaUsers(jaeger v1.Jaeger, desired []kafkav1beta1.KafkaUser) error {
	opts := []client.ListOption{
		client.InNamespace(jaeger.Namespace),
		client.MatchingLabels(map[string]string{
			"app.kubernetes.io/instance": jaeger.Name,

			// workaround for https://github.com/strimzi/strimzi-kafka-operator/issues/2107
			"app.kubernetes.io/managed---by": "jaeger-operator",
		}),
	}
	list := &kafkav1beta1.KafkaUserList{}
	if err := r.client.List(context.Background(), list, opts...); err != nil {
		return err
	}

	inv := inventory.ForKafkaUsers(list.Items, desired)
	for _, d := range inv.Create {
		jaeger.Logger().WithFields(log.Fields{
			"kafka":     d.GetName(),
			"namespace": d.GetNamespace(),
		}).Debug("creating kafka users")
		if err := r.client.Create(context.Background(), &d); err != nil {
			return err
		}
	}

	for _, d := range inv.Update {
		jaeger.Logger().WithFields(log.Fields{
			"kafka":     d.GetName(),
			"namespace": d.GetNamespace(),
		}).Debug("updating kafka user")
		if err := r.client.Update(context.Background(), &d); err != nil {
			return err
		}
	}

	// now, wait for all KafkaUsers to estabilize
	for _, d := range inv.Create {
		if err := r.waitForKafkaUserStability(d); err != nil {
			return err
		}
	}
	for _, d := range inv.Update {
		if err := r.waitForKafkaUserStability(d); err != nil {
			return err
		}
	}

	for _, d := range inv.Delete {
		jaeger.Logger().WithFields(log.Fields{
			"kafka":     d.GetName(),
			"namespace": d.GetNamespace(),
		}).Debug("deleting kafka user")
		if err := r.client.Delete(context.Background(), &d); err != nil {
			return err
		}
	}

	return nil
}

func (r *ReconcileJaeger) waitForKafkaUserStability(kafkaUser kafkav1beta1.KafkaUser) error {
	seen := false
	return wait.PollImmediate(time.Second, 5*time.Minute, func() (done bool, err error) {
		k := &kafkav1beta1.KafkaUser{}
		if err := r.client.Get(context.Background(), types.NamespacedName{Name: kafkaUser.GetName(), Namespace: kafkaUser.GetNamespace()}, k); err != nil {
			if k8serrors.IsNotFound(err) {
				if seen {
					// we have seen this object before, but it doesn't exist anymore!
					// we don't have anything else to do here, break the poll
					log.WithFields(log.Fields{
						"namespace": kafkaUser.GetNamespace(),
						"name":      kafkaUser.GetName(),
					}).Warn("kafka user secret has been removed.")
					return true, ErrKafkaUserRemoved
				}

				// the object might have not been created yet
				log.WithFields(log.Fields{
					"namespace": kafkaUser.GetNamespace(),
					"name":      kafkaUser.GetName(),
				}).Debug("kafka user secret doesn't exist yet.")
				return false, nil
			}
			return false, err
		}

		seen = true
		readyCondition := getReadyCondition(k.Status.Conditions)
		if !strings.EqualFold(readyCondition.Status, "true") {
			log.WithFields(log.Fields{
				"namespace":       k.GetNamespace(),
				"name":            k.GetName(),
				"conditionStatus": readyCondition.Status,
				"conditionType":   readyCondition.Type,
			}).Debug("Waiting for kafka user to stabilize")
			return false, nil
		}

		log.WithFields(log.Fields{
			"namespace":       k.GetNamespace(),
			"name":            k.GetName(),
			"conditionStatus": readyCondition.Status,
			"conditionType":   readyCondition.Type,
		}).Debug("kafka user has stabilized")
		return true, nil
	})
}
