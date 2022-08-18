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

// ErrKafkaRemoved is returned when a kafka existed but has been removed
var ErrKafkaRemoved = errors.New("kafka has been removed")

func (r *ReconcileJaeger) applyKafkas(ctx context.Context, jaeger v1.Jaeger, desired []kafkav1beta2.Kafka) error {
	tracer := otel.GetTracerProvider().Tracer(v1.ReconciliationTracer)
	ctx, span := tracer.Start(ctx, "applyKafkas")
	defer span.End()

	opts := []client.ListOption{
		client.InNamespace(jaeger.Namespace),
		client.MatchingLabels(map[string]string{
			"app.kubernetes.io/instance":   jaeger.Name,
			"app.kubernetes.io/managed-by": "jaeger-operator",
		}),
	}
	list := &kafkav1beta2.KafkaList{}
	if err := r.rClient.List(ctx, list, opts...); err != nil {
		return tracing.HandleError(err, span)
	}

	inv := inventory.ForKafkas(list.Items, desired)
	for i := range inv.Create {
		d := inv.Create[i]
		jaeger.Logger().V(-1).Info(
			"creating kafkas",
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
			"updating kafkas",
			"kafka", d.GetName(),
			"namespace", d.GetNamespace(),
		)
		if err := r.client.Update(ctx, &d); err != nil {
			return tracing.HandleError(err, span)
		}
	}

	// now, wait for all Kafkas to estabilize
	for _, d := range inv.Create {
		// inv.Create has two objects at first: a Kafka and a KafkaUser object
		// right now, they both share the same name, so, it doesn't matter much that they are
		// different objects. A side effect is that we'll wait twice for the same objects, but that's also
		// not a big problem, as the second check will be fast, as the objects will exist already
		if err := r.waitForKafkaStability(ctx, d); err != nil {
			return tracing.HandleError(err, span)
		}
	}
	for _, d := range inv.Update {
		if err := r.waitForKafkaStability(ctx, d); err != nil {
			return tracing.HandleError(err, span)
		}
	}

	for i := range inv.Delete {
		d := inv.Delete[i]
		jaeger.Logger().V(-1).Info(
			"deleting kafka",
			"kafka", d.GetName(),
			"namespace", d.GetNamespace(),
		)
		if err := r.client.Delete(ctx, &d); err != nil {
			return tracing.HandleError(err, span)
		}
	}

	return nil
}

func (r *ReconcileJaeger) waitForKafkaStability(ctx context.Context, kafka kafkav1beta2.Kafka) error {
	tracer := otel.GetTracerProvider().Tracer(v1.ReconciliationTracer)
	ctx, span := tracer.Start(ctx, "waitForKafkaStability")
	defer span.End()

	seen := false
	once := &sync.Once{}
	return wait.PollImmediate(time.Second, 5*time.Minute, func() (done bool, err error) {
		k := &kafkav1beta2.Kafka{}
		if err := r.client.Get(ctx, types.NamespacedName{Name: kafka.GetName(), Namespace: kafka.GetNamespace()}, k); err != nil {
			if k8serrors.IsNotFound(err) {
				if seen {
					// we have seen this object before, but it doesn't exist anymore!
					// we don't have anything else to do here, break the poll
					log.Log.V(1).Info(
						"kafka has been removed.",
						"namespace", kafka.GetNamespace(),
						"name", kafka.GetName(),
					)
					return true, ErrKafkaRemoved
				}

				// the object might have not been created yet
				log.Log.V(-1).Info(
					"kafka doesn't exist yet.",
					"namespace", kafka.GetNamespace(),
					"name", kafka.GetName(),
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
					"Waiting for kafka to stabilize",
					"namespace", k.GetNamespace(),
					"name", k.GetName(),
					"conditionStatus", readyCondition.Status,
					"conditionType", readyCondition.Type,
				)
			})
			return false, nil
		}

		log.Log.V(-1).Info(
			"kafka has stabilized",
			"namespace", k.GetNamespace(),
			"name", k.GetName(),
			"conditionStatus", readyCondition.Status,
			"conditionType", readyCondition.Type,
		)
		return true, nil
	})
}

func getReadyCondition(conditions []kafkav1beta2.KafkaStatusCondition) kafkav1beta2.KafkaStatusCondition {
	for _, c := range conditions {
		if strings.EqualFold(c.Type, "ready") {
			return c
		}
	}

	return kafkav1beta2.KafkaStatusCondition{Type: "unknown", Status: "unknown"}
}
