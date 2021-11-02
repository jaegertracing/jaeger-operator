package jaeger

import (
	"context"
	"errors"
	"strings"
	"sync"
	"time"

	log "github.com/sirupsen/logrus"
	"go.opentelemetry.io/otel"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/wait"
	"sigs.k8s.io/controller-runtime/pkg/client"

	v1 "github.com/jaegertracing/jaeger-operator/pkg/apis/jaegertracing/v1"
	kafkav1beta2 "github.com/jaegertracing/jaeger-operator/pkg/apis/kafka/v1beta2"
	"github.com/jaegertracing/jaeger-operator/pkg/inventory"
	"github.com/jaegertracing/jaeger-operator/pkg/tracing"
)

var (
	// ErrKafkaUserRemoved is returned when a kafka user existed but has been removed
	ErrKafkaUserRemoved = errors.New("kafka user has been removed")
)

func (r *ReconcileJaeger) applyKafkaUsers(ctx context.Context, jaeger v1.Jaeger, desired []kafkav1beta2.KafkaUser) error {
	tracer := otel.GetTracerProvider().Tracer(v1.ReconciliationTracer)
	ctx, span := tracer.Start(ctx, "applyKafkaUsers")
	defer span.End()

	opts := []client.ListOption{
		client.InNamespace(jaeger.Namespace),
		client.MatchingLabels(map[string]string{
			"app.kubernetes.io/instance":   jaeger.Name,
			"app.kubernetes.io/managed-by": "jaeger-operator",
		}),
	}
	list := &kafkav1beta2.KafkaUserList{}
	if err := r.rClient.List(ctx, list, opts...); err != nil {
		return tracing.HandleError(err, span)
	}

	inv := inventory.ForKafkaUsers(list.Items, desired)
	for _, d := range inv.Create {
		jaeger.Logger().WithFields(log.Fields{
			"kafka":     d.GetName(),
			"namespace": d.GetNamespace(),
		}).Debug("creating kafka users")
		if err := r.client.Create(ctx, &d); err != nil {
			return tracing.HandleError(err, span)
		}
	}

	for _, d := range inv.Update {
		jaeger.Logger().WithFields(log.Fields{
			"kafka":     d.GetName(),
			"namespace": d.GetNamespace(),
		}).Debug("updating kafka user")
		if err := r.client.Update(ctx, &d); err != nil {
			return tracing.HandleError(err, span)
		}
	}

	// now, wait for all KafkaUsers to estabilize
	for _, d := range inv.Create {
		if err := r.waitForKafkaUserStability(ctx, d); err != nil {
			return tracing.HandleError(err, span)
		}
	}
	for _, d := range inv.Update {
		if err := r.waitForKafkaUserStability(ctx, d); err != nil {
			return tracing.HandleError(err, span)
		}
	}

	for _, d := range inv.Delete {
		jaeger.Logger().WithFields(log.Fields{
			"kafka":     d.GetName(),
			"namespace": d.GetNamespace(),
		}).Debug("deleting kafka user")
		if err := r.client.Delete(ctx, &d); err != nil {
			return tracing.HandleError(err, span)
		}
	}

	return nil
}

func (r *ReconcileJaeger) waitForKafkaUserStability(ctx context.Context, kafkaUser kafkav1beta2.KafkaUser) error {
	tracer := otel.GetTracerProvider().Tracer(v1.ReconciliationTracer)
	ctx, span := tracer.Start(ctx, "waitForKafkaUserStability")
	defer span.End()

	seen := false
	once := &sync.Once{}
	return wait.PollImmediate(time.Second, 5*time.Minute, func() (done bool, err error) {
		k := &kafkav1beta2.KafkaUser{}
		if err := r.client.Get(ctx, types.NamespacedName{Name: kafkaUser.GetName(), Namespace: kafkaUser.GetNamespace()}, k); err != nil {
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
			return false, tracing.HandleError(err, span)
		}

		seen = true
		readyCondition := getReadyCondition(k.Status.Conditions)
		if !strings.EqualFold(readyCondition.Status, "true") {
			once.Do(func() {
				log.WithFields(log.Fields{
					"namespace":       k.GetNamespace(),
					"name":            k.GetName(),
					"conditionStatus": readyCondition.Status,
					"conditionType":   readyCondition.Type,
				}).Debug("Waiting for kafka user to stabilize")
			})
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
