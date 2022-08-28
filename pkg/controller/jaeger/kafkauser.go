package jaeger

import (
	"context"
	"errors"
	"strings"
	"sync"
	"time"

	"go.opentelemetry.io/otel"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/wait"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"

	v1 "github.com/jaegertracing/jaeger-operator/apis/v1"
	"github.com/jaegertracing/jaeger-operator/pkg/inventory"
	kafkav1beta2 "github.com/jaegertracing/jaeger-operator/pkg/kafka/v1beta2"
	"github.com/jaegertracing/jaeger-operator/pkg/tracing"
)

// ErrKafkaUserRemoved is returned when a kafka user existed but has been removed
var ErrKafkaUserRemoved = errors.New("kafka user has been removed")

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
	for i := range inv.Create {
		d := inv.Create[i]
		jaeger.Logger().V(-1).Info(
			"creating kafka users",
			"kafka", d.GetName(),
			"namespace", d.GetNamespace(),
		)
		if err := r.client.Create(ctx, &d); err != nil {
			return tracing.HandleError(err, span)
		}
	}

	for i := range inv.Update {
		d := inv.Update[i]
		jaeger.Logger().V(-1).Info(
			"updating kafka user",
			"kafka", d.GetName(),
			"namespace", d.GetNamespace(),
		)
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

	for i := range inv.Delete {
		d := inv.Delete[i]
		jaeger.Logger().V(-1).Info(
			"deleting kafka user",
			"kafka", d.GetName(),
			"namespace", d.GetNamespace(),
		)
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
					log.Log.V(1).Info(
						"kafka user secret has been removed.",
						"namespace", kafkaUser.GetNamespace(),
						"name", kafkaUser.GetName(),
					)
					return true, ErrKafkaUserRemoved
				}

				// the object might have not been created yet
				log.Log.V(-1).Info(
					"kafka user secret doesn't exist yet.",
					"namespace", kafkaUser.GetNamespace(),
					"name", kafkaUser.GetName(),
				)
				return false, nil
			}
			return false, tracing.HandleError(err, span)
		}

		seen = true
		readyCondition := getReadyCondition(k.Status.Conditions)
		if !strings.EqualFold(readyCondition.Status, "true") {
			once.Do(func() {
				log.Log.V(-1).Info(
					"Waiting for kafka user to stabilize",
					"namespace", k.GetNamespace(),
					"name", k.GetName(),
					"conditionStatus", readyCondition.Status,
					"conditionType", readyCondition.Type,
				)
			})
			return false, nil
		}

		log.Log.V(-1).Info(
			"kafka user has stabilized",
			"namespace", k.GetNamespace(),
			"name", k.GetName(),
			"conditionStatus", readyCondition.Status,
			"conditionType", readyCondition.Type,
		)
		return true, nil
	})
}
